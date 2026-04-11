package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/store"
)

// ──────────────────────────────────────────────
// Config endpoints
// ──────────────────────────────────────────────

func (s *Server) handleCCConnectGetConfig(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.ccStore.Load()
	if err != nil {
		s.serverError(w, err)
		return
	}
	hasToken := cfg.Token != ""
	_ = json.NewEncoder(w).Encode(map[string]any{
		"apiUrl":   cfg.APIURL,
		"hasToken": hasToken,
	})
}

func (s *Server) handleCCConnectPutConfig(w http.ResponseWriter, r *http.Request) {
	var body struct {
		APIURL string `json:"apiUrl"`
		Token  string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		s.jsonError(w, http.StatusBadRequest, "invalid json")
		return
	}
	body.APIURL = strings.TrimRight(strings.TrimSpace(body.APIURL), "/")
	if body.APIURL == "" {
		s.jsonError(w, http.StatusBadRequest, "apiUrl is required")
		return
	}

	existing, _ := s.ccStore.Load()
	cfg := &store.CCConnectConfig{APIURL: body.APIURL, Token: body.Token}
	if cfg.Token == "" && existing != nil {
		cfg.Token = existing.Token
	}
	if err := s.ccStore.Save(cfg); err != nil {
		s.serverError(w, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) handleCCConnectTest(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.ccStore.Load()
	if err != nil || cfg.APIURL == "" {
		s.jsonError(w, http.StatusBadRequest, "cc-connect not configured")
		return
	}
	resp, err := s.ccProxy(cfg, "GET", "/status", nil)
	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": string(body)})
		return
	}
	// cc-connect wraps in {"data": ..., "ok": true}
	var wrapped struct {
		Data map[string]any `json:"data"`
	}
	if json.Unmarshal(body, &wrapped) == nil && wrapped.Data != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": wrapped.Data})
		return
	}
	var data map[string]any
	_ = json.Unmarshal(body, &data)
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "data": data})
}

// ──────────────────────────────────────────────
// Generic proxy to cc-connect API
// ──────────────────────────────────────────────

var ccHTTPClient = &http.Client{Timeout: 30 * time.Second}

func (s *Server) ccProxy(cfg *store.CCConnectConfig, method, path string, reqBody io.Reader) (*http.Response, error) {
	url := cfg.APIURL + "/api/v1" + path
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.Token)
	}
	return ccHTTPClient.Do(req)
}

func (s *Server) ccProxyForward(w http.ResponseWriter, r *http.Request, method, path string) {
	cfg, err := s.ccStore.Load()
	if err != nil || cfg.APIURL == "" {
		s.jsonError(w, http.StatusBadRequest, "cc-connect not configured")
		return
	}

	var reqBody io.Reader
	if r.Body != nil {
		data, err := io.ReadAll(r.Body)
		if err == nil && len(data) > 0 {
			reqBody = bytes.NewReader(data)
		}
	}

	resp, err := s.ccProxy(cfg, method, path, reqBody)
	if err != nil {
		s.jsonError(w, http.StatusBadGateway, fmt.Sprintf("cc-connect unreachable: %v", err))
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	// cc-connect wraps successful responses in {"data": ..., "ok": true}.
	// Unwrap so the frontend receives the inner data directly.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var wrapped struct {
			Data json.RawMessage `json:"data"`
			OK   bool           `json:"ok"`
		}
		if json.Unmarshal(body, &wrapped) == nil && wrapped.OK && len(wrapped.Data) > 0 {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(resp.StatusCode)
			_, _ = w.Write(wrapped.Data)
			return
		}
	}

	for k, vv := range resp.Header {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = w.Write(body)
}

// ──────────────────────────────────────────────
// Project / platform proxy handlers
// ──────────────────────────────────────────────

func (s *Server) handleCCProxyProjects(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "GET", "/projects")
}

func (s *Server) handleCCProxyProjectDetail(w http.ResponseWriter, r *http.Request) {
	name := url.PathEscape(r.PathValue("name"))
	s.ccProxyForward(w, r, "GET", "/projects/"+name)
}

func (s *Server) handleCCProxyProjectDelete(w http.ResponseWriter, r *http.Request) {
	name := url.PathEscape(r.PathValue("name"))
	s.ccProxyForward(w, r, "DELETE", "/projects/"+name)
}

func (s *Server) handleCCProxyAddPlatform(w http.ResponseWriter, r *http.Request) {
	name := url.PathEscape(r.PathValue("name"))
	s.ccProxyForward(w, r, "POST", "/projects/"+name+"/add-platform")
}

func (s *Server) handleCCProxyReload(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/reload")
}

func (s *Server) handleCCProxyRestart(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/restart")
}

// ──────────────────────────────────────────────
// Setup (QR scan) proxy handlers
// ──────────────────────────────────────────────

func (s *Server) handleCCProxySetupFeishuBegin(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/setup/feishu/begin")
}

func (s *Server) handleCCProxySetupFeishuPoll(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/setup/feishu/poll")
}

func (s *Server) handleCCProxySetupFeishuSave(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/setup/feishu/save")
}

func (s *Server) handleCCProxySetupWeixinBegin(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/setup/weixin/begin")
}

func (s *Server) handleCCProxySetupWeixinPoll(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/setup/weixin/poll")
}

func (s *Server) handleCCProxySetupWeixinSave(w http.ResponseWriter, r *http.Request) {
	s.ccProxyForward(w, r, "POST", "/setup/weixin/save")
}

// ──────────────────────────────────────────────
// Sessions proxy
// ──────────────────────────────────────────────

func (s *Server) handleCCProxyProjectSessions(w http.ResponseWriter, r *http.Request) {
	name := url.PathEscape(r.PathValue("name"))
	s.ccProxyForward(w, r, "GET", "/projects/"+name+"/sessions")
}
