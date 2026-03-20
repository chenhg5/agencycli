package main

import (
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newRoleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "role",
		Short: "Manage role definitions (skills, setup)",
	}
	cmd.AddCommand(
		newRoleSkillCmd(),
		newRoleListCmd(),
	)
	return cmd
}

// ── role skill ────────────────────────────────────────────────────────────────

func newRoleSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Add or remove skills bound to a role",
	}
	cmd.AddCommand(
		newRoleSkillAddCmd(),
		newRoleSkillRemoveCmd(),
	)
	return cmd
}

func newRoleSkillAddCmd() *cobra.Command {
	var team, roleName string
	var skills []string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Bind one or more skills to a role",
		Example: `  agencycli role skill add --team growth --role content-writer --skill article-publisher
  agencycli role skill add --team engineering --role go-developer --skill github-push-relay,docker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || roleName == "" || len(skills) == 0 {
				return fmt.Errorf("--team, --role and --skill are all required")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			r, err := s.Role(team, roleName)
			if err != nil {
				return err
			}

			added := 0
			existing := make(map[string]bool, len(r.Skills))
			for _, sk := range r.Skills {
				existing[sk] = true
			}
			for _, sk := range skills {
				sk = strings.TrimSpace(sk)
				if sk == "" || existing[sk] {
					continue
				}
				r.Skills = append(r.Skills, sk)
				existing[sk] = true
				added++
			}

			if added == 0 {
				fmt.Printf("ℹ  No new skills to add (all already bound to role %q/%q).\n", team, roleName)
				return nil
			}

			if err := s.SaveRole(team, roleName, r); err != nil {
				return err
			}

			fmt.Printf("✓ Added %d skill(s) to role %q/%q\n", added, team, roleName)
			fmt.Printf("  Skills now: %s\n", strings.Join(r.Skills, ", "))
			fmt.Printf("\n  Run `agencycli sync` to push the updated context to hired agents.\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Team path, e.g. \"growth\"")
	cmd.Flags().StringVar(&roleName, "role", "", "Role name, e.g. \"content-writer\"")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "Skill name(s) to add (comma-separated or repeated flag)")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("role")
	_ = cmd.MarkFlagRequired("skill")
	return cmd
}

func newRoleSkillRemoveCmd() *cobra.Command {
	var team, roleName string
	var skills []string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Unbind one or more skills from a role",
		Example: `  agencycli role skill remove --team growth --role content-writer --skill article-publisher`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" || roleName == "" || len(skills) == 0 {
				return fmt.Errorf("--team, --role and --skill are all required")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			r, err := s.Role(team, roleName)
			if err != nil {
				return err
			}

			remove := make(map[string]bool, len(skills))
			for _, sk := range skills {
				remove[strings.TrimSpace(sk)] = true
			}

			before := len(r.Skills)
			kept := r.Skills[:0]
			for _, sk := range r.Skills {
				if !remove[sk] {
					kept = append(kept, sk)
				}
			}
			r.Skills = kept
			removed := before - len(r.Skills)

			if removed == 0 {
				fmt.Printf("ℹ  None of the specified skills were bound to role %q/%q.\n", team, roleName)
				return nil
			}

			if err := s.SaveRole(team, roleName, r); err != nil {
				return err
			}

			fmt.Printf("✓ Removed %d skill(s) from role %q/%q\n", removed, team, roleName)
			if len(r.Skills) > 0 {
				fmt.Printf("  Skills now: %s\n", strings.Join(r.Skills, ", "))
			} else {
				fmt.Printf("  Skills now: (none)\n")
			}
			fmt.Printf("\n  Run `agencycli sync` to push the updated context to hired agents.\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Team path, e.g. \"growth\"")
	cmd.Flags().StringVar(&roleName, "role", "", "Role name, e.g. \"content-writer\"")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "Skill name(s) to remove (comma-separated or repeated flag)")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("role")
	_ = cmd.MarkFlagRequired("skill")
	return cmd
}

// ── role list ─────────────────────────────────────────────────────────────────

func newRoleListCmd() *cobra.Command {
	var team string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List roles under a team",
		Example: `  agencycli role list --team engineering
  agencycli role list --team growth`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if team == "" {
				return fmt.Errorf("--team is required")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			roles, err := s.ListRoles(team)
			if err != nil {
				return err
			}
			if len(roles) == 0 {
				fmt.Printf("No roles defined under team %q.\n", team)
				fmt.Printf("Create one with: agencycli create role --team %q --name <name>\n", team)
				return nil
			}

			for _, re := range roles {
				skillStr := "(none)"
				if len(re.Role.Skills) > 0 {
					skillStr = strings.Join(re.Role.Skills, ", ")
				}
				fmt.Printf("  %-22s  skills:%-40s", re.Name, skillStr)
				if re.Role.Description != "" {
					fmt.Printf("  %s", re.Role.Description)
				}
				fmt.Println()
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&team, "team", "", "Team path to list roles for")
	_ = cmd.MarkFlagRequired("team")
	return cmd
}
