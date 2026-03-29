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

type userRecord struct {
	Username string `json:"username"`
	Hash     string `json:"hash"`
	Role     string `json:"role"`
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
			if bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(password)) == nil {
				return u
			}
			return nil
		}
	}
	return nil
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
		"token":    token,
		"username": user.Username,
		"role":     user.Role,
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
		"username": user.Username,
		"role":     user.Role,
	})
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
