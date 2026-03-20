package ctxbuild

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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

// Build assembles the MergedContext for projectName with the team context from
// teamPath. roleName is optional (pass "" to skip the role layer).
//
// Layer order:
//
//  1. Agency
//  2. Each team in the chain (top-level → teamPath)
//  3. Role (when roleName != "")
//  4. Project
//
// Skills are deduplicated and collected from team chain → role (role skills
// take precedence / are appended). Empty prompt files are silently skipped.
func (b *Builder) Build(projectName, teamPath, roleName string) (*MergedContext, error) {
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

	addSkill := func(skillName string) {
		if seenSkills[skillName] {
			return
		}
		seenSkills[skillName] = true
		skill, err := b.store.Skill(skillName)
		if err != nil {
			return // skill definition missing — skip gracefully
		}
		skillPrompt, _ := b.store.SkillPrompt(skillName)
		files := loadSkillFiles(b.store.SkillDir(skillName))
		mc.Skills = append(mc.Skills, SkillDef{
			Name:        skill.Name,
			Description: skill.Description,
			Prompt:      skillPrompt,
			Files:       files,
		})
	}

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
			addSkill(skillName)
		}
	}

	// 3. Role layer (optional)
	if roleName != "" {
		// The role lives under the leaf team of the chain.
		leafTeam := teamPath
		role, err := b.store.Role(leafTeam, roleName)
		if err != nil {
			return nil, fmt.Errorf("ctxbuild: role %q/%q: %w", leafTeam, roleName, err)
		}
		rolePrompt, err := b.store.RolePrompt(leafTeam, roleName)
		if err != nil {
			return nil, fmt.Errorf("ctxbuild: role prompt %q/%q: %w", leafTeam, roleName, err)
		}
		if strings.TrimSpace(rolePrompt) != "" {
			mc.Layers = append(mc.Layers, ContextLayer{
				Source:  "role:" + leafTeam + "/" + roleName,
				Content: rolePrompt,
			})
		}
		// Role skills are appended after team skills.
		for _, skillName := range role.Skills {
			addSkill(skillName)
		}
	}

	// 4. Project layer
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
		// Include file contents in the hash so that script changes trigger sync.
		combined := sk.Prompt
		for _, f := range sk.Files {
			combined += f.Name + string(f.Content)
		}
		hashes["skill:"+sk.Name] = ContentHash(combined)
	}
	return hashes
}

// loadSkillFiles recursively scans skillDir and returns all bundled
// non-definition files as SkillFile entries.
// Definition files (SKILL.md, skill.yaml, prompt.md) at the root level are
// excluded because they are consumed by the store layer, not deployed to agents.
// Files in subdirectories are included with their relative path as the Name
// (e.g. "scripts/deploy.sh"), preserving the directory structure when deployed.
// Errors are silently ignored so a missing or empty directory is handled
// gracefully (skills without bundled files are perfectly valid).
func loadSkillFiles(skillDir string) []SkillFile {
	var files []SkillFile
	_ = filepath.WalkDir(skillDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(skillDir, path)
		if err != nil {
			return nil
		}
		// Exclude top-level definition files only.
		if !strings.Contains(rel, string(filepath.Separator)) {
			switch rel {
			case "SKILL.md", "skill.yaml", "prompt.md":
				return nil
			}
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		files = append(files, SkillFile{Name: rel, Content: content})
		return nil
	})
	return files
}
