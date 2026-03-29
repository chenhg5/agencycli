package main

import (
	"fmt"
	"io/fs"
	"log"
	"mime"
	"net/http"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/chenhg5/agencycli/internal/api"
	"github.com/chenhg5/agencycli/web"
	"github.com/spf13/cobra"
)

func newStartCmd() *cobra.Command {
	var (
		addr   string
		apiKey string
		open   bool
	)

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start the web console (API + embedded frontend)",
		Long: `Launches a single HTTP server that serves both the JSON API and the
built-in web console on the same port. The frontend is embedded in the
binary at build time (web/dist).

This is the recommended way to run agencycli in production or on a
remote server. For local development with hot-reload, use
'agencycli api serve' combined with 'cd web && pnpm dev'.`,
		Example: `  # Default: listen on 127.0.0.1:27892
  agencycli start

  # Custom address
  agencycli start --addr 0.0.0.0:8080

  # With API key auth
  agencycli start --api-key my-secret

  # Auto-open browser
  agencycli start --open`,
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			key := api.ResolveAPIKey(apiKey)
			srv := api.NewServer(root, key)
			srv.SetVersion(version)

			handler := newSPAHandler(srv.Handler())

			url := fmt.Sprintf("http://%s", addr)
			log.Printf("agencycli web console: %s (workspace %s)", url, root)
			if key != "" {
				log.Printf("API key auth enabled")
			}
			if open {
				go openBrowser(url)
			}
			if err := http.ListenAndServe(addr, handler); err != nil {
				return fmt.Errorf("http server: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:27892", "listen address (host:port)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "optional Bearer token (or AGENCYCLI_WEB_API_KEY)")
	cmd.Flags().BoolVar(&open, "open", false, "open the web console in default browser")
	return cmd
}

// newSPAHandler wraps the API handler with an SPA file server.
// /api/ requests are forwarded to the API handler; everything else is served
// from the embedded web/dist with SPA fallback to index.html.
func newSPAHandler(apiHandler http.Handler) http.Handler {
	distFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		log.Fatalf("embedded web assets not found: %v", err)
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			apiHandler.ServeHTTP(w, r)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		f, err := distFS.Open(path)
		if err != nil {
			serveEmbeddedFile(w, distFS, "index.html")
			return
		}
		f.Close()
		serveEmbeddedFile(w, distFS, path)
	})
}

func serveEmbeddedFile(w http.ResponseWriter, fsys fs.FS, name string) {
	data, err := fs.ReadFile(fsys, name)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	ct := mime.TypeByExtension(filepath.Ext(name))
	if ct == "" {
		ct = http.DetectContentType(data)
	}
	w.Header().Set("Content-Type", ct)

	if strings.HasPrefix(name, "assets/") {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	w.Write(data)
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd.exe", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
