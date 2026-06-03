package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/daemon"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newDocsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "docs",
		Aliases: []string{"doc", "kb"},
		Short:   "Manage the knowledge base",
		Long: `Manage the knowledge base: source documents, synthesized wiki pages, and activity log.

The knowledge base has three layers (following the Karpathy LLM-wiki pattern):

  Raw sources  — documents you add via "docs add" or "docs ingest"; never modified
  Wiki         — LLM-generated markdown summaries in .agencycli/wiki/ (agent-owned)
  index.md     — auto-maintained catalog of wiki pages + sources for fast agent lookup
  log.md       — append-only activity timeline (ingest/query/lint events)

Typical agent workflow:
  agencycli docs ingest --path report.md --title "Q1 Report" --created-by human
  # → creates wiki stub, updates index.md and log.md; agent fills in the stub
  agencycli docs query "what are the main risks?"
  # → prints index.md + relevant wiki pages as context for the agent to answer`,
	}
	cmd.AddCommand(
		newDocsAddCmd(),
		newDocsIngestCmd(),
		newDocsListCmd(),
		newDocsTreeCmd(),
		newDocsShowCmd(),
		newDocsUpdateCmd(),
		newDocsMoveCmd(),
		newDocsRemoveCmd(),
		newDocsSearchCmd(),
		newDocsQueryCmd(),
		newDocsLintCmd(),
		newDocsWikiCmd(),
	)
	return cmd
}

func newDocsAddCmd() *cobra.Command {
	var (
		filePath    string
		title       string
		index       string
		createdBy   string
		tags        []string
		description string
	)
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a document to the knowledge base index",
		Long: `Add a document bookmark to the index. The file stays where it is;
only metadata is recorded.

  agencycli docs add --path ./docs/design.md --title "System Design" \
    --index "cc-connect/architecture" --created-by human

Virtual directories in --index are created automatically.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if filePath == "" {
				return fmt.Errorf("--path is required")
			}
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return err
			}
			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("file not found: %s", absPath)
			}
			if title == "" {
				title = strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
			}
			if createdBy == "" {
				return fmt.Errorf("--created-by is required (e.g. human, project/agent)")
			}
			index = strings.Trim(index, "/")

			ds := store.NewDocsStore(root)
			entry := &store.DocEntry{
				Title:       title,
				FilePath:    absPath,
				Index:       index,
				CreatedBy:   createdBy,
				Tags:        tags,
				Description: description,
			}
			if err := ds.Add(entry); err != nil {
				return err
			}
			fmt.Printf("✓ Document added: %s [%s]\n", entry.ID, index)
			fmt.Printf("  Title: %s\n  Path:  %s\n", title, absPath)
			fmt.Printf("  Web:   %s\n", docsWebPath(entry.ID))
			if webURL := docsWebURL(root, entry.ID); webURL != "" {
				fmt.Printf("  URL:   %s\n", webURL)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "path", "", "file path (required)")
	cmd.Flags().StringVar(&title, "title", "", "document title (default: filename)")
	cmd.Flags().StringVar(&index, "index", "", "virtual directory path (e.g. project/articles)")
	cmd.Flags().StringVar(&createdBy, "created-by", "", "who added this (required, e.g. human, project/agent)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "tags (repeatable)")
	cmd.Flags().StringVar(&description, "description", "", "short description")
	return cmd
}

func docsWebPath(docID string) string {
	return "/docs/" + url.PathEscape(docID)
}

func docsWebURL(root, docID string) string {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("AGENCYCLI_WEB_BASE_URL")), "/")
	if base != "" {
		return base + docsWebPath(docID)
	}
	if base := runningWebBaseURL(root); base != "" {
		return base + docsWebPath(docID)
	}
	if base := daemonWebBaseURL(root); base != "" {
		return base + docsWebPath(docID)
	}
	return ""
}

func runningWebBaseURL(root string) string {
	meta, err := daemon.LoadWebRuntimeMeta(root)
	if err != nil || meta.WorkDir != root || meta.Addr == "" {
		return ""
	}
	base := webBaseURLFromAddr(meta.Addr)
	if base == "" || !webHealthOK(base) {
		return ""
	}
	return base
}

func daemonWebBaseURL(root string) string {
	meta, err := daemon.LoadMeta()
	if err != nil || meta.WorkDir != root || meta.Addr == "" {
		return ""
	}
	base := webBaseURLFromAddr(meta.Addr)
	if base == "" || !webHealthOK(base) {
		return ""
	}
	return base
}

func webBaseURLFromAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return ""
	}
	host, port, ok := strings.Cut(addr, ":")
	if !ok {
		return "http://" + addr
	}
	switch strings.Trim(host, "[]") {
	case "", "0.0.0.0", "::":
		host = "127.0.0.1"
	}
	if strings.Contains(host, ":") && !strings.HasPrefix(host, "[") {
		host = "[" + host + "]"
	}
	return "http://" + host + ":" + port
}

func webHealthOK(base string) bool {
	client := &http.Client{Timeout: 500 * time.Millisecond}
	resp, err := client.Get(strings.TrimRight(base, "/") + "/api/v1/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}

func newDocsListCmd() *cobra.Command {
	var (
		index     string
		tag       string
		createdBy string
		asJSON    bool
	)
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List all indexed documents",
		Aliases: []string{"ls"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			docs, err := ds.List()
			if err != nil {
				return err
			}
			filtered := filterDocs(docs, index, tag, createdBy)
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(filtered)
			}
			if len(filtered) == 0 {
				fmt.Println("No documents found.")
				return nil
			}
			for _, d := range filtered {
				fmt.Printf("%-22s %-30s %s\n", d.ID, truncStr(d.Title, 28), d.Index)
			}
			fmt.Printf("\n%d document(s)\n", len(filtered))
			return nil
		},
	}
	cmd.Flags().StringVar(&index, "index", "", "filter by index prefix")
	cmd.Flags().StringVar(&tag, "tag", "", "filter by tag")
	cmd.Flags().StringVar(&createdBy, "created-by", "", "filter by creator")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newDocsTreeCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "tree",
		Short: "Show the virtual directory tree",
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			tree, err := ds.Tree()
			if err != nil {
				return err
			}
			if asJSON {
				return json.NewEncoder(os.Stdout).Encode(tree)
			}
			printTree(tree, "")
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newDocsShowCmd() *cobra.Command {
	var withContent bool
	cmd := &cobra.Command{
		Use:   "show <doc-id>",
		Short: "Show document details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			d, err := ds.Get(args[0])
			if err != nil {
				return err
			}
			fmt.Printf("ID         : %s\n", d.ID)
			fmt.Printf("Title      : %s\n", d.Title)
			fmt.Printf("Index      : %s\n", d.Index)
			fmt.Printf("File       : %s\n", d.FilePath)
			fmt.Printf("Created by : %s\n", d.CreatedBy)
			fmt.Printf("Created at : %s\n", d.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Updated at : %s\n", d.UpdatedAt.Format(time.RFC3339))
			if len(d.Tags) > 0 {
				fmt.Printf("Tags       : %s\n", strings.Join(d.Tags, ", "))
			}
			if d.Description != "" {
				fmt.Printf("Description: %s\n", d.Description)
			}
			if withContent {
				content, err := ds.ReadContent(d.FilePath)
				if err != nil {
					fmt.Fprintf(os.Stderr, "\nwarning: could not read file: %v\n", err)
				} else {
					fmt.Printf("\n--- content ---\n%s\n", content)
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&withContent, "content", false, "also print file content")
	return cmd
}

func newDocsUpdateCmd() *cobra.Command {
	var (
		title       string
		tags        []string
		description string
		index       string
	)
	cmd := &cobra.Command{
		Use:   "update <doc-id>",
		Short: "Update document metadata",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			return ds.Update(args[0], func(e *store.DocEntry) {
				if cmd.Flags().Changed("title") {
					e.Title = title
				}
				if cmd.Flags().Changed("index") {
					e.Index = strings.Trim(index, "/")
				}
				if cmd.Flags().Changed("tag") {
					e.Tags = tags
				}
				if cmd.Flags().Changed("description") {
					e.Description = description
				}
			})
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "new title")
	cmd.Flags().StringVar(&index, "index", "", "new virtual directory")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "replace tags")
	cmd.Flags().StringVar(&description, "description", "", "new description")
	return cmd
}

func newDocsMoveCmd() *cobra.Command {
	var index string
	cmd := &cobra.Command{
		Use:   "move <doc-id>",
		Short: "Move a document to a different virtual directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if index == "" {
				return fmt.Errorf("--index is required")
			}
			ds := store.NewDocsStore(root)
			if err := ds.Update(args[0], func(e *store.DocEntry) {
				e.Index = strings.Trim(index, "/")
			}); err != nil {
				return err
			}
			fmt.Printf("✓ Moved %s → %s\n", args[0], index)
			return nil
		},
	}
	cmd.Flags().StringVar(&index, "index", "", "target virtual directory (required)")
	return cmd
}

func newDocsRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "remove <doc-id>",
		Aliases: []string{"rm"},
		Short:   "Remove a document from the index (file is not deleted)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			if err := ds.Remove(args[0]); err != nil {
				return err
			}
			fmt.Printf("✓ Removed %s from index\n", args[0])
			return nil
		},
	}
	return cmd
}

func newDocsSearchCmd() *cobra.Command {
	var withContent bool
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search documents by title, description, tags, path, or content",
		Example: `  agencycli docs search "authentication"
  agencycli docs search "JWT token" --content    # also grep file contents
  agencycli docs search "oauth" --json`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")
			ds := store.NewDocsStore(root)
			results, err := ds.SearchOpts(query, withContent)
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(results)
			}
			if len(results) == 0 {
				fmt.Println("No results.")
				return nil
			}
			for _, d := range results {
				fmt.Printf("%-22s %-30s %s\n", d.ID, truncStr(d.Title, 28), d.Index)
			}
			fmt.Printf("\n%d result(s)\n", len(results))
			return nil
		},
	}
	cmd.Flags().BoolVar(&withContent, "content", false, "also search inside file contents")
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

// ── docs ingest ───────────────────────────────────────────────────────────────

func newDocsIngestCmd() *cobra.Command {
	var (
		filePath    string
		title       string
		index       string
		createdBy   string
		tags        []string
		description string
		noWiki      bool
	)
	cmd := &cobra.Command{
		Use:   "ingest",
		Short: "Register a source document and create a wiki stub for the agent to fill in",
		Long: `Register a source document in the index and create a wiki stub page in
.agencycli/wiki/. The stub contains template sections (Summary, Key Points,
Related Pages, Notes) that the agent should read the source and fill in.

index.md and log.md are updated automatically.

The agent's workflow after running this command:
  1. Read the source file at the path shown
  2. Read the wiki stub at the path shown
  3. Fill in the stub sections based on the source
  4. Run "agencycli docs wiki rebuild-index" after major additions`,
		Example: `  agencycli docs ingest --path ./docs/adr-001.md --title "ADR-001: Auth Strategy" \
    --index "architecture/decisions" --created-by human
  agencycli docs ingest --path report.pdf --title "Q1 Report" \
    --tag finance --tag q1-2026 --created-by human`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if filePath == "" {
				return fmt.Errorf("--path is required")
			}
			absPath, err := filepath.Abs(filePath)
			if err != nil {
				return err
			}
			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("file not found: %s", absPath)
			}
			if title == "" {
				title = strings.TrimSuffix(filepath.Base(absPath), filepath.Ext(absPath))
			}
			if createdBy == "" {
				return fmt.Errorf("--created-by is required (e.g. human, project/agent)")
			}
			index = strings.Trim(index, "/")

			ds := store.NewDocsStore(root)
			entry := &store.DocEntry{
				Title:       title,
				FilePath:    absPath,
				Index:       index,
				CreatedBy:   createdBy,
				Tags:        tags,
				Description: description,
			}
			if err := ds.Add(entry); err != nil {
				return err
			}

			// Create wiki stub
			slug := store.DocSlug(title)
			stubPath := ds.WikiPagePath(slug)
			var wikiMsg string
			if noWiki {
				wikiMsg = "(wiki stub skipped — use --no-wiki flag was set)"
			} else if _, err := os.Stat(stubPath); err == nil {
				wikiMsg = fmt.Sprintf("wiki stub already exists: %s", stubPath)
			} else {
				stub := store.WikiStub(entry)
				if err := ds.WriteWikiPage(slug, stub); err != nil {
					fmt.Fprintf(os.Stderr, "warning: could not write wiki stub: %v\n", err)
					wikiMsg = "(wiki stub creation failed)"
				} else {
					wikiMsg = stubPath
				}
			}

			// Update index.md and log.md
			_ = ds.RebuildWikiIndex()
			_ = ds.AppendWikiLog("ingest", title, fmt.Sprintf("  source: %s\n  doc-id: %s", absPath, entry.ID))

			fmt.Printf("✓ Ingested: %s [%s]\n", entry.ID, index)
			fmt.Printf("  Title     : %s\n", title)
			fmt.Printf("  Source    : %s\n", absPath)
			fmt.Printf("  Wiki stub : %s\n", wikiMsg)
			fmt.Printf("  Index     : %s\n", ds.WikiIndexPath())
			fmt.Printf("  Log       : %s\n", ds.WikiLogPath())
			fmt.Println()
			fmt.Println("Next step: read the source file and fill in the wiki stub sections.")
			return nil
		},
	}
	cmd.Flags().StringVar(&filePath, "path", "", "source file path (required)")
	cmd.Flags().StringVar(&title, "title", "", "document title (default: filename)")
	cmd.Flags().StringVar(&index, "index", "", "virtual directory path (e.g. project/articles)")
	cmd.Flags().StringVar(&createdBy, "created-by", "", "who added this (required)")
	cmd.Flags().StringArrayVar(&tags, "tag", nil, "tags (repeatable)")
	cmd.Flags().StringVar(&description, "description", "", "short description (shown in index.md)")
	cmd.Flags().BoolVar(&noWiki, "no-wiki", false, "skip creating a wiki stub page")
	return cmd
}

// ── docs query ────────────────────────────────────────────────────────────────

func newDocsQueryCmd() *cobra.Command {
	var (
		maxPages int
		noLog    bool
	)
	cmd := &cobra.Command{
		Use:   "query <question>",
		Short: "Print wiki context for answering a question (for agents to read and synthesize)",
		Long: `Prints a context document that an agent should read before answering a question.
The output includes:
  1. The full index.md (catalog of all wiki pages and sources)
  2. The contents of any wiki pages whose title/summary matches the question
  3. A reminder of the question

The agent reads this output, synthesizes an answer, and optionally files the
answer back as a new wiki page using "docs wiki write".

Example agent loop:
  context=$(agencycli docs query "what is the auth strategy?")
  # agent reads $context, writes answer, saves to wiki if valuable`,
		Example: `  agencycli docs query "what is the authentication strategy?"
  agencycli docs query "Q1 revenue" --max-pages 5
  agencycli docs query "deployment process" --no-log`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			question := strings.Join(args, " ")
			ds := store.NewDocsStore(root)

			var out strings.Builder
			out.WriteString("# Knowledge Base Context\n\n")
			out.WriteString(fmt.Sprintf("**Question:** %s\n\n", question))
			out.WriteString("---\n\n")

			// 1. Include index.md
			if indexContent, err := os.ReadFile(ds.WikiIndexPath()); err == nil {
				out.WriteString("## index.md\n\n")
				out.WriteString(string(indexContent))
				out.WriteString("\n---\n\n")
			} else {
				out.WriteString("*(index.md not found — run `agencycli docs wiki rebuild-index` to generate it)*\n\n")
			}

			// 2. Find relevant wiki pages
			pages, err := ds.ListWikiPages()
			if err == nil && len(pages) > 0 {
				q := strings.ToLower(question)
				words := strings.Fields(q)
				type scored struct {
					page  *store.WikiPage
					score int
				}
				var ranked []scored
				for _, p := range pages {
					score := 0
					haystack := strings.ToLower(p.Title + " " + p.Summary + " " + p.Slug)
					for _, w := range words {
						if len(w) > 2 && strings.Contains(haystack, w) {
							score++
						}
					}
					if score > 0 {
						ranked = append(ranked, scored{p, score})
					}
				}
				// Sort by score descending
				for i := 0; i < len(ranked)-1; i++ {
					for j := i + 1; j < len(ranked); j++ {
						if ranked[j].score > ranked[i].score {
							ranked[i], ranked[j] = ranked[j], ranked[i]
						}
					}
				}
				if maxPages > 0 && len(ranked) > maxPages {
					ranked = ranked[:maxPages]
				}
				if len(ranked) > 0 {
					out.WriteString("## Relevant Wiki Pages\n\n")
					for _, r := range ranked {
						content, err := ds.ReadWikiPage(r.page.Slug)
						if err != nil {
							continue
						}
						out.WriteString(fmt.Sprintf("### %s (`%s`)\n\n", r.page.Title, r.page.Slug))
						out.WriteString(content)
						out.WriteString("\n\n---\n\n")
					}
				}
			}

			out.WriteString("## Your Task\n\n")
			out.WriteString(fmt.Sprintf("Answer the question: **%s**\n\n", question))
			out.WriteString("If the answer is valuable for future reference, save it as a wiki page:\n")
			out.WriteString("  agencycli docs wiki write <slug> --title \"<title>\" --content-file <file>\n")

			fmt.Print(out.String())

			if !noLog {
				_ = ds.AppendWikiLog("query", question, "")
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&maxPages, "max-pages", 10, "maximum number of wiki pages to include")
	cmd.Flags().BoolVar(&noLog, "no-log", false, "do not append this query to log.md")
	return cmd
}

// ── docs lint ─────────────────────────────────────────────────────────────────

func newDocsLintCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Health-check the knowledge base wiki for issues",
		Long: `Scans the wiki for common maintenance issues:
  - Wiki pages not linked from index.md (orphans)
  - Source documents with no wiki summary page
  - Empty wiki stubs (< 50 bytes)
  - Source documents without a description

Run this periodically or after bulk ingests to keep the wiki healthy.`,
		Example: `  agencycli docs lint
  agencycli docs lint --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			result, err := ds.LintWiki()
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(result)
			}

			healthy := true
			fmt.Printf("Knowledge Base Health Check\n")
			fmt.Printf("  Sources    : %d\n", result.TotalDocs)
			fmt.Printf("  Wiki pages : %d\n\n", result.TotalWikiPages)

			printIssues := func(label string, items []string) {
				if len(items) == 0 {
					return
				}
				healthy = false
				fmt.Printf("⚠ %s (%d):\n", label, len(items))
				for _, s := range items {
					fmt.Printf("    - %s\n", s)
				}
				fmt.Println()
			}
			printIssues("Orphan wiki pages (not in index.md)", result.OrphanWikiPages)
			printIssues("Source docs without a wiki page", result.DocsWithoutWiki)
			printIssues("Empty wiki stubs (< 50 bytes)", result.EmptyWikiPages)
			printIssues("Source docs without description", result.DocsWithoutDesc)

			if healthy {
				fmt.Println("✓ No issues found.")
			}
			_ = ds.AppendWikiLog("lint", fmt.Sprintf("orphans=%d, no-wiki=%d, empty=%d, no-desc=%d",
				len(result.OrphanWikiPages), len(result.DocsWithoutWiki),
				len(result.EmptyWikiPages), len(result.DocsWithoutDesc)), "")
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

// ── docs wiki ─────────────────────────────────────────────────────────────────

func newDocsWikiCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wiki",
		Short: "Manage wiki pages (LLM-generated summaries)",
		Long: `Wiki pages are LLM-generated markdown files stored in .agencycli/wiki/.
They are the synthesized knowledge layer — agents write and maintain them.
Humans can read and edit them too, but the LLM should own the content.`,
	}
	cmd.AddCommand(
		newDocsWikiListCmd(),
		newDocsWikiShowCmd(),
		newDocsWikiWriteCmd(),
		newDocsWikiRebuildIndexCmd(),
	)
	return cmd
}

func newDocsWikiListCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List all wiki pages",
		Example: `  agencycli docs wiki list
  agencycli docs wiki list --json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			pages, err := ds.ListWikiPages()
			if err != nil {
				return err
			}
			if asJSON {
				return printJSON(pages)
			}
			if len(pages) == 0 {
				fmt.Println("No wiki pages. Use 'docs ingest' or 'docs wiki write' to create some.")
				return nil
			}
			for _, p := range pages {
				summary := truncStr(p.Summary, 50)
				if summary == "" {
					summary = "(empty)"
				}
				fmt.Printf("%-30s %s\n", p.Slug, summary)
			}
			fmt.Printf("\n%d wiki page(s) in %s\n", len(pages), ds.WikiDir())
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	return cmd
}

func newDocsWikiShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <slug>",
		Short: "Print a wiki page",
		Args:  cobra.ExactArgs(1),
		Example: `  agencycli docs wiki show auth-strategy
  agencycli docs wiki show q1-report`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			content, err := ds.ReadWikiPage(args[0])
			if err != nil {
				return fmt.Errorf("wiki page %q not found (use 'docs wiki list' to see available pages)", args[0])
			}
			fmt.Print(content)
			return nil
		},
	}
	return cmd
}

func newDocsWikiWriteCmd() *cobra.Command {
	var (
		title       string
		contentFile string
		content     string
		append_     bool
	)
	cmd := &cobra.Command{
		Use:   "write <slug>",
		Short: "Write or update a wiki page",
		Long: `Write content to a wiki page. The agent typically uses this after running
"docs query" to file a valuable answer back into the wiki for future reference.

After writing, run "docs wiki rebuild-index" to update index.md.`,
		Example: `  agencycli docs wiki write auth-strategy --title "Auth Strategy" --content-file /tmp/answer.md
  agencycli docs wiki write q1-summary --content "# Q1 Summary\n\nRevenue was..."
  agencycli docs wiki write auth-strategy --content-file /tmp/notes.md --append`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			slug := args[0]
			ds := store.NewDocsStore(root)

			var body string
			if contentFile != "" {
				data, err := os.ReadFile(contentFile)
				if err != nil {
					return fmt.Errorf("reading content file: %w", err)
				}
				body = string(data)
			} else if content != "" {
				body = content
			} else {
				return fmt.Errorf("either --content or --content-file is required")
			}

			if append_ {
				existing, _ := ds.ReadWikiPage(slug)
				body = existing + "\n" + body
			}

			if title != "" && !strings.HasPrefix(strings.TrimSpace(body), "# ") {
				body = "# " + title + "\n\n" + body
			}

			if err := ds.WriteWikiPage(slug, body); err != nil {
				return err
			}
			_ = ds.AppendWikiLog("write", slug, "  agent wrote wiki page")
			_ = ds.RebuildWikiIndex()

			fmt.Printf("✓ Wiki page written: %s\n", ds.WikiPagePath(slug))
			fmt.Printf("  Index updated: %s\n", ds.WikiIndexPath())
			return nil
		},
	}
	cmd.Flags().StringVar(&title, "title", "", "page title (prepended as H1 if content has none)")
	cmd.Flags().StringVar(&contentFile, "content-file", "", "read content from this file")
	cmd.Flags().StringVar(&content, "content", "", "page content (use --content-file for longer content)")
	cmd.Flags().BoolVar(&append_, "append", false, "append to existing content instead of replacing")
	return cmd
}

func newDocsWikiRebuildIndexCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rebuild-index",
		Short: "Rebuild index.md from current wiki pages and source documents",
		Example: `  agencycli docs wiki rebuild-index`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			ds := store.NewDocsStore(root)
			if err := ds.RebuildWikiIndex(); err != nil {
				return err
			}
			fmt.Printf("✓ Rebuilt: %s\n", ds.WikiIndexPath())
			return nil
		},
	}
	return cmd
}

// ── helpers ───────────────────────────────────────────────────────────────────

func filterDocs(docs []*store.DocEntry, index, tag, createdBy string) []*store.DocEntry {
	if index == "" && tag == "" && createdBy == "" {
		return docs
	}
	var out []*store.DocEntry
	for _, d := range docs {
		if index != "" && !strings.HasPrefix(d.Index, index) {
			continue
		}
		if createdBy != "" && d.CreatedBy != createdBy {
			continue
		}
		if tag != "" {
			has := false
			for _, t := range d.Tags {
				if strings.EqualFold(t, tag) {
					has = true
					break
				}
			}
			if !has {
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

func printTree(n *store.TreeNode, prefix string) {
	if n.Name != "/" {
		fmt.Printf("%s📁 %s\n", prefix, n.Name)
		prefix += "  "
	}
	for _, c := range n.Children {
		printTree(c, prefix)
	}
	for _, d := range n.Docs {
		fmt.Printf("%s📄 %s  (%s)\n", prefix, d.Title, d.ID)
	}
}
