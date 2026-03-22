package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/spf13/cobra"
)

// ── ANSI colour helpers ───────────────────────────────────────────────────
//
// Optimised for dark terminals. Never use \033[2m (dim) — it's invisible on
// many themes. \033[90m is used only for purely decorative separator chars.

const (
	ansiReset   = "\033[0m"
	ansiBold    = "\033[1m"
	ansiCyan    = "\033[36m"
	ansiBCyan   = "\033[1;36m"
	ansiGreen   = "\033[32m"
	ansiBGreen  = "\033[1;32m"
	ansiYellow  = "\033[33m"
	ansiBYellow = "\033[1;33m"
	ansiBlue    = "\033[34m"
	ansiMagenta = "\033[35m"
	ansiWhite   = "\033[97m"
	ansiSilver  = "\033[37m" // visible secondary text on dark bg
	ansiMuted   = "\033[90m" // decorative separators only
)

func col(code, s string) string { return code + s + ansiReset }
func bold(s string) string      { return ansiBold + s + ansiReset }
func silver(s string) string    { return ansiSilver + s + ansiReset }
func muted(s string) string     { return ansiMuted + s + ansiReset }

// runeWidth returns the terminal column width of a single rune.
// CJK and other East Asian wide characters occupy 2 columns; all others occupy 1.
func runeWidth(r rune) int {
	switch {
	case r < 0x20 || (r >= 0x7F && r < 0xA0):
		return 0 // control characters
	case r < 0x1100:
		return 1
	case r <= 0x115F, // Hangul Jamo
		r == 0x2329 || r == 0x232A,
		r >= 0x2E80 && r <= 0x303E,  // CJK Radicals, CJK Symbols & Punctuation
		r >= 0x3040 && r <= 0x33FF,  // Hiragana, Katakana, Bopomofo, Hangul Compat, CJK Compat
		r >= 0x3400 && r <= 0x4DBF,  // CJK Extension A
		r >= 0x4E00 && r <= 0xA4CF,  // CJK Unified Ideographs + Yi
		r >= 0xA960 && r <= 0xA97F,  // Hangul Jamo Extended-A
		r >= 0xAC00 && r <= 0xD7AF,  // Hangul Syllables
		r >= 0xF900 && r <= 0xFAFF,  // CJK Compatibility Ideographs
		r >= 0xFE10 && r <= 0xFE19,  // Vertical Forms
		r >= 0xFE30 && r <= 0xFE6F,  // CJK Compat Forms, Small Form Variants
		r >= 0xFF01 && r <= 0xFF60,  // Fullwidth ASCII
		r >= 0xFFE0 && r <= 0xFFE6,  // Fullwidth Signs
		r >= 0x1B000 && r <= 0x1B001,
		r >= 0x1F004 && r <= 0x1F004,
		r >= 0x1F0CF && r <= 0x1F0CF,
		r >= 0x1F200 && r <= 0x1F251,
		r >= 0x20000 && r <= 0x2FFFD,
		r >= 0x30000 && r <= 0x3FFFD:
		return 2
	default:
		return 1
	}
}

// visibleLen returns the terminal column width of s, ignoring ANSI escape sequences
// and correctly counting wide (CJK) characters as 2 columns.
// Box-drawing characters (U+2500–U+257F) also render as double-width in terminals.
func visibleLen(s string) int {
	n := 0
	inEsc := false
	for _, r := range s {
		switch {
		case inEsc:
			if r == 'm' {
				inEsc = false
			}
		case r == '\033':
			inEsc = true
		default:
			w := runeWidth(r)
			// Override: box-drawing and geometric shapes render as 2 columns in terminals
			// even though Unicode EastAsianWidth says "narrow" (U+2500–U+257F, U+25A0–U+25FF).
			if r >= 0x2500 && r <= 0x257F {
				w = 2
			}
			n += w
		}
	}
	return n
}

// padV pads s to exactly w terminal columns.
func padV(s string, w int) string {
	vis := visibleLen(s)
	if vis >= w {
		return s
	}
	return s + strings.Repeat(" ", w-vis)
}

// padStr pads/truncates s to n bytes (ASCII-safe; use for fixed-width ASCII names).
func padStr(s string, n int) string {
	if len(s) >= n {
		return s[:n]
	}
	return s + strings.Repeat(" ", n-len(s))
}

// truncate cuts s so the result (including the trailing "…") fits in maxCols
// terminal columns.  "…" counts as 1 column, so content is allowed maxCols-1.
func truncate(s string, maxCols int) string {
	cols := 0
	for i, r := range s {
		w := runeWidth(r)
		// Stop as soon as adding this rune would leave no room for "…".
		if cols+w >= maxCols {
			if i == 0 {
				return "…"
			}
			return s[:i] + "…"
		}
		cols += w
	}
	return s // fits without truncation
}

// ── box drawing ───────────────────────────────────────────────────────────
//
// Total visible width of every line: boxW + 4
//   ╭─  inner(boxW)  ─╮   → boxW + 4
//   ├─────(boxW+2)─────┤   → boxW + 4
//   │  content(boxW)  │   → 1 + 1 + boxW + 1 + 1 = boxW + 4

const boxW = 74

func boxTop(title, right string) string {
	mid := fmt.Sprintf(" ◈ %s ", title)
	rightSeg := fmt.Sprintf(" %s ", right)
	dashes := boxW - visibleLen(mid) - visibleLen(rightSeg)
	if dashes < 2 {
		dashes = 2
	}
	inner := mid + strings.Repeat("─", dashes) + rightSeg
	return col(ansiBCyan, "╭─"+inner+"─╮")
}

func boxSep() string { return col(ansiCyan, "├"+strings.Repeat("─", boxW+2)+"┤") }
func boxBot() string { return col(ansiCyan, "╰"+strings.Repeat("─", boxW+2)+"╯") }

// boxRow wraps content in │ bars. Content is padded to exactly boxW visible chars.
func boxRow(content string) string {
	return col(ansiCyan, "│") + " " + padV(content, boxW) + " " + col(ansiCyan, "│")
}

func boxBlank() string {
	return col(ansiCyan, "│") + strings.Repeat(" ", boxW+2) + col(ansiCyan, "│")
}

func secHeader(title string) string {
	return boxRow(col(ansiBCyan, title))
}

// ── progress bar ──────────────────────────────────────────────────────────

func progressBar(fraction float64, width int) string {
	filled := int(math.Round(fraction * float64(width)))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	if fraction >= 0.85 {
		return col(ansiYellow, bar)
	}
	return col(ansiGreen, bar)
}

// ── heartbeat snapshot ────────────────────────────────────────────────────

type hbSnap struct {
	enabled     bool
	intervalStr string
	interval    time.Duration
	nextWakeup  time.Time
	fraction    float64
	isRunning   bool
}

func loadHBSnap(ts taskstore.Store, project, agent string) hbSnap {
	hb, err := ts.GetHeartbeat(project, agent)
	if err != nil || hb == nil || !hb.Enabled {
		return hbSnap{}
	}
	ivl, err := time.ParseDuration(hb.Interval)
	if err != nil || ivl <= 0 {
		return hbSnap{enabled: true, intervalStr: hb.Interval}
	}
	snap := hbSnap{
		enabled:     true,
		interval:    ivl,
		intervalStr: fmtInterval(ivl),
		isRunning:   hb.LastWakeupStatus == "running",
	}
	if hb.LastWakeup != nil {
		elapsed := time.Since(*hb.LastWakeup)
		snap.nextWakeup = hb.LastWakeup.Add(ivl)
		snap.fraction = math.Min(elapsed.Seconds()/ivl.Seconds(), 1.0)
	}
	return snap
}

func fmtInterval(d time.Duration) string {
	switch {
	case d >= time.Hour && d%time.Hour == 0:
		return fmt.Sprintf("%dh", int(d.Hours()))
	case d >= time.Minute && d%time.Minute == 0:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	default:
		return d.String()
	}
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	if d < 0 {
		d = 0
	}
	switch {
	case d >= time.Hour:
		return fmt.Sprintf("%dh%02dm", int(d.Hours()), int(d.Minutes())%60)
	case d >= time.Minute:
		return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
	default:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// ── agent row renderer ────────────────────────────────────────────────────

// renderAgentRow builds an agent-status line and returns exactly boxW visible
// characters by measuring each component's visible width.
func renderAgentRow(name string, meta *entity.AgentMeta, snap hbSnap, taskCount int) string {
	modelVis := 10
	modelStr := silver(padStr("—", modelVis))
	if meta != nil {
		modelStr = silver(padStr(string(meta.Model), modelVis))
	}

	nameVis := 16
	nameStr := bold(padStr(name, nameVis))

	statusIcon := silver("○")
	schedStr := silver("no heartbeat")
	schedVis := 12 // "no heartbeat" visible length

	switch {
	case !snap.enabled:
		// already set above

	case snap.isRunning:
		statusIcon = col(ansiBGreen, "▶")
		schedStr = col(ansiBGreen, "running…")
		schedVis = 8

	case snap.interval == 0:
		statusIcon = silver("○")
		schedStr = silver("↻" + snap.intervalStr)
		schedVis = 1 + visibleLen(snap.intervalStr)

	default:
		statusIcon = col(ansiGreen, "▶")
		intStr := muted("↻") + col(ansiCyan, snap.intervalStr)
		intVis := 1 + visibleLen(snap.intervalStr) // "↻" + interval
		bar := progressBar(snap.fraction, 10)
		barVis := 10
		nextStr := ""
		nextVis := 0
		if !snap.nextWakeup.IsZero() {
			remaining := time.Until(snap.nextWakeup)
			timeStr := snap.nextWakeup.Format("15:04:05")
			durStr := fmtDuration(remaining)
			nextStr = muted("→ ") + col(ansiWhite, timeStr) + silver(" ("+durStr+")")
			nextVis = 2 + visibleLen(timeStr) + 2 + visibleLen(durStr) + 2 // "→ " + time + " (" + dur + ")"
		}
		schedStr = intStr + "  " + bar + "  " + nextStr
		schedVis = intVis + 2 + barVis + 2 + nextVis
	}

	taskTag := ""
	taskTagVis := 0
	if taskCount > 0 {
		taskTag = "  " + col(ansiBYellow, fmt.Sprintf("[%d task%s]", taskCount, plural(taskCount)))
		taskTagVis = 2 + visibleLen(fmt.Sprintf("[%d task%s]", taskCount, plural(taskCount)))
	}

	// Build total by concatenation and measure
	row := fmt.Sprintf("    %s  %s  %s  %s%s", statusIcon, nameStr, modelStr, schedStr, taskTag)
	rowVis := 4 + visibleLen(statusIcon) + 2 + nameVis + 2 + modelVis + 2 + schedVis + taskTagVis

	// Pad to boxW if needed
	if rowVis < boxW {
		row += strings.Repeat(" ", boxW-rowVis)
	}

	return row
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// ── inbox footer ──────────────────────────────────────────────────────────

func renderInboxLine(confirms, unread int) string {
	icon := silver("✉  ")
	if confirms > 0 || unread > 0 {
		icon = col(ansiBYellow, "✉  ")
	}

	var parts []string
	if confirms > 0 {
		parts = append(parts, col(ansiYellow, fmt.Sprintf("%d confirmation%s pending", confirms, plural(confirms))))
	} else {
		parts = append(parts, silver("inbox clear"))
	}
	if unread > 0 {
		parts = append(parts, col(ansiCyan, fmt.Sprintf("%d unread message%s", unread, plural(unread))))
	}
	return icon + strings.Join(parts, muted("  ·  "))
}

// ── command ───────────────────────────────────────────────────────────────

func newOverviewCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "overview",
		Aliases: []string{"status", "stat"},
		Short:   "Show a dashboard overview of the agency",
		RunE:    runOverview,
	}
}

func runOverview(_ *cobra.Command, _ []string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}

	s  := store.NewFS(root)
	ts := taskstore.New(root)

	agency, err := s.Agency()
	if err != nil {
		return err
	}

	// ── gather agent data ───────────────────────────────────────────────
	type agentData struct {
		project   string
		name      string
		meta      *entity.AgentMeta
		snap      hbSnap
		taskCount int
	}

	projects, _ := ts.ListProjects()
	var allAgents []agentData

	for _, proj := range projects {
		agentNames, _ := ts.ListAgents(proj)
		for _, agName := range agentNames {
			if len(agName) > 0 && agName[0] == '.' {
				continue
			}
			meta, _ := s.AgentMeta(proj, agName)
			snap := loadHBSnap(ts, proj, agName)

			tasks, _ := ts.ListTasks(proj, agName)
			pending := 0
			for _, t := range tasks {
				if t.Status == entity.TaskStatusPending || t.Status == entity.TaskStatusInProgress {
					pending++
				}
			}
			allAgents = append(allAgents, agentData{
				project:   proj,
				name:      agName,
				meta:      meta,
				snap:      snap,
				taskCount: pending,
			})
		}
	}

	// ── inbox ───────────────────────────────────────────────────────────
	inboxItems, _ := ts.ListInbox()
	unreadMsgs, _ := ts.ListUnreadMessages("human")

	// ── teams & skills ──────────────────────────────────────────────────
	teamEntries, _ := s.ListTeams()
	skills, _ := s.ListSkills()

	// ── render ──────────────────────────────────────────────────────────
	agWord := "agents"
	if len(allAgents) == 1 {
		agWord = "agent"
	}
	rightLabel := fmt.Sprintf("%d %s · %d skill%s", len(allAgents), agWord, len(skills), plural(len(skills)))

	fmt.Println(boxTop(agency.Name, rightLabel))
	fmt.Println(boxBlank())

	// AGENTS
	fmt.Println(secHeader("AGENTS"))
	fmt.Println(boxBlank())

	if len(allAgents) == 0 {
		fmt.Println(boxRow(silver("  no agents hired yet — run: agencycli hire --help")))
	} else {
		curProj := ""
		for _, a := range allAgents {
			if a.project != curProj {
				curProj = a.project
				fmt.Println(boxRow(col(ansiSilver, "  "+curProj)))
			}
			fmt.Println(boxRow(renderAgentRow(a.name, a.meta, a.snap, a.taskCount)))
		}
	}
	fmt.Println(boxBlank())

	// TEAMS & SKILLS
	if len(teamEntries) > 0 || len(skills) > 0 {
		fmt.Println(boxSep())
		fmt.Println(boxBlank())

		if len(teamEntries) > 0 {
			fmt.Println(secHeader("TEAMS & ROLES"))
			const teamNameW = 14
			// prefix visible cols: "  " (2) + "◆" (1) + " " (1) + name (teamNameW) + "  " (2) = 20
			const teamPrefixCols = 2 + 1 + 1 + teamNameW + 2
			for _, te := range teamEntries {
				roles, _ := s.ListRoles(te.Path)
				roleNames := make([]string, 0, len(roles))
				for _, r := range roles {
					roleNames = append(roleNames, r.Name)
				}

				skillTag := ""
				if len(te.Team.Skills) > 0 {
					skillTag = silver("  [" + strings.Join(te.Team.Skills, ", ") + "]")
				}
				skillCols := visibleLen(skillTag)

				maxRolesCols := boxW - teamPrefixCols - skillCols
				if maxRolesCols < 3 {
					maxRolesCols = 3
				}

				rolesStr := silver("no roles")
				if len(roleNames) > 0 {
					joined := strings.Join(roleNames, " · ")
					if visibleLen(joined) <= maxRolesCols {
						parts := make([]string, len(roleNames))
						for i, r := range roleNames {
							parts[i] = col(ansiWhite, r)
						}
						rolesStr = strings.Join(parts, muted(" · "))
					} else {
						rolesStr = silver(truncate(joined, maxRolesCols))
					}
				}

				line := fmt.Sprintf("  %s %s  %s%s",
					col(ansiMagenta, "◆"),
					bold(padStr(te.Path, teamNameW)),
					rolesStr,
					skillTag,
				)
				fmt.Println(boxRow(line))
			}
			fmt.Println(boxBlank())
		}

		if len(skills) > 0 {
			fmt.Println(secHeader("SKILLS"))
			// "  ⬡ " (4) + name(22) + "  " (2) = 28 prefix visible chars
			const prefixW = 28
			descMaxW := boxW - prefixW - 2
			for _, sk := range skills {
				desc := silver("—")
				if sk.Description != "" {
					desc = silver(truncate(sk.Description, descMaxW))
				}
				fmt.Println(boxRow(fmt.Sprintf("  %s %-22s  %s",
					col(ansiBlue, "⬡"),
					bold(padStr(sk.Name, 22)),
					desc,
				)))
			}
			fmt.Println(boxBlank())
		}
	}

	// INBOX footer
	fmt.Println(boxSep())
	fmt.Println(boxRow(renderInboxLine(len(inboxItems), len(unreadMsgs))))
	fmt.Println(boxBot())

	return nil
}

