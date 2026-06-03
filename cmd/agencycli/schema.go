package main

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// schemaFlag describes one flag of a command.
type schemaFlag struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Default  string `json:"default,omitempty"`
	Usage    string `json:"usage"`
	Required bool   `json:"required,omitempty"`
}

// schemaCommand is the JSON representation of one cobra command.
type schemaCommand struct {
	Command     string          `json:"command"`
	Use         string          `json:"use"`
	Short       string          `json:"short"`
	Long        string          `json:"long,omitempty"`
	Examples    []string        `json:"examples,omitempty"`
	Flags       []schemaFlag    `json:"flags"`
	Subcommands []schemaCommand `json:"subcommands,omitempty"`
}

func newSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema [command...]",
		Short: "Print the command schema as JSON (for agent self-discovery)",
		Long: `schema emits the full command tree — or a single command — as JSON.

Agents can use this to discover available commands, required flags, and
flag types without parsing --help text.

Examples:
  agencycli schema                  # full command tree
  agencycli schema task             # subcommands of "task"
  agencycli schema task add         # flags for "task add"
  agencycli schema inbox send`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := rootCmd
			for _, segment := range args {
				found := false
				for _, sub := range target.Commands() {
					if sub.Name() == segment {
						target = sub
						found = true
						break
					}
				}
				if !found {
					return printJSON(map[string]string{
						"error": "command not found: " + strings.Join(args, " "),
					})
				}
			}
			return printJSON(buildSchema(target, args))
		},
	}
	return cmd
}

func buildSchema(cmd *cobra.Command, path []string) schemaCommand {
	fullPath := strings.Join(path, " ")
	if fullPath == "" {
		fullPath = cmd.Name()
	}

	s := schemaCommand{
		Command: fullPath,
		Use:     cmd.Use,
		Short:   cmd.Short,
		Flags:   []schemaFlag{},
	}
	if cmd.Long != "" && cmd.Long != cmd.Short {
		s.Long = cmd.Long
	}
	if cmd.Example != "" {
		for _, line := range strings.Split(cmd.Example, "\n") {
			line = strings.TrimSpace(line)
			if line != "" {
				s.Examples = append(s.Examples, line)
			}
		}
	}

	// Collect local flags (not inherited globals).
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		sf := schemaFlag{
			Name:    f.Name,
			Type:    f.Value.Type(),
			Default: f.DefValue,
			Usage:   f.Usage,
		}
		// cobra marks required flags with an annotation.
		if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			sf.Required = true
		}
		s.Flags = append(s.Flags, sf)
	})

	for _, sub := range cmd.Commands() {
		if sub.Hidden {
			continue
		}
		subPath := append(append([]string{}, path...), sub.Name())
		s.Subcommands = append(s.Subcommands, buildSchema(sub, subPath))
	}

	return s
}
