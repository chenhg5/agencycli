package runner

// updateClaudeSessionIndex adds or refreshes the given sessionID entry in the
// ~/.claude/projects/<hash>/sessions-index.json file.
//
// Claude Code only updates this index when running interactively. When agents
// run non-interactively (--output-format stream-json), the index is never
// updated, so `claude /resume` shows "No conversations found in this project."
// agencycli calls this after every agent run to keep the index current.

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type claudeSessionEntry struct {
	SessionID    string `json:"sessionId"`
	FullPath     string `json:"fullPath"`
	FileMtime    int64  `json:"fileMtime"`
	FirstPrompt  string `json:"firstPrompt"`
	Summary      string `json:"summary"`
	MessageCount int    `json:"messageCount"`
	Created      string `json:"created"`
	Modified     string `json:"modified"`
	GitBranch    string `json:"gitBranch"`
	ProjectPath  string `json:"projectPath"`
	IsSidechain  bool   `json:"isSidechain"`
}

type claudeSessionIndex struct {
	Version int                  `json:"version"`
	Entries []claudeSessionEntry `json:"entries"`
}

// updateClaudeSessionIndex updates the sessions-index.json for the project
// associated with agentDir so that sessionID is listed.
// Errors are silently swallowed — this is best-effort metadata maintenance.
func updateClaudeSessionIndex(agentDir, sessionID string) {
	if sessionID == "" || agentDir == "" {
		return
	}

	absDir, err := filepath.Abs(agentDir)
	if err != nil {
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return
	}

	// Claude Code computes the project directory name by replacing each "/"
	// in the absolute path with "-" (including the leading slash).
	projectHash := strings.ReplaceAll(absDir, "/", "-")
	claudeProjectDir := filepath.Join(homeDir, ".claude", "projects", projectHash)

	sessionFile := filepath.Join(claudeProjectDir, sessionID+".jsonl")
	if _, err := os.Stat(sessionFile); err != nil {
		// Session file not found; nothing to index yet.
		return
	}

	indexPath := filepath.Join(claudeProjectDir, "sessions-index.json")

	// Load current index (may not exist or may be empty).
	var idx claudeSessionIndex
	if raw, err := os.ReadFile(indexPath); err == nil {
		_ = json.Unmarshal(raw, &idx)
	}
	if idx.Version == 0 {
		idx.Version = 1
	}

	// Read metadata from the session file.
	st, err := os.Stat(sessionFile)
	if err != nil {
		return
	}
	firstPrompt, messageCount, created := parseClaudeSessionFile(sessionFile)
	if created == "" {
		created = st.ModTime().UTC().Format(time.RFC3339)
	}

	summary := firstPrompt
	if len(summary) > 80 {
		summary = summary[:80]
	}

	entry := claudeSessionEntry{
		SessionID:    sessionID,
		FullPath:     sessionFile,
		FileMtime:    st.ModTime().UnixMilli(),
		FirstPrompt:  firstPrompt,
		Summary:      summary,
		MessageCount: messageCount,
		Created:      created,
		Modified:     st.ModTime().UTC().Format(time.RFC3339),
		GitBranch:    "HEAD",
		ProjectPath:  absDir,
		IsSidechain:  false,
	}

	// Update existing entry or prepend new one.
	found := false
	for i, e := range idx.Entries {
		if e.SessionID == sessionID {
			idx.Entries[i] = entry
			found = true
			break
		}
	}
	if !found {
		// Prepend so the most recently touched session comes first.
		idx.Entries = append([]claudeSessionEntry{entry}, idx.Entries...)
	}

	// Remove entries whose session files no longer exist to keep the index clean.
	alive := idx.Entries[:0]
	for _, e := range idx.Entries {
		if _, err := os.Stat(e.FullPath); err == nil {
			alive = append(alive, e)
		}
	}
	idx.Entries = alive

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(indexPath, data, 0o644)
}

// parseClaudeSessionFile reads the .jsonl session file and extracts:
//   - firstPrompt: text of the first user message
//   - messageCount: number of user+assistant turns
//   - created: RFC3339 timestamp of the first message (if available)
func parseClaudeSessionFile(path string) (firstPrompt string, messageCount int, created string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 256*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, `"type"`) {
			continue
		}
		var obj map[string]any
		if json.Unmarshal([]byte(line), &obj) != nil {
			continue
		}
		typ, _ := obj["type"].(string)
		switch typ {
		case "user":
			if created == "" {
				if ts, ok := obj["timestamp"].(string); ok {
					created = ts
				}
			}
			if firstPrompt == "" {
				if msg, ok := obj["message"].(map[string]any); ok {
					firstPrompt = extractTextContent(msg["content"])
					if len(firstPrompt) > 200 {
						firstPrompt = firstPrompt[:200]
					}
				}
			}
			messageCount++
		case "assistant":
			messageCount++
		}
	}
	return
}

// extractTextContent returns the plain text from a Claude message content
// field, which can be a string or an array of content blocks.
func extractTextContent(content any) string {
	switch c := content.(type) {
	case string:
		return strings.TrimSpace(c)
	case []any:
		for _, item := range c {
			if m, ok := item.(map[string]any); ok {
				if m["type"] == "text" {
					if t, ok := m["text"].(string); ok {
						return strings.TrimSpace(t)
					}
				}
			}
		}
	}
	return ""
}
