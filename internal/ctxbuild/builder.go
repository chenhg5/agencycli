package ctxbuild

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/agentorg/agentorg/internal/store"
)

// Builder constructs a MergedContext for a given (project, department) pair
// by reading prompt files from the store in the correct inheritance order.
type Builder struct {
	store store.Store
}

// NewBuilder creates a Builder backed by the given store.
func NewBuilder(s store.Store) *Builder {
	return &Builder{store: s}
}

// Build assembles the MergedContext for projectName with the department
// context from deptPath. The returned context contains:
//
//  1. Company layer
//  2. Each department in the chain from top-level to deptPath
//  3. Project layer
//
// Empty prompt files are silently skipped so callers don't need to guard
// against empty layers. Skills are deduplicated and collected from all
// departments in the chain (parent skills are included).
func (b *Builder) Build(projectName, deptPath string) (*MergedContext, error) {
	mc := &MergedContext{}

	// 1. Company layer
	companyPrompt, err := b.store.CompanyPrompt()
	if err != nil {
		return nil, fmt.Errorf("ctxbuild: company prompt: %w", err)
	}
	if strings.TrimSpace(companyPrompt) != "" {
		mc.Layers = append(mc.Layers, ContextLayer{
			Source:  "company",
			Content: companyPrompt,
		})
	}

	// 2. Department chain layers + skill collection
	chain := ResolveChain(deptPath)
	seenSkills := make(map[string]bool)

	for _, dp := range chain {
		dept, err := b.store.Department(dp)
		if err != nil {
			return nil, fmt.Errorf("ctxbuild: department %q: %w", dp, err)
		}

		prompt, err := b.store.DeptPrompt(dp)
		if err != nil {
			return nil, fmt.Errorf("ctxbuild: dept prompt %q: %w", dp, err)
		}
		if strings.TrimSpace(prompt) != "" {
			mc.Layers = append(mc.Layers, ContextLayer{
				Source:  "department:" + dp,
				Content: prompt,
			})
		}

		for _, skillName := range dept.Skills {
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
