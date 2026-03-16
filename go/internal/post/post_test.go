package post

import (
	"strings"
	"testing"
	"time"
)

func TestGetWeekBounds(t *testing.T) {
	date := time.Date(2026, time.March, 18, 10, 0, 0, 0, time.UTC)
	bounds := GetWeekBounds(date)

	if bounds.WeekNumber != 12 {
		t.Fatalf("expected week 12, got %d", bounds.WeekNumber)
	}
	if got := bounds.Monday.Format("2006-01-02"); got != "2026-03-16" {
		t.Fatalf("expected Monday 2026-03-16, got %s", got)
	}
	if got := bounds.Sunday.Format("2006-01-02"); got != "2026-03-22" {
		t.Fatalf("expected Sunday 2026-03-22, got %s", got)
	}
}

func TestMondayOfISOWeek(t *testing.T) {
	monday := MondayOfISOWeek(2026, 1)
	if got := monday.Format("2006-01-02"); got != "2025-12-29" {
		t.Fatalf("expected 2025-12-29, got %s", got)
	}
}

func TestBuildPostTextBelowPace(t *testing.T) {
	text := BuildPostText(12, 53, 13.2, 1248.6, 12000)

	if !strings.Contains(text, "Semana 12/53") {
		t.Fatalf("expected week line in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "abaixo do ritmo") {
		t.Fatalf("expected below pace message, got:\n%s", text)
	}
}
