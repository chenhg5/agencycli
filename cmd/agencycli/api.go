package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/chenhg5/agencycli/internal/api"
	"github.com/spf13/cobra"
)

func newAPICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "api",
		Short: "HTTP JSON API for the web UI and integrations",
	}
	cmd.AddCommand(newAPIServeCmd())
	return cmd
}

func newAPIServeCmd() *cobra.Command {
	var (
		addr   string
		apiKey string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Start read-only HTTP server for the current workspace",
		Long: `Serves JSON under /api/v1/ for the resolved agency workspace.

Typical local use with the Vite dev server (see web/vite.config.ts proxy):

  agencycli --dir /path/to/MyAgency api serve
  cd web && pnpm dev

Read: GET health, agency, stats, teams, projects, tasks, messages, inbox, …
Write: POST .../tasks; POST /api/v1/messages; mark-read, archive, mark-all-read;
  POST .../projects/{name}/messages/mark-all-read; GET .../messages?includeArchived=1

Optional auth: set --api-key or AGENCYCLI_WEB_API_KEY; clients must send
Authorization: Bearer <key>.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			key := api.ResolveAPIKey(apiKey)
			srv := api.NewServer(root, key)
			srv.SetVersion(version)
			srv.SetUpdateChecker(GetCachedUpdateInfo)
			srv.SetDaemonStatus(daemonStatusJSON)
			log.Printf("agencycli api listening on http://%s (workspace %s)", addr, root)
			if key != "" {
				log.Printf("API key auth enabled (set AGENCYCLI_WEB_API_KEY or --api-key)")
			}

			httpSrv := &http.Server{Addr: addr, Handler: srv.Handler()}

			quit := make(chan os.Signal, 1)
			signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-quit
				log.Println("shutting down — stopping schedulers…")
				srv.Shutdown()
				_ = httpSrv.Shutdown(context.Background())
			}()

			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("http server: %w", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&addr, "addr", "127.0.0.1:27892", "listen address (host:port)")
	cmd.Flags().StringVar(&apiKey, "api-key", "", "optional Bearer token (or AGENCYCLI_WEB_API_KEY)")
	return cmd
}
