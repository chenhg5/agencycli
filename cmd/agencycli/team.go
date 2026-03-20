package main

import (
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/store"
	"github.com/spf13/cobra"
)

func newTeamCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "team",
		Short: "Manage team definitions (skills)",
	}
	cmd.AddCommand(
		newTeamSkillCmd(),
	)
	return cmd
}

// ── team skill ────────────────────────────────────────────────────────────────

func newTeamSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Add or remove skills bound to a team",
	}
	cmd.AddCommand(
		newTeamSkillAddCmd(),
		newTeamSkillRemoveCmd(),
	)
	return cmd
}

func newTeamSkillAddCmd() *cobra.Command {
	var teamPath string
	var skills []string

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Bind one or more skills to a team",
		Example: `  agencycli team skill add --team growth --skill article-publisher
  agencycli team skill add --team engineering --skill github-push-relay,docker`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamPath == "" || len(skills) == 0 {
				return fmt.Errorf("--team and --skill are all required")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			t, err := s.Team(teamPath)
			if err != nil {
				return err
			}

			added := 0
			existing := make(map[string]bool, len(t.Skills))
			for _, sk := range t.Skills {
				existing[sk] = true
			}
			for _, sk := range skills {
				sk = strings.TrimSpace(sk)
				if sk == "" || existing[sk] {
					continue
				}
				t.Skills = append(t.Skills, sk)
				existing[sk] = true
				added++
			}

			if added == 0 {
				fmt.Printf("ℹ  No new skills to add (all already bound to team %q).\n", teamPath)
				return nil
			}

			if err := s.SaveTeam(teamPath, t); err != nil {
				return err
			}

			fmt.Printf("✓ Added %d skill(s) to team %q\n", added, teamPath)
			fmt.Printf("  Skills now: %s\n", strings.Join(t.Skills, ", "))
			fmt.Printf("\n  Run `agencycli sync` to push the updated context to hired agents.\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&teamPath, "team", "", "Team path, e.g. \"growth\" or \"engineering/backend\"")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "Skill name(s) to add (comma-separated or repeated flag)")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("skill")
	return cmd
}

func newTeamSkillRemoveCmd() *cobra.Command {
	var teamPath string
	var skills []string

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Unbind one or more skills from a team",
		Example: `  agencycli team skill remove --team growth --skill article-publisher`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if teamPath == "" || len(skills) == 0 {
				return fmt.Errorf("--team and --skill are all required")
			}
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			s := store.NewFS(root)

			t, err := s.Team(teamPath)
			if err != nil {
				return err
			}

			remove := make(map[string]bool, len(skills))
			for _, sk := range skills {
				remove[strings.TrimSpace(sk)] = true
			}

			before := len(t.Skills)
			kept := t.Skills[:0]
			for _, sk := range t.Skills {
				if !remove[sk] {
					kept = append(kept, sk)
				}
			}
			t.Skills = kept
			removed := before - len(t.Skills)

			if removed == 0 {
				fmt.Printf("ℹ  None of the specified skills were bound to team %q.\n", teamPath)
				return nil
			}

			if err := s.SaveTeam(teamPath, t); err != nil {
				return err
			}

			fmt.Printf("✓ Removed %d skill(s) from team %q\n", removed, teamPath)
			if len(t.Skills) > 0 {
				fmt.Printf("  Skills now: %s\n", strings.Join(t.Skills, ", "))
			} else {
				fmt.Printf("  Skills now: (none)\n")
			}
			fmt.Printf("\n  Run `agencycli sync` to push the updated context to hired agents.\n")
			return nil
		},
	}

	cmd.Flags().StringVar(&teamPath, "team", "", "Team path, e.g. \"growth\" or \"engineering/backend\"")
	cmd.Flags().StringSliceVar(&skills, "skill", nil, "Skill name(s) to remove (comma-separated or repeated flag)")
	_ = cmd.MarkFlagRequired("team")
	_ = cmd.MarkFlagRequired("skill")
	return cmd
}