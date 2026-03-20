package formatter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/chenhg5/agencycli/internal/ctxbuild"
)

const skillDirPlaceholder = "{{SKILL_DIR}}"

// deploySkillFiles copies all bundled files from sk.Files into destDir and
// returns the absolute path of destDir (used to resolve {{SKILL_DIR}}).
// destDir is created if it does not exist.
func deploySkillFiles(sk ctxbuild.SkillDef, destDir string) (string, error) {
	if len(sk.Files) == 0 {
		return destDir, nil
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", fmt.Errorf("skill %q: create dir: %w", sk.Name, err)
	}
	for _, f := range sk.Files {
		dst := filepath.Join(destDir, f.Name)
		// Ensure parent directory exists (handles subdirectory paths like "scripts/foo.sh").
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", fmt.Errorf("skill %q: create dir for %q: %w", sk.Name, f.Name, err)
		}
		mode := os.FileMode(0o644)
		// Preserve executable bit for shell scripts.
		if strings.HasSuffix(f.Name, ".sh") || strings.HasSuffix(f.Name, ".bash") {
			mode = 0o755
		}
		if err := os.WriteFile(dst, f.Content, mode); err != nil {
			return "", fmt.Errorf("skill %q: write file %q: %w", sk.Name, f.Name, err)
		}
	}
	return destDir, nil
}

// resolveSkillDir replaces all occurrences of {{SKILL_DIR}} in text with
// the absolute skill directory path. Returns text unchanged if skillDir is "".
func resolveSkillDir(text, skillDir string) string {
	if skillDir == "" || !strings.Contains(text, skillDirPlaceholder) {
		return text
	}
	return strings.ReplaceAll(text, skillDirPlaceholder, skillDir)
}

// resolveSkillsToDir deploys all skill files into <outDir>/.agencycli-skills/<name>/
// and returns a copy of the skills slice with {{SKILL_DIR}} resolved in each Prompt.
// Used by formatters that inline skills into a single text file (codex, cursor, etc.).
func resolveSkillsToDir(skills []ctxbuild.SkillDef, outDir string) []ctxbuild.SkillDef {
	result := make([]ctxbuild.SkillDef, len(skills))
	for i, sk := range skills {
		dest := filepath.Join(outDir, ".agencycli-skills", sk.Name)
		absDir, err := deploySkillFiles(sk, dest)
		if err != nil {
			absDir = dest // fallback: still resolve even if copy failed
		}
		resolved := sk
		resolved.Prompt = resolveSkillDir(sk.Prompt, absDir)
		result[i] = resolved
	}
	return result
}
