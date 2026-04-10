package api

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// System roles.
const (
	RoleAdmin  = "admin"
	RoleMember = "member"
)

// Project-level roles (ascending privilege).
const (
	ProjectRoleViewer   = "viewer"
	ProjectRoleOperator = "operator"
	ProjectRoleManager  = "manager"
)

type projectAccess struct {
	Project string `json:"project"`
	Role    string `json:"role"` // viewer | operator | manager
}

type userRecord struct {
	Username     string          `json:"username"`
	Hash         string          `json:"hash"`
	Role         string          `json:"role"` // admin | member
	DisplayName  string          `json:"displayName,omitempty"`
	Email        string          `json:"email,omitempty"`
	Avatar       string          `json:"avatar,omitempty"` // URL
	Phone        string          `json:"phone,omitempty"`
	Bio          string          `json:"bio,omitempty"`
	Projects     []projectAccess `json:"projects,omitempty"`
	LinkedAgents []string        `json:"linkedAgents,omitempty"`
	Disabled     bool            `json:"disabled,omitempty"`
	CreatedAt    string          `json:"createdAt,omitempty"`
}

type usersFile struct {
	Users     []userRecord `json:"users"`
	JWTSecret string       `json:"jwtSecret"`
}

type UserStore struct {
	mu   sync.RWMutex
	path string
	data usersFile
}

func newUserStore(workspaceRoot string) *UserStore {
	dir := filepath.Join(workspaceRoot, ".agencycli")
	_ = os.MkdirAll(dir, 0o755)
	s := &UserStore{path: filepath.Join(dir, "users.json")}
	s.load()
	return s
}

func (s *UserStore) load() {
	raw, err := os.ReadFile(s.path)
	if err != nil {
		s.initDefault()
		return
	}
	if err := json.Unmarshal(raw, &s.data); err != nil || len(s.data.Users) == 0 {
		s.initDefault()
		return
	}
	if s.data.JWTSecret == "" {
		s.data.JWTSecret = generateSecret()
		s.save()
	}
}

func (s *UserStore) initDefault() {
	hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	s.data = usersFile{
		Users: []userRecord{
			{Username: "admin", Hash: string(hash), Role: "admin"},
		},
		JWTSecret: generateSecret(),
	}
	s.save()
}

func (s *UserStore) save() {
	raw, _ := json.MarshalIndent(s.data, "", "  ")
	_ = os.WriteFile(s.path, raw, 0o600)
}

func generateSecret() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *UserStore) Authenticate(username, password string) *userRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data.Users {
		u := &s.data.Users[i]
		if u.Username == username {
			if u.Disabled {
				return nil
			}
			if bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)) == nil {
				return u
			}
			return nil
		}
	}
	return nil
}

func (s *UserStore) ListUsers() []userRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]userRecord, len(s.data.Users))
	copy(out, s.data.Users)
	return out
}

func (s *UserStore) CreateUser(username, password, role, displayName, email, avatar, phone, bio string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Username == username {
			return fmt.Errorf("user %q already exists", username)
		}
	}
	if role != RoleAdmin && role != RoleMember {
		role = RoleMember
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	s.data.Users = append(s.data.Users, userRecord{
		Username:    username,
		Hash:        string(hash),
		Role:        role,
		DisplayName: displayName,
		Email:       email,
		Avatar:      avatar,
		Phone:       phone,
		Bio:         bio,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	})
	s.save()
	return nil
}

func (s *UserStore) UpdateUser(username string, role, displayName, email, avatar, phone, bio *string, disabled *bool, projects []projectAccess, linkedAgents []string, newPassword *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		u := &s.data.Users[i]
		if u.Username != username {
			continue
		}
		if role != nil {
			u.Role = *role
		}
		if displayName != nil {
			u.DisplayName = *displayName
		}
		if email != nil {
			u.Email = *email
		}
		if avatar != nil {
			u.Avatar = *avatar
		}
		if phone != nil {
			u.Phone = *phone
		}
		if bio != nil {
			u.Bio = *bio
		}
		if disabled != nil {
			u.Disabled = *disabled
		}
		if projects != nil {
			u.Projects = projects
		}
		if linkedAgents != nil {
			u.LinkedAgents = linkedAgents
		}
		if newPassword != nil && *newPassword != "" {
			hash, err := bcrypt.GenerateFromPassword([]byte(*newPassword), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			u.Hash = string(hash)
		}
		s.save()
		return nil
	}
	return fmt.Errorf("user not found")
}

func (s *UserStore) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		if s.data.Users[i].Username == username {
			s.data.Users = append(s.data.Users[:i], s.data.Users[i+1:]...)
			s.save()
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

func (s *UserStore) HasProjectAccess(username, project string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.data.Users {
		if u.Username != username {
			continue
		}
		if u.Role == RoleAdmin {
			return ProjectRoleManager, true
		}
		for _, pa := range u.Projects {
			if pa.Project == project {
				return pa.Role, true
			}
		}
		for _, la := range u.LinkedAgents {
			parts := strings.SplitN(la, "/", 2)
			if len(parts) == 2 && parts[0] == project {
				return ProjectRoleOperator, true
			}
		}
		return "", false
	}
	return "", false
}

func projectRoleLevel(role string) int {
	switch role {
	case ProjectRoleViewer:
		return 1
	case ProjectRoleOperator:
		return 2
	case ProjectRoleManager:
		return 3
	default:
		return 0
	}
}

func (s *UserStore) ChangePassword(username, oldPass, newPass string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.data.Users {
		u := &s.data.Users[i]
		if u.Username == username {
			if bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(oldPass)) != nil {
				return fmt.Errorf("wrong old password")
			}
			hash, err := bcrypt.GenerateFromPassword([]byte(newPass), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			u.Hash = string(hash)
			s.save()
			return nil
		}
	}
	return fmt.Errorf("user not found")
}

func (s *UserStore) GetUser(username string) *userRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range s.data.Users {
		if s.data.Users[i].Username == username {
			u := s.data.Users[i]
			return &u
		}
	}
	return nil
}

func (s *UserStore) Secret() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.data.JWTSecret
}

// Simple JWT: header.payload.signature with HMAC-SHA256.

type jwtPayload struct {
	Sub string `json:"sub"`
	Exp int64  `json:"exp"`
	Iat int64  `json:"iat"`
}

func (s *UserStore) IssueToken(username string, dur time.Duration) string {
	now := time.Now()
	payload := jwtPayload{Sub: username, Exp: now.Add(dur).Unix(), Iat: now.Unix()}
	return s.signJWT(payload)
}

func (s *UserStore) ValidateToken(token string) (string, bool) {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return "", false
	}
	wantSig := s.hmacSign(parts[0] + "." + parts[1])
	if parts[2] != wantSig {
		return "", false
	}
	raw, err := base64Decode(parts[1])
	if err != nil {
		return "", false
	}
	var p jwtPayload
	if json.Unmarshal(raw, &p) != nil {
		return "", false
	}
	if time.Now().Unix() > p.Exp {
		return "", false
	}
	return p.Sub, true
}

func (s *UserStore) signJWT(p jwtPayload) string {
	header := base64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, _ := json.Marshal(p)
	payloadB64 := base64Encode(payload)
	sig := s.hmacSign(header + "." + payloadB64)
	return header + "." + payloadB64 + "." + sig
}

func (s *UserStore) hmacSign(msg string) string {
	mac := hmac.New(sha256.New, []byte(s.Secret()))
	mac.Write([]byte(msg))
	return base64Encode(mac.Sum(nil))
}

func base64Encode(data []byte) string {
	const enc = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	result := make([]byte, 0, (len(data)*4+2)/3)
	for i := 0; i < len(data); i += 3 {
		val := uint(data[i]) << 16
		if i+1 < len(data) {
			val |= uint(data[i+1]) << 8
		}
		if i+2 < len(data) {
			val |= uint(data[i+2])
		}
		result = append(result, enc[(val>>18)&0x3F])
		result = append(result, enc[(val>>12)&0x3F])
		if i+1 < len(data) {
			result = append(result, enc[(val>>6)&0x3F])
		}
		if i+2 < len(data) {
			result = append(result, enc[val&0x3F])
		}
	}
	return string(result)
}

func base64Decode(s string) ([]byte, error) {
	const dec = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	var lookup [256]byte
	for i := range lookup {
		lookup[i] = 0xFF
	}
	for i, c := range dec {
		lookup[c] = byte(i)
	}

	out := make([]byte, 0, len(s)*3/4)
	buf := make([]byte, 0, 4)
	for i := 0; i < len(s); i++ {
		v := lookup[s[i]]
		if v == 0xFF {
			continue
		}
		buf = append(buf, v)
		if len(buf) == 4 {
			out = append(out, byte(buf[0]<<2|buf[1]>>4))
			out = append(out, byte(buf[1]<<4|buf[2]>>2))
			out = append(out, byte(buf[2]<<6|buf[3]))
			buf = buf[:0]
		}
	}
	switch len(buf) {
	case 3:
		out = append(out, byte(buf[0]<<2|buf[1]>>4))
		out = append(out, byte(buf[1]<<4|buf[2]>>2))
	case 2:
		out = append(out, byte(buf[0]<<2|buf[1]>>4))
	}
	return out, nil
}

// HTTP handlers

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Password = strings.TrimSpace(body.Password)
	if body.Username == "" || body.Password == "" {
		s.jsonError(w, http.StatusBadRequest, "username and password required")
		return
	}

	user := s.users.Authenticate(body.Username, body.Password)
	if user == nil {
		s.jsonError(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token := s.users.IssueToken(user.Username, 7*24*time.Hour)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"token":        token,
		"username":     user.Username,
		"role":         user.Role,
		"displayName":  user.DisplayName,
		"email":        user.Email,
		"avatar":       user.Avatar,
		"projects":     user.Projects,
		"linkedAgents": user.LinkedAgents,
	})
}

func (s *Server) handleAuthMe(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(ctxUserKey).(string)
	user := s.users.GetUser(username)
	if user == nil {
		s.jsonError(w, http.StatusNotFound, "user not found")
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"username":     user.Username,
		"role":         user.Role,
		"displayName":  user.DisplayName,
		"email":        user.Email,
		"avatar":       user.Avatar,
		"projects":     user.Projects,
		"linkedAgents": user.LinkedAgents,
	})
}

// RBAC helpers

func (s *Server) currentUser(r *http.Request) *userRecord {
	username, _ := r.Context().Value(ctxUserKey).(string)
	if username == "" || username == "apikey" {
		return &userRecord{Username: username, Role: RoleAdmin}
	}
	u := s.users.GetUser(username)
	if u == nil {
		return &userRecord{Username: username, Role: RoleMember}
	}
	return u
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	u := s.currentUser(r)
	if u.Role != RoleAdmin {
		s.jsonError(w, http.StatusForbidden, "admin access required")
		return false
	}
	return true
}

func (s *Server) canAccessProject(r *http.Request, project string) bool {
	u := s.currentUser(r)
	if u.Role == RoleAdmin {
		return true
	}
	_, ok := s.users.HasProjectAccess(u.Username, project)
	return ok
}

// User management handlers

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	users := s.users.ListUsers()
	type safeUser struct {
		Username     string          `json:"username"`
		Role         string          `json:"role"`
		DisplayName  string          `json:"displayName,omitempty"`
		Email        string          `json:"email,omitempty"`
		Avatar       string          `json:"avatar,omitempty"`
		Phone        string          `json:"phone,omitempty"`
		Bio          string          `json:"bio,omitempty"`
		Projects     []projectAccess `json:"projects,omitempty"`
		LinkedAgents []string        `json:"linkedAgents,omitempty"`
		Disabled     bool            `json:"disabled,omitempty"`
		CreatedAt    string          `json:"createdAt,omitempty"`
	}
	out := make([]safeUser, len(users))
	for i, u := range users {
		out[i] = safeUser{
			Username:     u.Username,
			Role:         u.Role,
			DisplayName:  u.DisplayName,
			Email:        u.Email,
			Avatar:       u.Avatar,
			Phone:        u.Phone,
			Bio:          u.Bio,
			Projects:     u.Projects,
			LinkedAgents: u.LinkedAgents,
			Disabled:     u.Disabled,
			CreatedAt:    u.CreatedAt,
		}
	}
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	var body struct {
		Username    string `json:"username"`
		Password    string `json:"password"`
		Role        string `json:"role"`
		DisplayName string `json:"displayName"`
		Email       string `json:"email"`
		Avatar      string `json:"avatar"`
		Phone       string `json:"phone"`
		Bio         string `json:"bio"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	body.Username = strings.TrimSpace(body.Username)
	body.Password = strings.TrimSpace(body.Password)
	if body.Username == "" || body.Password == "" {
		s.jsonError(w, http.StatusBadRequest, "username and password required")
		return
	}
	if len(body.Password) < 6 {
		s.jsonError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	if err := s.users.CreateUser(body.Username, body.Password, body.Role, body.DisplayName, body.Email, body.Avatar, body.Phone, body.Bio); err != nil {
		if strings.Contains(err.Error(), "already exists") {
			s.jsonError(w, http.StatusConflict, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleGetUser(w http.ResponseWriter, r *http.Request) {
	target := r.PathValue("username")
	cur := s.currentUser(r)
	if cur.Role != RoleAdmin && cur.Username != target {
		s.jsonError(w, http.StatusForbidden, "access denied")
		return
	}
	u := s.users.GetUser(target)
	if u == nil {
		s.jsonError(w, http.StatusNotFound, "user not found")
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"username":     u.Username,
		"role":         u.Role,
		"displayName":  u.DisplayName,
		"email":        u.Email,
		"avatar":       u.Avatar,
		"phone":        u.Phone,
		"bio":          u.Bio,
		"projects":     u.Projects,
		"linkedAgents": u.LinkedAgents,
		"disabled":     u.Disabled,
		"createdAt":    u.CreatedAt,
	})
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	target := r.PathValue("username")
	var body struct {
		Role         *string         `json:"role"`
		DisplayName  *string         `json:"displayName"`
		Email        *string         `json:"email"`
		Avatar       *string         `json:"avatar"`
		Phone        *string         `json:"phone"`
		Bio          *string         `json:"bio"`
		Disabled     *bool           `json:"disabled"`
		Password     *string         `json:"password"`
		Projects     []projectAccess `json:"projects"`
		LinkedAgents []string        `json:"linkedAgents"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Password != nil && len(*body.Password) > 0 && len(*body.Password) < 6 {
		s.jsonError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	if err := s.users.UpdateUser(target, body.Role, body.DisplayName, body.Email, body.Avatar, body.Phone, body.Bio, body.Disabled, body.Projects, body.LinkedAgents, body.Password); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	if !s.requireAdmin(w, r) {
		return
	}
	target := r.PathValue("username")
	cur := s.currentUser(r)
	if target == cur.Username {
		s.jsonError(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}
	if err := s.users.DeleteUser(target); err != nil {
		if strings.Contains(err.Error(), "not found") {
			s.jsonError(w, http.StatusNotFound, err.Error())
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value(ctxUserKey).(string)
	var body struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if err := s.readJSON(w, r, &body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.NewPassword) < 6 {
		s.jsonError(w, http.StatusBadRequest, "password must be at least 6 characters")
		return
	}
	if err := s.users.ChangePassword(username, body.OldPassword, body.NewPassword); err != nil {
		if strings.Contains(err.Error(), "wrong old password") {
			s.jsonError(w, http.StatusForbidden, "wrong old password")
			return
		}
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}
