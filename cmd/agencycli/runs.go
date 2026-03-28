package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/chenhg5/agencycli/internal/store"
	"github.com/chenhg5/agencycli/internal/taskstore"
	"github.com/chenhg5/agencycli/internal/telemetry"
	"github.com/spf13/cobra"
)

func newRunsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "runs",
		Short: "Telemetry from recorded agent invocations (SQLite)",
		Long: `Inspect agent runs stored in .agencycli/agencycli.db (token/cost/status and log paths).

Time window flags (--since / --until) filter on run start time (UTC in DB, you can use local dates).

Examples:
  agencycli runs summary                          # today 00:00 local → now
  agencycli runs summary --since 7d               # last 7 local midnights → now
  agencycli runs summary --since 2026-03-01 --until 2026-03-28
  agencycli runs summary --all-time
  agencycli runs agents --since today --project my-api`,
	}
	cmd.AddCommand(newRunsSummaryCmd(), newRunsAgentsCmd())
	return cmd
}

func newRunsSummaryCmd() *cobra.Command {
	var (
		since, until string
		allTime      bool
		project      string
	)
	cmd := &cobra.Command{
		Use:     "summary",
		Aliases: []string{"overview", "ov"},
		Short:   "Aggregate run stats for a time window (overview-style)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRunsSummary(since, until, allTime, project)
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "window start: all, today, yesterday, 24h, 7d, 2006-01-02, RFC3339 (default with empty --until: start of today local)")
	cmd.Flags().StringVar(&until, "until", "", "window end: now, today, yesterday, date, RFC3339 (default: now)")
	cmd.Flags().BoolVar(&allTime, "all-time", false, "entire history (ignore --since/--until)")
	cmd.Flags().StringVar(&project, "project", "", "limit to one project")
	return cmd
}

func newRunsAgentsCmd() *cobra.Command {
	var (
		since, until string
		allTime      bool
		project      string
	)
	cmd := &cobra.Command{
		Use:     "agents",
		Aliases: []string{"by-agent"},
		Short:   "Per-agent run stats for the workspace (all hired agents, including zeros)",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRunsAgents(since, until, allTime, project)
		},
	}
	cmd.Flags().StringVar(&since, "since", "", "same as runs summary")
	cmd.Flags().StringVar(&until, "until", "", "same as runs summary")
	cmd.Flags().BoolVar(&allTime, "all-time", false, "entire history")
	cmd.Flags().StringVar(&project, "project", "", "only agents under this project")
	return cmd
}

func runRunsSummary(since, until string, allTime bool, project string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	from, to, err := telemetry.ParseWindow(since, until, allTime, time.Now(), time.Local)
	if err != nil {
		return err
	}

	db, err := telemetry.OpenReadOnly(root)
	if err != nil {
		if err == telemetry.ErrNoDatabase {
			return fmt.Errorf("%w\n  (execute tasks or agencycli exec to generate telemetry)", err)
		}
		return err
	}
	defer db.Close()

	rows, err := telemetry.ReadRuns(db, from, to, project)
	if err != nil {
		return err
	}
	sum := telemetry.Summarize(rows)

	s := store.NewFS(root)
	ag, err := s.Agency()
	if err != nil {
		return err
	}

	right := windowLabel(from, to, allTime)
	fmt.Println(boxTop(ag.Name+" · run telemetry", right))
	fmt.Println(boxBlank())
	fmt.Println(secHeader("WINDOW"))
	fmt.Println(boxRow(silver("  "+right)))
	fmt.Println(boxBlank())

	if sum.Runs == 0 {
		fmt.Println(boxRow(silver("  no runs in this window")))
		fmt.Println(boxBot())
		return nil
	}

	fmt.Println(secHeader("VOLUME"))
	fmt.Println(boxRow(fmt.Sprintf("  %s  %s  ·  task %s  ·  exec %s",
		col(ansiWhite, fmt.Sprintf("%d runs", sum.Runs)),
		silver("·"),
		col(ansiCyan, fmt.Sprintf("%d", sum.TaskRuns)),
		col(ansiMagenta, fmt.Sprintf("%d", sum.ExecRuns)),
	)))
	fmt.Println(boxRow(silver("  wall time (sum of run spans)  "+formatDurationHuman(sum.WallDuration))))
	fmt.Println(boxBlank())

	fmt.Println(secHeader("TOKENS (aggregated)"))
	fmt.Println(boxRow(fmt.Sprintf("  %s  in  %s  out  %s  cache read",
		col(ansiGreen, formatTokens(sum.InputTokens)),
		col(ansiGreen, formatTokens(sum.OutputTokens)),
		col(ansiYellow, formatTokens(sum.CacheReadTokens)),
	)))
	fmt.Println(boxBlank())

	costLine := fmt.Sprintf("  %s  reported API total  (%s runs with usage block)",
		col(ansiBYellow, fmt.Sprintf("$%.4f", sum.CostUSD)),
		col(ansiWhite, fmt.Sprintf("%d", sum.RunsWithCost)),
	)
	fmt.Println(secHeader("COST"))
	fmt.Println(boxRow(costLine))
	fmt.Println(boxBlank())

	fmt.Println(secHeader("STATUS"))
	fmt.Println(boxRow(fmt.Sprintf("  %s  ok   %s  fail   %s  confirm   %s  other",
		col(ansiBGreen, fmt.Sprintf("%d", sum.Success)),
		col(ansiBYellow, fmt.Sprintf("%d", sum.Failed)),
		col(ansiYellow, fmt.Sprintf("%d", sum.Awaiting)),
		silver(fmt.Sprintf("%d", sum.Other)),
	)))
	fmt.Println(boxBot())
	return nil
}

func runRunsAgents(since, until string, allTime bool, project string) error {
	root, err := resolveRoot()
	if err != nil {
		return err
	}
	from, to, err := telemetry.ParseWindow(since, until, allTime, time.Now(), time.Local)
	if err != nil {
		return err
	}

	db, err := telemetry.OpenReadOnly(root)
	if err != nil {
		if err == telemetry.ErrNoDatabase {
			return fmt.Errorf("%w\n  (execute tasks or agencycli exec to generate telemetry)", err)
		}
		return err
	}
	defer db.Close()

	rows, err := telemetry.ReadRuns(db, from, to, project)
	if err != nil {
		return err
	}
	bySlice := telemetry.SummarizeByAgent(rows)
	stats := make(map[string]telemetry.AgentSummary, len(bySlice))
	for _, a := range bySlice {
		stats[a.Project+"/"+a.Agent] = a
	}

	ts := taskstore.New(root)
	s := store.NewFS(root)
	ag, err := s.Agency()
	if err != nil {
		return err
	}

	projects, err := ts.ListProjects()
	if err != nil {
		return err
	}
	if project != "" {
		found := false
		for _, p := range projects {
			if p == project {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("unknown project %q", project)
		}
		projects = []string{project}
	}

	type hired struct {
		proj, name string
	}
	var agents []hired
	for _, proj := range projects {
		names, err := ts.ListAgents(proj)
		if err != nil {
			continue
		}
		for _, n := range names {
			if len(n) > 0 && n[0] == '.' {
				continue
			}
			agents = append(agents, hired{proj, n})
		}
	}

	right := windowLabel(from, to, allTime)
	fmt.Println(boxTop(ag.Name+" · agents / runs", right))
	fmt.Println(boxBlank())
	fmt.Println(secHeader("WINDOW"))
	fmt.Println(boxRow(silver("  " + right)))
	fmt.Println(boxBlank())

	if len(agents) == 0 {
		fmt.Println(boxRow(silver("  no hired agents — run: agencycli hire --help")))
		fmt.Println(boxBot())
		return nil
	}

	// Wide table below the box (tabwriter does not fit boxW).
	fmt.Println(boxBot())
	fmt.Println()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "PROJECT\tAGENT\tRUNS\tTASK\tEXEC\tOK\tFAIL\tINPUT\tOUTPUT\tCACHE\tCOST\tWALL")
	fmt.Fprintln(w, "───────\t─────\t────\t────\t────\t──\t────\t─────\t──────\t─────\t────\t────")
	for _, h := range agents {
		key := h.proj + "/" + h.name
		st, ok := stats[key]
		if !ok {
			fmt.Fprintf(w, "%s\t%s\t0\t0\t0\t0\t0\t0\t0\t0\t$0.0000\t0s\n",
				h.proj, h.name)
			continue
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%s\t%s\t%s\t%s\t%s\n",
			st.Project, st.Agent,
			st.Runs, st.Task, st.Exec,
			st.Success, st.Failed,
			formatTokens(st.InputTokens),
			formatTokens(st.OutputTokens),
			formatTokens(st.CacheReadTokens),
			formatCostCol(st.CostUSD, st.RunsWithCost > 0),
			formatDurationHuman(st.WallDuration),
		)
	}
	_ = w.Flush()
	return nil
}

func windowLabel(from, to *time.Time, allTime bool) string {
	if allTime {
		return "all time"
	}
	loc := time.Local
	var parts []string
	if from == nil {
		parts = append(parts, "…")
	} else {
		parts = append(parts, from.In(loc).Format("2006-01-02 15:04 MST"))
	}
	parts = append(parts, "→")
	if to == nil {
		parts = append(parts, "…")
	} else {
		parts = append(parts, to.In(loc).Format("2006-01-02 15:04 MST"))
	}
	return strings.Join(parts, " ")
}

func formatDurationHuman(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}
	if d < time.Minute {
		return d.Round(time.Second).String()
	}
	if d < time.Hour {
		return (d / time.Second * time.Second).String()
	}
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	if h >= 48 {
		return fmt.Sprintf("%dh", int(h))
	}
	return fmt.Sprintf("%dh%dm", int(h), int(m))
}

func formatCostCol(usd float64, has bool) string {
	if !has || usd == 0 {
		return "—"
	}
	return fmt.Sprintf("$%.4f", usd)
}
