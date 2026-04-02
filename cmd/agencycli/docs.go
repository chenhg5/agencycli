package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newDocsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "docs",
		Aliases: []string{"doc", "kb"},
		Short:   "Manage the knowledge base document index",
	}
	cmd.AddCommand(
		newDocsAddCmd(),
		newDocsListCmd(),
		newDocsTreeCmd(),
		newDocsShowCmd(),
		newDocsUpdateCmd(),
		newDocsMoveCmd(),
		newDocsRemoveCmd(),
		newDocsSearchCmd(),
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

func newDocsListCmd() *cobra.Command {
	var (
		index     string
		tag       string
		createdBy string
		asJSON    bool
	)
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all indexed documents",
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
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search documents by title, description, tags, or path",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			query := strings.Join(args, " ")
			ds := store.NewDocsStore(root)
			results, err := ds.Search(query)
			if err != nil {
				return err
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

