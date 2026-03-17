// Package template provides Pack / Unpack helpers for agencycli agency templates.
//
// A template is a .tar.gz archive containing:
//
//	template.json         — TemplateManifest metadata (npm package.json style)
//	agency-prompt.md      — agency-level prompt ({{AGENCY_NAME}} substituted on apply)
//	teams/<team>/...      — team prompts and role definitions
//	skills/<skill>/...    — reusable skill files
//
// Runtime state (projects/, agents/, tasks, heartbeat, crons, .agencycli/) is
// always excluded from archives.
package template

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/scaffold"
)

const ManifestFile = "template.json"

// skeletonDirs are the directories / files copied into a template archive.
// Runtime directories (projects, .agencycli) are intentionally absent.
var skeletonEntries = []string{
	ManifestFile,
	"agency-prompt.md",
	"teams",
	"skills",
	"agent-playbooks",      // wakeup.md files for each agent role
	"project-blueprints",  // declarative project.yaml blueprints
}

// ── Pack ─────────────────────────────────────────────────────────────────────

// Pack creates a .tar.gz template archive from an existing agency workspace.
// manifest provides the metadata written to template.json inside the archive.
// outputPath should end in ".tar.gz".
func Pack(agencyDir, outputPath string, manifest *entity.TemplateManifest) error {
	agencyDir = filepath.Clean(agencyDir)

	if manifest.CreatedAt == "" {
		manifest.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	}

	out, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	gz := gzip.NewWriter(out)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()

	// Write template.json as the first entry.
	jsonBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal template.json: %w", err)
	}
	if err := writeBytes(tw, ManifestFile, jsonBytes); err != nil {
		return err
	}

	// Write the rest of the skeleton (skip template.json — already written).
	for _, entry := range skeletonEntries[1:] {
		src := filepath.Join(agencyDir, entry)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := addDirToTar(tw, src, entry); err != nil {
				return err
			}
		} else {
			if err := addFileToTar(tw, src, entry); err != nil {
				return err
			}
		}
	}
	return nil
}

// ── Unpack ────────────────────────────────────────────────────────────────────

// Unpack extracts a .tar.gz template archive into destDir (must already exist),
// substituting {{AGENCY_NAME}} with agencyName in all text files.
func Unpack(archivePath, destDir, agencyName string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()
	return unpackReader(f, destDir, agencyName)
}

func unpackReader(r io.Reader, destDir, agencyName string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read tar: %w", err)
		}
		// Security: reject absolute paths and path traversal.
		if filepath.IsAbs(hdr.Name) || strings.Contains(hdr.Name, "..") {
			return fmt.Errorf("unsafe path in archive: %s", hdr.Name)
		}
		// Skip template.json — it's metadata, not part of the workspace.
		if hdr.Name == ManifestFile {
			continue
		}
		target := filepath.Join(destDir, filepath.FromSlash(hdr.Name))
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			data, err := io.ReadAll(tr)
			if err != nil {
				return err
			}
			content := strings.ReplaceAll(string(data), "{{AGENCY_NAME}}", agencyName)
			if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
				return err
			}
		}
	}
	return nil
}

// ── ApplyDir ──────────────────────────────────────────────────────────────────

// ApplyDir copies a template from a plain directory into destDir, substituting
// {{AGENCY_NAME}} in all text file contents.
func ApplyDir(templateDir, destDir, agencyName string) error {
	for _, entry := range skeletonEntries[1:] { // skip template.json
		src := filepath.Join(templateDir, entry)
		if _, err := os.Stat(src); os.IsNotExist(err) {
			continue
		}
		info, err := os.Stat(src)
		if err != nil {
			return err
		}
		dest := filepath.Join(destDir, entry)
		if info.IsDir() {
			if err := copyDirWithSubstitution(src, dest, agencyName); err != nil {
				return err
			}
		} else {
			if err := copyFileWithSubstitution(src, dest, agencyName); err != nil {
				return err
			}
		}
	}
	return nil
}

func copyDirWithSubstitution(srcDir, destDir, agencyName string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		target := filepath.Join(destDir, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFileWithSubstitution(path, target, agencyName)
	})
}

func copyFileWithSubstitution(src, dest, agencyName string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	content := strings.ReplaceAll(string(data), "{{AGENCY_NAME}}", agencyName)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dest, []byte(content), 0o644)
}

// ── Manifest helpers ──────────────────────────────────────────────────────────

// ReadManifestFromDir reads template.json from a directory.
func ReadManifestFromDir(dir string) (*entity.TemplateManifest, error) {
	data, err := os.ReadFile(filepath.Join(dir, ManifestFile))
	if err != nil {
		return nil, err
	}
	var m entity.TemplateManifest
	return &m, json.Unmarshal(data, &m)
}

// ReadManifestFromArchive reads template.json from a .tar.gz without fully
// extracting it.
func ReadManifestFromArchive(archivePath string) (*entity.TemplateManifest, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return nil, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err != nil {
			break
		}
		if filepath.Base(hdr.Name) == ManifestFile {
			data, err := io.ReadAll(tr)
			if err != nil {
				return nil, err
			}
			var m entity.TemplateManifest
			return &m, json.Unmarshal(data, &m)
		}
	}
	return nil, fmt.Errorf("template.json not found in archive")
}

// ── InitAgencyFromTemplate ────────────────────────────────────────────────────

// InitAgencyFromTemplate initialises a new agency workspace in root by:
//  1. Creating the directory
//  2. Applying the template skeleton (substituting {{AGENCY_NAME}})
//  3. Writing .agencycli/agency.yaml via scaffold.InitAgency
func InitAgencyFromTemplate(root, agencyName, agencyDesc string,
	apply func(destDir, agencyName string) error) error {
	if err := os.MkdirAll(root, 0o755); err != nil {
		return err
	}
	if err := apply(root, agencyName); err != nil {
		return fmt.Errorf("apply template: %w", err)
	}
	a := &entity.Agency{Name: agencyName, Description: agencyDesc}
	return scaffold.InitAgency(root, a)
}

// ── tar helpers ───────────────────────────────────────────────────────────────

func writeBytes(tw *tar.Writer, name string, data []byte) error {
	if err := tw.WriteHeader(&tar.Header{
		Typeflag: tar.TypeReg,
		Name:     name,
		Size:     int64(len(data)),
		Mode:     0o644,
	}); err != nil {
		return err
	}
	_, err := tw.Write(data)
	return err
}

func addDirToTar(tw *tar.Writer, srcDir, archivePath string) error {
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(srcDir, path)
		target := filepath.Join(archivePath, rel)
		if d.IsDir() {
			return tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     target + "/",
				Mode:     0o755,
			})
		}
		return addFileToTar(tw, path, target)
	})
}

func addFileToTar(tw *tar.Writer, srcPath, archivePath string) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = archivePath
	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}
	_, err = io.Copy(tw, f)
	return err
}
