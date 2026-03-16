package ctxbuild

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/chenhg5/agencycli/internal/store"
)

// Builder constructs a MergedContext for a given (project, team) pair
// by reading prompt files from the store in the correct inheritance order.
type Builder struct {
	store store.Store
}

// NewBuilder creates a Builder backed by the given store.
func NewBuilder(s store.Store) *Builder {
	return &Builder{store: s}
}

// Build assembles the MergedContext for projectName with the team
// context from teamPath. The returned context contains:
//
//  1. Agency layer
//  2. Each team in the chain from top-level to teamPath
//  3. Project layer
//
// Empty prompt files are silently skipped so callers don't need to guard
// against empty layers. Skills are deduplicated and collected from all
// teams in the chain (parent skills are included).
func (b *Builder) Build(projectName, teamPath string) (*MergedContext, error) {
	mc := &MergedContext{}

	// 1. Agency layer
	agencyPrompt, err := b.store.AgencyPrompt()
	if err != nil {
		return nil, fmt.Errorf("ctxbuild: agency prompt: %w", err)
	}
	if strings.TrimSpace(agencyPrompt) != "" {
		mc.Layers = append(mc.Layers, ContextLayer{
			Source:  "agency",
			Content: agencyPrompt,
		})
	}

	// 2. Team chain layers + skill collection
	chain := ResolveChain(teamPath)
	seenSkills := make(map[string]bool)

	for _, tp := range chain {
		team, err := b.store.Team(tp)
		if err != nil {
			return nil, fmt.Errorf(
				"ctxbuild: team %q not found — "+
					"every level in the chain must exist; "+
					"run: agencycli create team --name %q",
				tp, tp,
			)
		}

		prompt, err := b.store.TeamPrompt(tp)
		if err != nil {
			return nil, fmt.Errorf("ctxbuild: team prompt %q: %w", tp, err)
		}
		if strings.TrimSpace(prompt) != "" {
			mc.Layers = append(mc.Layers, ContextLayer{
				Source:  "team:" + tp,
				Content: prompt,
			})
		}

		for _, skillName := range team.Skills {
			if seenSkills[skillName] {
				continue
			}
			seenSkills[skillName] = true

			skill, err := b.store.Skill(skillName)
			if err != nil {
				// Skill definition missing — skip gracefully but note it.
				continue
			}
			skillPrompt, _ := b.store.SkillPrompt(skillName)
			mc.Skills = append(mc.Skills, SkillDef{
				Name:        skill.Name,
				Description: skill.Description,
				Prompt:      skillPrompt,
			})
		}
	}

	// 3. Project layer
	projectPrompt, err := b.store.ProjectPrompt(projectName)
	if err != nil {
		return nil, fmt.Errorf("ctxbuild: project prompt %q: %w", projectName, err)
	}
	if strings.TrimSpace(projectPrompt) != "" {
		mc.Layers = append(mc.Layers, ContextLayer{
			Source:  "project:" + projectName,
			Content: projectPrompt,
		})
	}

	return mc, nil
}

// ContentHash computes a SHA-256 digest over all layer contents and skill
// prompts. It is used by AgentMeta.ContextHash to detect staleness.
func ContentHash(content string) string {
	sum := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", sum)
}

// LayerHashes returns a map from each layer's Source to the SHA-256 hash
// of its Content, ready to store in AgentMeta.ContextHash.
func LayerHashes(mc *MergedContext) map[string]string {
	hashes := make(map[string]string, len(mc.Layers)+len(mc.Skills))
	for _, l := range mc.Layers {
		hashes[l.Source] = ContentHash(l.Content)
	}
	for _, sk := range mc.Skills {
		hashes["skill:"+sk.Name] = ContentHash(sk.Prompt)
	}
	return hashes
}
