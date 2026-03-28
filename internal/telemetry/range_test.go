package telemetry

import (
	"testing"
	"time"
)

func TestParseWindowDefaultToday(t *testing.T) {
	loc := time.FixedZone("test", 8*3600)
	now := time.Date(2026, 3, 28, 15, 30, 0, 0, loc)
	from, to, err := ParseWindow("", "", false, now, loc)
	if err != nil {
		t.Fatal(err)
	}
	if from == nil || to == nil {
		t.Fatal("expected bounds")
	}
	wantStart := time.Date(2026, 3, 28, 0, 0, 0, 0, loc).UTC()
	if !from.Equal(wantStart) {
		t.Fatalf("from=%v want %v", from, wantStart)
	}
	if !to.Equal(now.UTC()) {
		t.Fatalf("to=%v want %v", to, now.UTC())
	}
}

func TestParseWindowAllTime(t *testing.T) {
	from, to, err := ParseWindow("", "", true, time.Now(), time.Local)
	if err != nil {
		t.Fatal(err)
	}
	if from != nil || to != nil {
		t.Fatalf("expected nil bounds, got %v %v", from, to)
	}
}

func TestParseSince7d(t *testing.T) {
	loc := time.Local
	now := time.Date(2026, 3, 28, 12, 0, 0, 0, loc)
	from, err := parseSince("7d", now, loc)
	if err != nil {
		t.Fatal(err)
	}
	want := startOfDay(now, loc).AddDate(0, 0, -7).UTC()
	if !from.Equal(want) {
		t.Fatalf("from=%v want %v", from, want)
	}
}
