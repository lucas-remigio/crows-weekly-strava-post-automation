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
	text := BuildPostText(12, 53, 13.2, 1248.6, 12000, map[string]float64{
		"Run":  10.2,
		"Walk": 3.0,
	}, map[string]float64{
		"Alice": 10.2,
		"Bob":   3.0,
	})

	if !strings.Contains(text, "Semana 12/53") {
		t.Fatalf("expected week line in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "Por modalidade:") {
		t.Fatalf("expected by-sport header in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "├─ 🏃 Corrida: 10.2 km") {
		t.Fatalf("expected tree line for walk in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "└─ 🚶 Caminhada: 3.0 km") {
		t.Fatalf("expected tree line for run in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "abaixo do ritmo") {
		t.Fatalf("expected below pace message, got:\n%s", text)
	}
}

func TestFormatWeeklyByAthleteLines(t *testing.T) {
	athletes := map[string]float64{
		"Bob":     10.5, // 3rd Bronze
		"Charlie": 5.2,  // 4th Poop
		"Alice":   42.0, // 1st Gold
		"Eve":     1.0,  // 5th Poop
		"Dave":    21.0, // 2nd Silver
	}

	lines := formatWeeklyByAthleteLines(athletes)

	if len(lines) != 6 { // Header + 5 athletes
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}

	if !strings.Contains(lines[1], "🥇 Alice") {
		t.Errorf("expected Gold for Alice, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "🥈 Dave") {
		t.Errorf("expected Silver for Dave, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "🥉 Bob") {
		t.Errorf("expected Bronze for Bob, got: %s", lines[3])
	}
	if !strings.Contains(lines[4], "💩 Charlie") {
		t.Errorf("expected Poop for Charlie, got: %s", lines[4])
	}
	if !strings.Contains(lines[5], "└─ 💩 Eve: 1.0 km") {
		t.Errorf("expected Poop for Eve and ending branch, got: %s", lines[5])
	}
}

func TestFormatWeeklyByAthleteLinesEmpty(t *testing.T) {
	lines := formatWeeklyByAthleteLines(nil)
	if len(lines) != 2 || !strings.Contains(lines[1], "└─ -") {
		t.Errorf("expected empty leaderboard format, got: %v", lines)
	}
}
