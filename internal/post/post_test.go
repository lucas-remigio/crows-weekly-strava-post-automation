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
}

func TestMondayOfISOWeek(t *testing.T) {
	monday := MondayOfISOWeek(2026, 1)
	if got := monday.Format("2006-01-02"); got != "2025-12-29" {
		t.Fatalf("expected 2025-12-29, got %s", got)
	}
}

func TestBuildPostTextBelowPace(t *testing.T) {
	text := BuildPostText(12, 53, 1248.6, 12000, WeeklyStats{
		TotalDistanceKM: 13.2,
		TotalMovingTime: 3660, // 1h1min
		TotalElevation:  105,
		DistanceBySport: map[string]float64{
			"Run":  10.2,
			"Walk": 3.0,
		},
		DistanceByAthlete: map[string]float64{
			"Alice": 10.2,
			"Bob":   3.0,
		},
		ElevationByAthlete: map[string]float64{
			"Alice": 105,
		},
		TimeByAthlete: map[string]int{
			"Alice": 3600,
			"Bob":   60,
		},
		EpicActivityName: "Big Loop",
		EpicAthlete:      "Alice",
		EpicActivityKM:   10.2,
	})

	if !strings.Contains(text, "Semana 12/53") {
		t.Fatalf("expected week line in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "⏱️ Tempo em movimento: 1h 01min") {
		t.Fatalf("expected moving time in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "⛰️ Subimos juntos: 105 metros") {
		t.Fatalf("expected elevation in post text, got:\n%s", text)
	}
	if !strings.Contains(text, "├─ 🏃 Corrida: 10.2 km") {
		t.Fatalf("expected tree line for run, got:\n%s", text)
	}
	if !strings.Contains(text, "Média por atleta: 6.6 km | 30min | +52m alt.") {
		t.Fatalf("expected valid average line, got:\n%s", text)
	}
	if !strings.Contains(text, "Destaque: A volta épica do(a) Alice") {
		t.Fatalf("expected epic run feature, got:\n%s", text)
	}
}

func TestFormatWeeklyByAthleteLines(t *testing.T) {
	stats := WeeklyStats{
		DistanceByAthlete: map[string]float64{
			"Bob":     10.5, // 3rd
			"Charlie": 5.2,  // 4th
			"Alice":   52.0, // 1st
			"Eve":     1.0,  // 5th
			"Dave":    21.0, // 2nd
		},
		ElevationByAthlete: map[string]float64{
			"Alice":   200,
			"Dave":    100,
			"Charlie": 50,
			"Eve":     3000, // Goated!
		},
		TimeByAthlete: map[string]int{
			"Alice": 3660, // 1h 01m
			"Dave":  3600, // 1h 00m
			"Bob":   1800, // 30m
		},
		MountainGoat:   "Eve",
		MachineAthlete: "Dave",
	}

	lines := formatWeeklyByAthleteLines(stats)

	if !strings.Contains(lines[1], "🥇 Alice: 52.0 km (1h 01min | +200m alt.) (🚀 Lenda)") {
		t.Errorf("expected Gold and details for Alice, got: %s", lines[1])
	}
	if !strings.Contains(lines[2], "🥈 Dave: 21.0 km (1h 00min | +100m alt.) (🤖 Papa-Treinos)") {
		t.Errorf("expected Silver and Machine for Dave, got: %s", lines[2])
	}
	if !strings.Contains(lines[3], "🥉 Bob: 10.5 km (30min)") {
		t.Errorf("expected Bronze for Bob, got: %s", lines[3])
	}
	if !strings.Contains(lines[4], "🐢 Charlie: 5.2 km (+50m alt.)") {
		t.Errorf("expected Turtle for Charlie, got: %s", lines[4])
	}
	if !strings.Contains(lines[5], "└─ 🐢 Eve: 1.0 km (+3000m alt.) (🐐 Cabra da Montanha)") {
		t.Errorf("expected Turtle and Mountain Goat for Eve, got: %s", lines[5])
	}
}

func TestFormatWeeklyByAthleteLinesEmpty(t *testing.T) {
	lines := formatWeeklyByAthleteLines(WeeklyStats{})
	if len(lines) != 2 || !strings.Contains(lines[1], "└─ -") {
		t.Errorf("expected empty leaderboard format, got: %v", lines)
	}
}
