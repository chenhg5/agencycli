package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
)

const (
	updateGithubRepo   = "chenhg5/agencycli"
	updateGithubAPI    = "https://api.github.com/repos/" + updateGithubRepo + "/releases/latest"
	updateGithubAllAPI = "https://api.github.com/repos/" + updateGithubRepo + "/releases"
	updateDownloadBase = "https://github.com/" + updateGithubRepo + "/releases/download"
	updateGiteeAPI     = "https://gitee.com/api/v5/repos/cg33/agentorg/releases/latest"
)

var cachedLatestVersion struct {
	version   string
	body      string
	timestamp time.Time
	mu        sync.RWMutex
}

const versionCheckTTL = time.Hour

type githubRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Body       string `json:"body"`
	Prerelease bool   `json:"prerelease"`
}

func newCheckUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "check-update",
		Short: "Check if a newer version is available",
		Run: func(_ *cobra.Command, args []string) {
			pre := false
			for _, a := range args {
				if a == "--pre" || a == "--beta" {
					pre = true
				}
			}
			release, err := fetchUpdateRelease(pre)
			if err != nil {
				return
			}
			if isNewerVersion(release.TagName, version) {
				hint := "agencycli update"
				if release.Prerelease {
					hint = "agencycli update --pre"
				}
				fmt.Fprintf(os.Stderr, "Update available: %s -> %s (run: %s)\n", version, release.TagName, hint)
			} else {
				fmt.Printf("Already up to date (%s).\n", version)
			}
		},
	}
}

func newUpdateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Self-update to the latest version",
		Long: `Downloads and replaces the agencycli binary with the latest release
from GitHub. Use --pre to include pre-release versions.`,
		RunE: func(_ *cobra.Command, args []string) error {
			pre := false
			for _, a := range args {
				if a == "--pre" || a == "--beta" {
					pre = true
				}
			}
			return runSelfUpdate(pre)
		},
	}
}

func runSelfUpdate(pre bool) error {
	fmt.Printf("agencycli %s\n", version)
	if pre {
		fmt.Println("Checking for updates (including pre-releases)...")
	} else {
		fmt.Println("Checking for updates...")
	}

	release, err := fetchUpdateRelease(pre)
	if err != nil {
		return fmt.Errorf("error checking updates: %w", err)
	}

	latest := release.TagName
	if !isNewerVersion(latest, version) {
		fmt.Printf("Already up to date (%s >= %s).\n", version, latest)
		return nil
	}

	label := latest
	if release.Prerelease {
		label += " (pre-release)"
	}
	fmt.Printf("New version available: %s -> %s\n", version, label)

	archiveAsset := updateArchiveAssetName(latest)
	archiveURL := fmt.Sprintf("%s/%s/%s", updateDownloadBase, latest, archiveAsset)
	fmt.Printf("Downloading %s ...\n", archiveURL)

	tmpFile, err := updateDownloadToTemp(archiveURL)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer os.Remove(tmpFile)

	binTmp, err := updateExtractBinary(tmpFile, archiveAsset)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}
	defer os.Remove(binTmp)

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot locate current binary: %w", err)
	}

	if err := updateReplaceExecutable(execPath, binTmp); err != nil {
		return fmt.Errorf("update failed: %w", err)
	}

	syncNpmVersion(execPath, strings.TrimPrefix(latest, "v"))

	fmt.Printf("Updated to %s\n", latest)
	fmt.Println("Restart agencycli to use the new version.")
	return nil
}

// ── async version check ─────────────────────────────────────

func checkUpdateAsync() {
	if version == "dev" || version == "" {
		return
	}
	go func() {
		release, err := fetchLatestStableFromGitee()
		if err != nil || release == nil || release.TagName == "" {
			release, err = fetchLatestStable()
			if err != nil || release == nil {
				return
			}
		}
		cachedLatestVersion.mu.Lock()
		cachedLatestVersion.version = release.TagName
		cachedLatestVersion.body = release.Body
		cachedLatestVersion.timestamp = time.Now()
		cachedLatestVersion.mu.Unlock()
	}()
}

// GetCachedUpdateInfo returns cached version check results for the API.
func GetCachedUpdateInfo() (latestVersion, releaseBody string, hasUpdate bool) {
	if version == "dev" || version == "" {
		return "", "", false
	}
	cachedLatestVersion.mu.RLock()
	ver := cachedLatestVersion.version
	body := cachedLatestVersion.body
	ts := cachedLatestVersion.timestamp
	cachedLatestVersion.mu.RUnlock()

	if ver == "" || time.Since(ts) > versionCheckTTL {
		checkUpdateAsync()
		return "", "", false
	}

	if isNewerVersion(ver, version) {
		return ver, body, true
	}
	return ver, "", false
}

// ── fetch helpers ───────────────────────────────────────────

func fetchUpdateRelease(pre bool) (*githubRelease, error) {
	if pre {
		return fetchLatestPreRelease()
	}
	return fetchLatestStable()
}

func fetchLatestStableFromGitee() (*githubRelease, error) {
	client := &http.Client{Timeout: 3 * time.Second}
	req, _ := http.NewRequest("GET", updateGiteeAPI, nil)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("gitee API returned HTTP %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}
	if release.Prerelease {
		return nil, nil
	}
	return &release, nil
}

func fetchLatestStable() (*githubRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", updateGithubAPI, nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var release githubRelease
			if err := json.NewDecoder(resp.Body).Decode(&release); err == nil {
				return &release, nil
			}
		}
	}

	latestURL := "https://github.com/" + updateGithubRepo + "/releases/latest"
	noRedirect := &http.Client{
		Timeout: 15 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp2, err := noRedirect.Get(latestURL)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp2.Body.Close()

	loc := resp2.Header.Get("Location")
	if loc == "" {
		return nil, fmt.Errorf("no release found")
	}
	parts := strings.Split(loc, "/tag/")
	if len(parts) != 2 {
		return nil, fmt.Errorf("unexpected redirect: %s", loc)
	}
	return &githubRelease{TagName: parts[1], HTMLURL: loc}, nil
}

func fetchLatestPreRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 15 * time.Second}
	req, _ := http.NewRequest("GET", updateGithubAllAPI+"?per_page=10", nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned HTTP %d", resp.StatusCode)
	}

	var releases []githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("parse releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}
	return &releases[0], nil
}

// ── version comparison ──────────────────────────────────────

func isNewerVersion(latest, current string) bool {
	if latest == "" || current == "" {
		return false
	}
	if strings.HasPrefix(current, "dev") {
		return true
	}

	l := strings.TrimPrefix(latest, "v")
	c := strings.TrimPrefix(current, "v")

	lBase, lPre, _ := strings.Cut(l, "-")
	cBase, cPre, _ := strings.Cut(c, "-")

	lParts := strings.Split(lBase, ".")
	cParts := strings.Split(cBase, ".")

	for i := 0; i < len(lParts) || i < len(cParts); i++ {
		var lv, cv int
		if i < len(lParts) {
			_, _ = fmt.Sscanf(lParts[i], "%d", &lv)
		}
		if i < len(cParts) {
			_, _ = fmt.Sscanf(cParts[i], "%d", &cv)
		}
		if lv > cv {
			return true
		}
		if lv < cv {
			return false
		}
	}

	if cPre != "" && lPre == "" {
		return true
	}
	if cPre == "" && lPre != "" {
		return false
	}
	if lPre != "" && cPre != "" {
		return comparePreReleaseSuffix(lPre, cPre) > 0
	}
	return false
}

func comparePreReleaseSuffix(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	max := len(aParts)
	if len(bParts) > max {
		max = len(bParts)
	}
	for i := 0; i < max; i++ {
		var ap, bp string
		if i < len(aParts) {
			ap = aParts[i]
		}
		if i < len(bParts) {
			bp = bParts[i]
		}

		var an, bn int
		aN, _ := fmt.Sscanf(ap, "%d", &an)
		bN, _ := fmt.Sscanf(bp, "%d", &bn)
		aIsNum := aN == 1 && fmt.Sprintf("%d", an) == ap
		bIsNum := bN == 1 && fmt.Sprintf("%d", bn) == bp

		if aIsNum && bIsNum {
			if an != bn {
				return an - bn
			}
			continue
		}
		if ap < bp {
			return -1
		}
		if ap > bp {
			return 1
		}
	}
	return 0
}

// ── download & extract ──────────────────────────────────────

func updateArchiveAssetName(tag string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	base := fmt.Sprintf("agencycli-%s-%s-%s", tag, goos, goarch)
	if goos == "windows" {
		return base + ".zip"
	}
	return base + ".tar.gz"
}

func updateDownloadToTemp(url string) (string, error) {
	client := &http.Client{
		Timeout: 5 * time.Minute,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "agencycli-update-*")
	if err != nil {
		return "", err
	}

	size, err := io.Copy(tmp, resp.Body)
	if err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", fmt.Errorf("write: %w", err)
	}
	tmp.Close()

	fmt.Printf("Downloaded %.1f MB\n", float64(size)/1024/1024)
	return tmp.Name(), nil
}

func updateExtractBinary(archivePath, archiveName string) (string, error) {
	if strings.HasSuffix(archiveName, ".zip") {
		return updateExtractFromZip(archivePath)
	}
	return updateExtractFromTarGz(archivePath)
}

func updateExtractFromTarGz(archivePath string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if strings.HasPrefix(hdr.Name, "agencycli") {
			tmp, err := os.CreateTemp("", "agencycli-update-bin-*")
			if err != nil {
				return "", err
			}
			if _, err := io.Copy(tmp, tr); err != nil {
				tmp.Close()
				os.Remove(tmp.Name())
				return "", fmt.Errorf("extract: %w", err)
			}
			tmp.Close()
			return tmp.Name(), nil
		}
	}
	return "", fmt.Errorf("binary not found in archive")
}

func updateExtractFromZip(archivePath string) (string, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", fmt.Errorf("zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
		if !strings.HasPrefix(f.Name, "agencycli") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", err
		}
		tmp, err := os.CreateTemp("", "agencycli-update-bin-*")
		if err != nil {
			rc.Close()
			return "", err
		}
		if _, err := io.Copy(tmp, rc); err != nil {
			tmp.Close()
			rc.Close()
			os.Remove(tmp.Name())
			return "", fmt.Errorf("extract: %w", err)
		}
		rc.Close()
		tmp.Close()
		return tmp.Name(), nil
	}
	return "", fmt.Errorf("binary not found in archive")
}

func updateReplaceExecutable(target, src string) error {
	if err := os.Chmod(src, 0o755); err != nil {
		return fmt.Errorf("chmod: %w", err)
	}

	backup := target + ".old"
	os.Remove(backup)

	if err := os.Rename(target, backup); err != nil {
		return fmt.Errorf("backup old binary: %w", err)
	}

	if err := updateCopyFile(src, target); err != nil {
		if restoreErr := os.Rename(backup, target); restoreErr != nil {
			slog.Warn("update: failed to restore old binary after copy error", "error", restoreErr)
		}
		return fmt.Errorf("install new binary: %w", err)
	}

	if err := os.Chmod(target, 0o755); err != nil {
		return fmt.Errorf("chmod new binary: %w", err)
	}

	os.Remove(backup)
	return nil
}

func updateCopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func syncNpmVersion(execPath, newVer string) {
	binDir := filepath.Dir(execPath)
	if filepath.Base(binDir) != "bin" {
		return
	}
	pkgDir := filepath.Dir(binDir)
	pkgJSON := filepath.Join(pkgDir, "package.json")

	data, err := os.ReadFile(pkgJSON)
	if err != nil {
		return
	}

	var pkg map[string]any
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	name, _ := pkg["name"].(string)
	if !strings.Contains(name, "agencycli") {
		return
	}

	oldVer, _ := pkg["version"].(string)
	oldNorm := strings.TrimPrefix(oldVer, "v")
	newNorm := strings.TrimPrefix(newVer, "v")
	if oldNorm == newNorm {
		return
	}

	pkg["version"] = newVer
	out, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		return
	}
	out = append(out, '\n')
	if err := os.WriteFile(pkgJSON, out, 0o644); err != nil {
		slog.Warn("update: failed to sync npm package.json version", "error", err)
		fmt.Println("Note: npm package version not synced. If the next run re-downloads an old version,")
		fmt.Println("  please run: npm update -g @agencycli/agencycli")
	}
}
