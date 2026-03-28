package telemetry

import (
	"fmt"
	"strings"
	"time"
)

// ParseWindow resolves CLI --since / --until into bounds for filtering on started_at (stored as UTC RFC3339Nano).
//
// Rules:
//   - If allTime: both bounds are nil (no filter).
//   - If since and until are both empty and !allTime: [local start of today, now] in UTC.
//   - If only since is set: lower bound from parseSince; upper bound is now (UTC).
//   - If only until is set: lower bound is nil (full history); upper bound from parseUntil.
//   - If both set: use both; error if since > until.
func ParseWindow(since, until string, allTime bool, now time.Time, loc *time.Location) (from, to *time.Time, err error) {
	if allTime {
		return nil, nil, nil
	}
	if loc == nil {
		loc = time.Local
	}
	if since == "" && until == "" {
		st := startOfDay(now, loc).UTC()
		en := now.UTC()
		return &st, &en, nil
	}

	var fromPtr, toPtr *time.Time
	if since != "" {
		fromPtr, err = parseSince(since, now, loc)
		if err != nil {
			return nil, nil, err
		}
	}
	if until != "" {
		toPtr, err = parseUntil(until, now, loc)
		if err != nil {
			return nil, nil, err
		}
	}
	if toPtr == nil {
		t := now.UTC()
		toPtr = &t
	}
	if fromPtr != nil && toPtr != nil && fromPtr.After(*toPtr) {
		return nil, nil, fmt.Errorf("since is after until")
	}
	return fromPtr, toPtr, nil
}

func startOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, loc)
}

func endOfDay(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	y, m, d := t.Date()
	return time.Date(y, m, d, 23, 59, 59, 999999999, loc)
}

func parseSince(s string, now time.Time, loc *time.Location) (*time.Time, error) {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	switch low {
	case "all":
		return nil, nil
	case "today":
		t := startOfDay(now, loc).UTC()
		return &t, nil
	case "yesterday":
		t := startOfDay(now, loc).AddDate(0, 0, -1).UTC()
		return &t, nil
	}
	if d, err := time.ParseDuration(s); err == nil {
		if d <= 0 {
			return nil, fmt.Errorf("duration for --since must be positive (e.g. 24h or 168h)")
		}
		t := now.Add(-d).UTC()
		return &t, nil
	}
	// Calendar-day suffix: 7d → start of the day (now - n days) in local time
	if strings.HasSuffix(low, "d") {
		var n int
		if _, err := fmt.Sscanf(low, "%dd", &n); err == nil && n > 0 {
			day := startOfDay(now, loc).AddDate(0, 0, -n)
			u := day.UTC()
			return &u, nil
		}
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		u := startOfDay(t, loc).UTC()
		return &u, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		u := t.UTC()
		return &u, nil
	}
	return nil, fmt.Errorf("invalid --since %q: use all, today, yesterday, 24h, 7d, 2006-01-02, or RFC3339", s)
}

func parseUntil(s string, now time.Time, loc *time.Location) (*time.Time, error) {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	switch low {
	case "now":
		t := now.UTC()
		return &t, nil
	case "today":
		t := endOfDay(now, loc).UTC()
		return &t, nil
	case "yesterday":
		t := endOfDay(now.AddDate(0, 0, -1), loc).UTC()
		return &t, nil
	}
	if t, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		u := endOfDay(t, loc).UTC()
		return &u, nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		u := t.UTC()
		return &u, nil
	}
	return nil, fmt.Errorf("invalid --until %q: use now, today, yesterday, 2006-01-02, or RFC3339", s)
}
