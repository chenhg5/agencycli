package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	tmpl "github.com/chenhg5/agencycli/internal/template"
	"github.com/spf13/cobra"
)

func newTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "Pack and inspect agency templates",
		Long: `Templates let you share and reuse agency configurations.

A template is a .tar.gz archive containing:
  template.json     — metadata (name, version, author, description…)
  agency-prompt.md  — top-level agency prompt
  teams/            — team prompts and role definitions
  skills/           — reusable skill files

Runtime state (projects, agents, tasks) is never included.

Pack your agency as a template:
  agencycli template pack --output my-agency.tar.gz

Inspect a template:
  agencycli template info my-agency.tar.gz

Create an agency from a template:
  agencycli create agency --name "My Agency" --template my-agency.tar.gz
  agencycli create agency --name "My Agency" --template https://example.com/template.tar.gz`,
	}
	cmd.AddCommand(
		newTemplatePackCmd(),
		newTemplateInfoCmd(),
	)
	return cmd
}

// ── template pack ─────────────────────────────────────────────────────────────

func newTemplatePackCmd() *cobra.Command {
	var (
		agencyDir   string
		outputPath  string
		name        string
		version     string
		description string
		author      string
		email       string
		homepage    string
		license     string
		keywords    []string
	)

	cmd := &cobra.Command{
		Use:   "pack",
		Short: "Pack the current agency into a shareable template archive",
		Example: `  # Pack the current agency (auto-detect name from agency.yaml)
  agencycli template pack --output my-agency-template.tar.gz

  # Pack with full metadata
  agencycli template pack --output tech-project.tar.gz \
    --name "tech-project" --version "1.0.0" \
    --description "Standard software engineering agency" \
    --author "Alice" --email "alice@example.com" \
    --homepage "https://github.com/alice/tech-project-template" \
    --keywords "engineering,software,go"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Resolve agency dir.
			if agencyDir == "" {
				root, err := resolveRoot()
				if err != nil {
					return fmt.Errorf("not inside an agencycli workspace; use --dir or run from the agency root: %w", err)
				}
				agencyDir = root
			}

			// Load agency.yaml to fill in defaults.
			agencyYAML := filepath.Join(agencyDir, ".agencycli", "agency.yaml")
			if name == "" {
				if n := readAgencyName(agencyYAML); n != "" {
					name = n
				} else {
					name = filepath.Base(agencyDir)
				}
			}
			if version == "" {
				version = "1.0.0"
			}

			// Default output filename.
			if outputPath == "" {
				slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
				outputPath = slug + ".tar.gz"
			}

			// Build manifest.
			manifest := &entity.TemplateManifest{
				Name:        name,
				Version:     version,
				Description: description,
				Author:      author,
				Email:       email,
				Homepage:    homepage,
				License:     license,
				Keywords:    splitKeywords(keywords),
				CreatedAt:   time.Now().UTC().Format(time.RFC3339),
			}

			absOut, err := filepath.Abs(outputPath)
			if err != nil {
				return err
			}

			if err := tmpl.Pack(agencyDir, absOut, manifest); err != nil {
				return err
			}

			info, _ := os.Stat(absOut)
			size := int64(0)
			if info != nil {
				size = info.Size()
			}

			fmt.Printf("✓ Template packed: %s  (%.1f KB)\n", absOut, float64(size)/1024)
			fmt.Printf("\n  name       : %s\n", manifest.Name)
			fmt.Printf("  version    : %s\n", manifest.Version)
			if manifest.Description != "" {
				fmt.Printf("  description: %s\n", manifest.Description)
			}
			if manifest.Author != "" {
				fmt.Printf("  author     : %s", manifest.Author)
				if manifest.Email != "" {
					fmt.Printf(" <%s>", manifest.Email)
				}
				fmt.Println()
			}
			if manifest.Homepage != "" {
				fmt.Printf("  homepage   : %s\n", manifest.Homepage)
			}

			fmt.Printf("\nShare it:\n")
			fmt.Printf("  Upload %s to GitHub releases or any URL\n", filepath.Base(absOut))
			fmt.Printf("  Others can use it with:\n")
			fmt.Printf("    agencycli create agency --name \"My Agency\" --template %s\n", filepath.Base(absOut))
			return nil
		},
	}

	cmd.Flags().StringVar(&agencyDir, "dir", "", "agency workspace directory (default: auto-detect from CWD)")
	cmd.Flags().StringVar(&outputPath, "output", "", "output file path (default: <name>.tar.gz)")
	cmd.Flags().StringVar(&name, "name", "", "template name (default: from agency.yaml)")
	cmd.Flags().StringVar(&version, "version", "", "version string, e.g. 1.0.0 (default: 1.0.0)")
	cmd.Flags().StringVar(&description, "description", "", "short description of what this template provides")
	cmd.Flags().StringVar(&author, "author", "", "author name")
	cmd.Flags().StringVar(&email, "email", "", "author email")
	cmd.Flags().StringVar(&homepage, "homepage", "", "URL to documentation or source repo")
	cmd.Flags().StringVar(&license, "license", "", "SPDX license identifier, e.g. MIT")
	cmd.Flags().StringArrayVar(&keywords, "keywords", nil, "comma-separated keywords (repeatable)")
	return cmd
}

// ── template info ─────────────────────────────────────────────────────────────

func newTemplateInfoCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "info <template.tar.gz|directory>",
		Short: "Show metadata from a template archive or directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			src := args[0]

			var (
				m   *entity.TemplateManifest
				err error
			)

			info, statErr := os.Stat(src)
			if statErr != nil {
				return fmt.Errorf("cannot access %q: %w", src, statErr)
			}
			if info.IsDir() {
				m, err = tmpl.ReadManifestFromDir(src)
			} else {
				m, err = tmpl.ReadManifestFromArchive(src)
			}
			if err != nil {
				return fmt.Errorf("read template.json: %w", err)
			}

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(m)
			}

			fmt.Printf("Template: %s  v%s\n", m.Name, m.Version)
			if m.Description != "" {
				fmt.Printf("  %s\n", m.Description)
			}
			fmt.Println()
			if m.Author != "" {
				line := "  author  : " + m.Author
				if m.Email != "" {
					line += " <" + m.Email + ">"
				}
				fmt.Println(line)
			}
			if m.Homepage != "" {
				fmt.Printf("  homepage: %s\n", m.Homepage)
			}
			if m.License != "" {
				fmt.Printf("  license : %s\n", m.License)
			}
			if len(m.Keywords) > 0 {
				fmt.Printf("  keywords: %s\n", strings.Join(m.Keywords, ", "))
			}
			if m.CreatedAt != "" {
				fmt.Printf("  created : %s\n", m.CreatedAt)
			}
			fmt.Printf("\nUse it:\n")
			fmt.Printf("  agencycli create agency --name \"My Agency\" --template %s\n", src)
			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "output raw JSON")
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

func readAgencyName(yamlPath string) string {
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return ""
	}
	// Simple key scan — avoid pulling in yaml just for one field.
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		}
	}
	return ""
}

// splitKeywords flattens ["a,b", "c"] → ["a", "b", "c"].
func splitKeywords(raw []string) []string {
	var out []string
	for _, v := range raw {
		for _, kw := range strings.Split(v, ",") {
			if kw = strings.TrimSpace(kw); kw != "" {
				out = append(out, kw)
			}
		}
	}
	return out
}
