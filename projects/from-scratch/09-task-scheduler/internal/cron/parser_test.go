package cron_test

import (
	"testing"
	"time"

	"tour_of_go/projects/from-scratch/09-task-scheduler/internal/cron"
)

func mustParse(t *testing.T, expr string) *cron.Schedule {
	t.Helper()
	s, err := cron.Parse(expr)
	if err != nil {
		t.Fatalf("parse %q: %v", expr, err)
	}
	return s
}

func TestParse_EveryMinute(t *testing.T) {
	s := mustParse(t, "* * * * *")
	for m := 0; m < 60; m++ {
		ts := time.Date(2024, 1, 1, 0, m, 0, 0, time.UTC)
		if !s.Match(ts) {
			t.Fatalf("* * * * * should match minute %d", m)
		}
	}
}

func TestParse_EveryFiveMinutes(t *testing.T) {
	s := mustParse(t, "*/5 * * * *")
	for m := 0; m < 60; m++ {
		ts := time.Date(2024, 1, 1, 0, m, 0, 0, time.UTC)
		want := m%5 == 0
		if s.Match(ts) != want {
			t.Fatalf("*/5 at minute %d: want %v", m, want)
		}
	}
}

func TestParse_SpecificTime(t *testing.T) {
	s := mustParse(t, "30 9 * * *") // 9:30 every day
	match := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	noMatch := time.Date(2024, 1, 1, 9, 31, 0, 0, time.UTC)
	if !s.Match(match) {
		t.Fatal("should match 9:30")
	}
	if s.Match(noMatch) {
		t.Fatal("should not match 9:31")
	}
}

func TestParse_InvalidExpr(t *testing.T) {
	if _, err := cron.Parse("* * *"); err == nil {
		t.Fatal("expected error for 3-field expr")
	}
}
