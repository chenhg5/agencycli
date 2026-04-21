package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type fileEntry struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	IsDir   bool      `json:"isDir"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"modTime"`
	Mime    string    `json:"mime,omitempty"`
}

func (s *Server) filesDir() string {
	return filepath.Join(s.root, ".agencycli", "files")
}

// resolveFilesPath returns the absolute path and whether it is safely inside filesDir.
func (s *Server) resolveFilesPath(sub string) (string, bool) {
	clean := filepath.Clean(sub)
	if clean == "." {
		clean = ""
	}
	full := filepath.Join(s.filesDir(), clean)
	return full, strings.HasPrefix(full, s.filesDir())
}

// GET /api/v1/files?path=...
func (s *Server) handleListFiles(w http.ResponseWriter, r *http.Request) {
	sub := r.URL.Query().Get("path")
	dir, ok := s.resolveFilesPath(sub)
	if !ok {
		s.jsonError(w, http.StatusBadRequest, "invalid path")
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(dir, 0o755)
			_ = json.NewEncoder(w).Encode([]fileEntry{})
			return
		}
		s.serverError(w, err)
		return
	}
	files := make([]fileEntry, 0, len(entries))
	for _, e := range entries {
		info, err := e.Info()
		if err != nil || info == nil {
			continue
		}
		fe := fileEntry{
			Name:    e.Name(),
			Path:    filepath.Join(sub, e.Name()),
			IsDir:   e.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
		}
		if !e.IsDir() {
			fe.Mime = mime.TypeByExtension(filepath.Ext(e.Name()))
		}
		files = append(files, fe)
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
	_ = json.NewEncoder(w).Encode(files)
}

// POST /api/v1/files/upload?path=...
func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		s.jsonError(w, http.StatusBadRequest, "parse form: "+err.Error())
		return
	}
	sub := r.URL.Query().Get("path")
	dir, ok := s.resolveFilesPath(sub)
	if !ok {
		s.jsonError(w, http.StatusBadRequest, "invalid path")
		return
	}
	_ = os.MkdirAll(dir, 0o755)
	uploaded := make([]fileEntry, 0)
	for _, fh := range r.MultipartForm.File["file"] {
		src, err := fh.Open()
		if err != nil {
			continue
		}
		dstPath := filepath.Join(dir, fh.Filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			src.Close()
			continue
		}
		_, _ = io.Copy(dst, src)
		src.Close()
		info, _ := dst.Stat()
		dst.Close()
		uploaded = append(uploaded, fileEntry{
			Name:    fh.Filename,
			Path:    filepath.Join(sub, fh.Filename),
			Size:    info.Size(),
			ModTime: info.ModTime().UTC(),
			Mime:    mime.TypeByExtension(filepath.Ext(fh.Filename)),
		})
	}
	_ = json.NewEncoder(w).Encode(uploaded)
}

// POST /api/v1/files/mkdir
func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid body")
		return
	}
	dir, ok := s.resolveFilesPath(body.Path)
	if !ok {
		s.jsonError(w, http.StatusBadRequest, "invalid path")
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"path": body.Path})
}

// GET /api/v1/files/content/{path...}
func (s *Server) handleFileContent(w http.ResponseWriter, r *http.Request) {
	sub := r.PathValue("path")
	fp, ok := s.resolveFilesPath(sub)
	if !ok {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	http.ServeFile(w, r, fp)
}

// POST /api/v1/files/move   {"from":"a/b.png","to":"folder"}
func (s *Server) handleMoveFile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.From == "" {
		s.jsonError(w, http.StatusBadRequest, "invalid body")
		return
	}
	srcAbs, ok1 := s.resolveFilesPath(body.From)
	if !ok1 {
		s.jsonError(w, http.StatusBadRequest, "invalid source path")
		return
	}
	srcInfo, err := os.Stat(srcAbs)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, "source not found")
		return
	}
	dstDir, ok2 := s.resolveFilesPath(body.To)
	if !ok2 {
		s.jsonError(w, http.StatusBadRequest, "invalid target path")
		return
	}
	dstInfo, err := os.Stat(dstDir)
	if err != nil || !dstInfo.IsDir() {
		s.jsonError(w, http.StatusBadRequest, "target is not a directory")
		return
	}
	dstAbs := filepath.Join(dstDir, srcInfo.Name())
	if _, err := os.Stat(dstAbs); err == nil {
		s.jsonError(w, http.StatusConflict, "target already exists")
		return
	}
	if err := os.Rename(srcAbs, dstAbs); err != nil {
		s.serverError(w, err)
		return
	}
	newRel := filepath.Join(body.To, srcInfo.Name())
	_ = json.NewEncoder(w).Encode(map[string]string{"path": newRel})
}

// DELETE /api/v1/files/{path...}
func (s *Server) handleDeleteFile(w http.ResponseWriter, r *http.Request) {
	sub := r.PathValue("path")
	fp, ok := s.resolveFilesPath(sub)
	if !ok {
		s.jsonError(w, http.StatusBadRequest, "invalid path")
		return
	}
	info, err := os.Stat(fp)
	if err != nil {
		s.jsonError(w, http.StatusNotFound, "not found")
		return
	}
	if info.IsDir() {
		err = os.RemoveAll(fp)
	} else {
		err = os.Remove(fp)
	}
	if err != nil {
		s.serverError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
