package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

// ── agent log ─────────────────────────────────────────────────────────────────

func newAgentLogCmd() *cobra.Command {
	var (
		project   string
		agentName string
		showThink bool
		noTools   bool
		maxToolOut int
		current   bool
		follow    bool
	)

	cmd := &cobra.Command{
		Use:   "log [run-id]",
		Short: "View an agent's run logs (conversation transcripts)",
		Long: `View an agent's run history or a specific run's conversation transcript.

Without arguments, lists recent runs for the agent.
With --current, shows the most recent run.
With a run-id, shows that specific run.
Use --follow (-f) to stream new output as the run progresses.

Examples:
  agencycli agent log --project cc-connect --agent pm
  agencycli agent log --project cc-connect --agent pm --current
  agencycli agent log --project cc-connect --agent pm 20260321-104532-t-20260321-abc123.log
  agencycli agent log --project cc-connect --agent pm --current --follow
  agencycli agent log --project cc-connect --agent pm --follow`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := resolveRoot()
			if err != nil {
				return err
			}
			if project == "" || agentName == "" {
				return fmt.Errorf("--project and --agent are required")
			}

			ts := taskstore.New(root)
			logDir, err := ts.RunLogDir(project, agentName)
			if err != nil {
				return fmt.Errorf("run log dir: %w", err)
			}

			// List mode: no args and no --current
			if len(args) == 0 && !current {
				return listRuns(logDir, project, agentName)
			}

			// Determine which run to show.
			var logPath string
			if current || len(args) == 0 {
				logPath, err = latestRun(logDir)
				if err != nil {
					return err
				}
			} else {
				runID := args[0]
				// Allow bare run-id without .log extension.
				if !strings.HasSuffix(runID, ".log") {
					runID += ".log"
				}
				logPath = filepath.Join(logDir, runID)
			}

			if follow {
				return renderRunLogFollow(logPath, showThink, noTools, maxToolOut)
			}
			return renderRunLog(logPath, showThink, noTools, maxToolOut)
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "project name (required)")
	cmd.Flags().StringVar(&agentName, "agent", "", "agent name (required)")
	cmd.Flags().BoolVar(&showThink, "thinking", false, "show thinking blocks")
	cmd.Flags().BoolVar(&noTools, "no-tools", false, "hide tool calls and results")
	cmd.Flags().IntVar(&maxToolOut, "max-tool-output", 300, "max characters shown for tool output (0 = unlimited)")
	cmd.Flags().BoolVar(&current, "current", false, "show the most recent run")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream new output as the run progresses")

	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("agent")
	return cmd
}

// listRuns prints the recent run log files for an agent.
func listRuns(logDir, project, agentName string) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No runs found.")
			return nil
		}
		return fmt.Errorf("read log dir: %w", err)
	}

	type runEntry struct {
		name    string
		modTime time.Time
		size    int64
	}

	var runs []runEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, _ := e.Info()
		runs = append(runs, runEntry{
			name:    e.Name(),
			modTime: info.ModTime(),
			size:    info.Size(),
		})
	}

	if len(runs) == 0 {
		fmt.Println("No runs found.")
		return nil
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].modTime.After(runs[j].modTime)
	})

	fmt.Printf("Recent runs — %s/%s\n\n", project, agentName)
	fmt.Println("RUN ID                               MODIFIED           SIZE")
	fmt.Println("────────────────────────────────────────────────────────────────")

	for _, r := range runs {
		modStr := r.modTime.Local().Format("01-02 15:04:05")
		sizeStr := formatSize(r.size)
		fmt.Printf("%s  %s  %s\n", r.name, modStr, sizeStr)
	}
	fmt.Println()
	return nil
}

func latestRun(logDir string) (string, error) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no runs found")
		}
		return "", fmt.Errorf("read log dir: %w", err)
	}

	type runEntry struct {
		name    string
		modTime time.Time
	}
	var runs []runEntry
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".log") {
			continue
		}
		info, _ := e.Info()
		runs = append(runs, runEntry{e.Name(), info.ModTime()})
	}

	if len(runs) == 0 {
		return "", fmt.Errorf("no runs found")
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].modTime.After(runs[j].modTime)
	})

	return filepath.Join(logDir, runs[0].name), nil
}

func formatSize(n int64) string {
	if n > 1024*1024 {
		return fmt.Sprintf("%.1fM", float64(n)/1024/1024)
	}
	if n > 1024 {
		return fmt.Sprintf("%.1fk", float64(n)/1024)
	}
	return fmt.Sprintf("%dB", n)
}

// ── renderer (moved from tasklog.go) ──────────────────────────────────────────

// logEvent is the minimal envelope shared by all stream-json event types.
type logEvent struct {
	Type     string          `json:"type"`
	Subtype  string          `json:"subtype"`
	Message  json.RawMessage `json:"message"`
	IsError  bool            `json:"is_error"`
	DurationMs int           `json:"duration_ms"`
	NumTurns int            `json:"num_turns"`
	Result   string          `json:"result"`
	SessionID string        `json:"session_id"`
	Model    string         `json:"model"`
}

type msgEnvelope struct {
	Role    string       `json:"role"`
	Content []msgContent `json:"content"`
	Model   string       `json:"model"`
	Usage   *usageBlock  `json:"usage"`
}

type msgContent struct {
	Type      string          `json:"type"`
	Text      string          `json:"text"`
	Thinking  string          `json:"thinking"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	ToolUseID string          `json:"tool_use_id"`
	Content   interface{}     `json:"content"`
}

type usageBlock struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func renderRunLog(path string, showThink, noTools bool, maxOut int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	// ── header section (non-JSON lines at the top) ───────────────────────────
	var headerLines []string
	var jsonLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			jsonLines = append(jsonLines, line)
			for scanner.Scan() {
				jsonLines = append(jsonLines, scanner.Text())
			}
			break
		}
		headerLines = append(headerLines, line)
	}

	// Parse run header.
	var runAgent, runTask, started string
	for _, l := range headerLines {
		if strings.HasPrefix(l, "=== agencycli run:") || strings.HasPrefix(l, "=== agencycli exec:") {
			parts := strings.Fields(l)
			for _, p := range parts {
				if strings.HasPrefix(p, "task=") {
					runTask = strings.TrimPrefix(p, "task=")
				}
				if strings.Contains(p, "/") && !strings.Contains(p, "=") && p != "===" {
					runAgent = p
				}
			}
		}
		if strings.HasPrefix(l, "Started:") {
			started = strings.TrimPrefix(l, "Started: ")
		}
	}

	// ── parse and render events ───────────────────────────────────────────────
	var (
		model          string
		sessionID      string
		totalIn, totalOut int
		numTurns       int
		durationMs     int
		finalResult    string
		isError        bool
		turnCount      int
	)

	type renderedTurn struct {
		lines []string
	}
	var turns []renderedTurn

	for _, raw := range jsonLines {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "===") {
			continue
		}
		if !strings.HasPrefix(raw, "{") {
			continue
		}
		var ev logEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "system":
			if ev.Subtype == "init" {
				var raw2 map[string]interface{}
				_ = json.Unmarshal([]byte(raw), &raw2)
				if m, ok := raw2["model"].(string); ok {
					model = m
				}
				if s, ok := raw2["session_id"].(string); ok {
					sessionID = s
				}
			}

		case "assistant":
			var msg msgEnvelope
			if err := json.Unmarshal(ev.Message, &msg); err != nil {
				continue
			}
			if msg.Model != "" {
				model = msg.Model
			}
			if msg.Usage != nil {
				totalIn += msg.Usage.InputTokens
				totalOut += msg.Usage.OutputTokens
			}

			var lines []string
			for _, c := range msg.Content {
				switch c.Type {
				case "thinking":
					if showThink && strings.TrimSpace(c.Thinking) != "" {
						lines = append(lines, formatThinking(c.Thinking))
					}
				case "text":
					if strings.TrimSpace(c.Text) != "" {
						lines = append(lines, formatAssistantText(c.Text))
					}
				case "tool_use":
					if !noTools {
						lines = append(lines, formatToolCall(c.Name, c.Input))
					}
				}
			}
			if len(lines) > 0 {
				turnCount++
				turns = append(turns, renderedTurn{lines: lines})
			}

		case "user":
			if noTools {
				continue
			}
			var msg msgEnvelope
			if err := json.Unmarshal(ev.Message, &msg); err != nil {
				continue
			}
			var lines []string
			for _, c := range msg.Content {
				if c.Type == "tool_result" {
					lines = append(lines, formatToolResult(c.Content, maxOut))
				}
			}
			if len(lines) > 0 {
				turns = append(turns, renderedTurn{lines: lines})
			}

		case "result":
			numTurns = ev.NumTurns
			durationMs = ev.DurationMs
			finalResult = ev.Result
			isError = ev.IsError
		}
	}

	// ── output ───────────────────────────────────────────────────────────────
	width := 72

	fmt.Println(divider('═', width))
	if runAgent != "" {
		fmt.Printf("  Agent    : %s\n", runAgent)
	}
	if runTask != "" {
		fmt.Printf("  Task     : %s\n", runTask)
	}
	if sessionID != "" {
		fmt.Printf("  Session  : %s\n", sessionID)
	}
	if model != "" {
		fmt.Printf("  Model    : %s\n", model)
	}
	if started != "" {
		fmt.Printf("  Started  : %s\n", strings.TrimSpace(started))
	}
	if durationMs > 0 {
		fmt.Printf("  Duration : %s\n", fmtDurationMs(durationMs))
	}
	if numTurns > 0 {
		fmt.Printf("  Turns    : %d\n", numTurns)
	}
	if totalIn+totalOut > 0 {
		fmt.Printf("  Tokens   : %d in / %d out\n", totalIn, totalOut)
	}
	fmt.Println(divider('═', width))
	fmt.Println()

	for _, t := range turns {
		for _, l := range t.lines {
			fmt.Println(l)
		}
	}

	if finalResult != "" {
		fmt.Println()
		if isError {
			fmt.Println(divider('─', width))
			fmt.Println("  ✗ FAILED")
		} else {
			fmt.Println(divider('─', width))
			fmt.Println("  ✓ Result")
		}
		fmt.Println(divider('─', width))
		for _, l := range strings.Split(finalResult, "\n") {
			fmt.Printf("  %s\n", l)
		}
		fmt.Println(divider('─', width))
	}

	_ = turnCount
	_ = sessionID
	return nil
}

// renderRunLogFollow renders an existing run log then tails it for new output.
// It streams each turn as it completes rather than accumulating everything first.
func renderRunLogFollow(path string, showThink, noTools bool, maxOut int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open log: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

	// ── Phase 1: parse and render existing content ────────────────────────────
	var headerLines []string
	var jsonLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(strings.TrimSpace(line), "{") {
			jsonLines = append(jsonLines, line)
			for scanner.Scan() {
				jsonLines = append(jsonLines, scanner.Text())
			}
			break
		}
		headerLines = append(headerLines, line)
	}

	// Parse header.
	var runAgent, runTask, started string
	for _, l := range headerLines {
		if strings.HasPrefix(l, "=== agencycli run:") || strings.HasPrefix(l, "=== agencycli exec:") {
			parts := strings.Fields(l)
			for _, p := range parts {
				if strings.HasPrefix(p, "task=") {
					runTask = strings.TrimPrefix(p, "task=")
				}
				if strings.Contains(p, "/") && !strings.Contains(p, "=") && p != "===" {
					runAgent = p
				}
			}
		}
		if strings.HasPrefix(l, "Started:") {
			started = strings.TrimPrefix(l, "Started: ")
		}
	}

	// Stream-parse existing lines and render each turn immediately.
	// Also track final metadata for the header we print after the replay.
	streamState := &followState{}

	for _, raw := range jsonLines {
		raw = strings.TrimSpace(raw)
		if raw == "" || strings.HasPrefix(raw, "===") {
			continue
		}
		if !strings.HasPrefix(raw, "{") {
			continue
		}
		var ev logEvent
		if err := json.Unmarshal([]byte(raw), &ev); err != nil {
			continue
		}

		streamParseEvent(ev, raw, streamState, showThink, noTools, maxOut)
	}

	// Flush any remaining buffered turn.
	if lines := streamState.flushLines(showThink, noTools, maxOut); len(lines) > 0 {
		for _, l := range lines {
			fmt.Println(l)
		}
	}

	// Print the header and any accumulated turns.
	printFollowHeader(runAgent, runTask, streamState.sessionID, streamState.model, started,
		streamState.durationMs, streamState.numTurns, streamState.totalIn, streamState.totalOut)

	// If result is already present, we're done.
	if streamState.finalResult != "" {
		fmt.Println()
		printFollowResult(streamState.finalResult, streamState.isError)
		return nil
	}

	// ── Phase 2: tail for new lines ─────────────────────────────────────────
	return followTail(f, streamState, showThink, noTools, maxOut)
}

// followState tracks streaming parse state across events.
type followState struct {
	model      string
	sessionID  string
	totalIn    int
	totalOut   int
	numTurns   int
	durationMs int
	isError    bool
	finalResult string

	// Turn buffering
	pendingLines []string // accumulated lines for current turn
	turnCount    int
}

func (s *followState) flushLines(showThink, noTools bool, maxOut int) []string {
	if len(s.pendingLines) == 0 {
		return nil
	}
	lines := s.pendingLines
	s.pendingLines = nil
	_ = showThink
	_ = noTools
	_ = maxOut
	return lines
}

// streamParseEvent processes a single log event in streaming mode.
func streamParseEvent(ev logEvent, raw string, s *followState, showThink, noTools bool, maxOut int) {

	switch ev.Type {
	case "system":
		if ev.Subtype == "init" {
			var raw2 map[string]interface{}
			_ = json.Unmarshal([]byte(raw), &raw2)
			if m, ok := raw2["model"].(string); ok {
				s.model = m
			}
			if sid, ok := raw2["session_id"].(string); ok {
				s.sessionID = sid
			}
		}

	case "assistant":
		// Flush any previous turn first.
		if len(s.pendingLines) > 0 {
			for _, l := range s.pendingLines {
				fmt.Println(l)
			}
			s.pendingLines = nil
		}

		var msg msgEnvelope
		if err := json.Unmarshal(ev.Message, &msg); err != nil {
			return
		}
		if msg.Model != "" {
			s.model = msg.Model
		}
		if msg.Usage != nil {
			s.totalIn += msg.Usage.InputTokens
			s.totalOut += msg.Usage.OutputTokens
		}
		s.turnCount++
		s.numTurns++

		for _, c := range msg.Content {
			switch c.Type {
			case "thinking":
				if showThink && strings.TrimSpace(c.Thinking) != "" {
					s.pendingLines = append(s.pendingLines, formatThinking(c.Thinking))
				}
			case "text":
				if strings.TrimSpace(c.Text) != "" {
					s.pendingLines = append(s.pendingLines, formatAssistantText(c.Text))
				}
			case "tool_use":
				if !noTools {
					s.pendingLines = append(s.pendingLines, formatToolCall(c.Name, c.Input))
				}
			}
		}

	case "user":
		if noTools {
			return
		}
		var msg msgEnvelope
		if err := json.Unmarshal(ev.Message, &msg); err != nil {
			return
		}
		for _, c := range msg.Content {
			if c.Type == "tool_result" {
				s.pendingLines = append(s.pendingLines, formatToolResult(c.Content, maxOut))
			}
		}

	case "result":
		// Flush pending turn.
		if len(s.pendingLines) > 0 {
			for _, l := range s.pendingLines {
				fmt.Println(l)
			}
			s.pendingLines = nil
		}
		s.numTurns = ev.NumTurns
		s.durationMs = ev.DurationMs
		s.finalResult = ev.Result
		s.isError = ev.IsError
	}
}

// followTail keeps reading new lines from the open file and streaming them.
func followTail(f *os.File, s *followState, showThink, noTools bool, maxOut int) error {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Track last seen size to detect rotation.
	lastSize, _ := f.Seek(0, 1)

	for {
		<-ticker.C

		// Check for file rotation (size shrank).
		fi, err := f.Stat()
		if err != nil {
			return fmt.Errorf("stat log file: %w", err)
		}
		if fi.Size() < lastSize {
			// File was rotated — re-open and start over.
			f.Close()
			f2, err := os.Open(f.Name())
			if err != nil {
				return fmt.Errorf("re-open log file: %w", err)
			}
			f = f2
			lastSize = 0
		}

		// Seek to where we left off and scan new lines.
		_, _ = f.Seek(lastSize, 0)

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)

		newLines := false
		for scanner.Scan() {
			newLines = true
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "===") {
				continue
			}
			if !strings.HasPrefix(line, "{") {
				continue
			}
			var ev logEvent
			if err := json.Unmarshal([]byte(line), &ev); err != nil {
				continue
			}
			streamParseEvent(ev, line, s, showThink, noTools, maxOut)
		}

		if newLines {
			lastSize, _ = f.Seek(0, 1)
		}

		// If result arrived, flush and exit.
		if s.finalResult != "" {
			if len(s.pendingLines) > 0 {
				for _, l := range s.pendingLines {
					fmt.Println(l)
				}
				s.pendingLines = nil
			}
			fmt.Println()
			printFollowResult(s.finalResult, s.isError)
			return nil
		}
	}
}

func printFollowHeader(runAgent, runTask, sessionID, model, started string,
	durationMs, numTurns, totalIn, totalOut int) {

	width := 72
	fmt.Println(divider('═', width))
	if runAgent != "" {
		fmt.Printf("  Agent    : %s\n", runAgent)
	}
	if runTask != "" {
		fmt.Printf("  Task     : %s\n", runTask)
	}
	if sessionID != "" {
		fmt.Printf("  Session  : %s\n", sessionID)
	}
	if model != "" {
		fmt.Printf("  Model    : %s\n", model)
	}
	if started != "" {
		fmt.Printf("  Started  : %s\n", strings.TrimSpace(started))
	}
	if durationMs > 0 {
		fmt.Printf("  Duration : %s\n", fmtDurationMs(durationMs))
	}
	if numTurns > 0 {
		fmt.Printf("  Turns    : %d\n", numTurns)
	}
	if totalIn+totalOut > 0 {
		fmt.Printf("  Tokens   : %d in / %d out\n", totalIn, totalOut)
	}
	fmt.Println(divider('═', width))
	fmt.Println()
}

func printFollowResult(finalResult string, isError bool) {
	width := 72
	if isError {
		fmt.Println(divider('─', width))
		fmt.Println("  ✗ FAILED")
	} else {
		fmt.Println(divider('─', width))
		fmt.Println("  ✓ Result")
	}
	fmt.Println(divider('─', width))
	for _, l := range strings.Split(finalResult, "\n") {
		fmt.Printf("  %s\n", l)
	}
	fmt.Println(divider('─', width))
}

// ── format helpers (moved from tasklog.go) ───────────────────────────────────

func formatThinking(text string) string {
	var sb strings.Builder
	sb.WriteString("  ┆ [thinking]\n")
	for _, l := range strings.Split(strings.TrimSpace(text), "\n") {
		sb.WriteString(fmt.Sprintf("  ┆   %s\n", l))
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatAssistantText(text string) string {
	var sb strings.Builder
	sb.WriteString("  ◆ ")
	lines := strings.Split(strings.TrimSpace(text), "\n")
	sb.WriteString(lines[0] + "\n")
	for _, l := range lines[1:] {
		sb.WriteString("    " + l + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func formatToolCall(name string, inputRaw json.RawMessage) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  ▶ [%s]", name))

	var input map[string]interface{}
	if err := json.Unmarshal(inputRaw, &input); err == nil {
		detail := toolInputSummary(name, input)
		if detail != "" {
			sb.WriteString("  ")
			sb.WriteString(detail)
		}
	}
	return sb.String()
}

func toolInputSummary(tool string, input map[string]interface{}) string {
	switch tool {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			return truncStr(strings.TrimSpace(cmd), 120)
		}
	case "Read", "Write", "Edit", "NotebookEdit":
		if p, ok := input["path"].(string); ok {
			return p
		}
		if p, ok := input["file_path"].(string); ok {
			return p
		}
	case "Glob":
		if p, ok := input["glob_pattern"].(string); ok {
			return p
		}
	case "Grep":
		pat, _ := input["pattern"].(string)
		dir, _ := input["path"].(string)
		if dir != "" {
			return fmt.Sprintf("%q in %s", pat, dir)
		}
		return fmt.Sprintf("%q", pat)
	case "WebFetch", "WebSearch":
		if u, ok := input["url"].(string); ok {
			return truncStr(u, 100)
		}
		if q, ok := input["search_term"].(string); ok {
			return truncStr(q, 100)
		}
	case "Task", "TaskOutput":
		if d, ok := input["description"].(string); ok {
			return truncStr(d, 80)
		}
	case "TodoWrite":
		return ""
	}
	for _, v := range input {
		if s, ok := v.(string); ok && s != "" {
			return truncStr(s, 100)
		}
	}
	return ""
}

func formatToolResult(content interface{}, maxOut int) string {
	var text string
	switch v := content.(type) {
	case string:
		text = v
	case []interface{}:
		var parts []string
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if t, ok := m["text"].(string); ok {
					parts = append(parts, t)
				}
			}
		}
		text = strings.Join(parts, "\n")
	default:
		b, _ := json.Marshal(content)
		text = string(b)
	}

	text = strings.TrimSpace(text)
	if maxOut > 0 && utf8.RuneCountInString(text) > maxOut {
		runes := []rune(text)
		text = string(runes[:maxOut]) + fmt.Sprintf("… [+%d chars]", utf8.RuneCountInString(text)-maxOut)
	}

	var sb strings.Builder
	sb.WriteString("  ╰ ")
	lines := strings.Split(text, "\n")
	sb.WriteString(lines[0] + "\n")
	for _, l := range lines[1:] {
		if strings.TrimSpace(l) == "" {
			continue
		}
		sb.WriteString("    " + l + "\n")
	}
	return strings.TrimRight(sb.String(), "\n")
}

func divider(ch rune, width int) string {
	return strings.Repeat(string(ch), width)
}

func truncStr(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func fmtDurationMs(ms int) string {
	d := time.Duration(ms) * time.Millisecond
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%02ds", m, s)
}
