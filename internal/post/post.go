package post

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

type sportBreakdown struct {
	Label string
	KM    float64
}

var sportTypePT = map[string]string{
	"run":              "🏃 Corrida",
	"trailrun":         "⛰️ Corrida em Trilho",
	"walk":             "🚶 Caminhada",
	"hike":             "🥾 Caminhada",
	"ride":             "🚴 Ciclismo",
	"virtualride":      "🚴 Ciclismo",
	"ebikeride":        "🚴 Ciclismo",
	"mountainbikeride": "🚵 Ciclismo",
	"gravelride":       "🚴 Ciclismo",
	"cyclocross":       "🚴 Ciclismo",
	"velomobile":       "🚴 Ciclismo",
	"swim":             "🏊 Natação",
	"rowing":           "🚣 Remo",
	"workout":          "💪 Treino",
	"weighttraining":   "🏋️ Musculação",
	"yoga":             "🧘 Yoga",
	"unknown":          "❓ Desconhecido",
	"":                 "❓ Desconhecido",
}

type WeekBounds struct {
	WeekNumber int
	Monday     time.Time
	Sunday     time.Time
}

func GetWeekBounds(forDate time.Time) WeekBounds {
	_, week := forDate.ISOWeek()

	weekday := int(forDate.Weekday())
	daysToMonday := (weekday + 6) % 7

	monday := forDate.AddDate(0, 0, -daysToMonday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, monday.Location())
	sunday := monday.AddDate(0, 0, 6)

	return WeekBounds{WeekNumber: week, Monday: monday, Sunday: sunday}
}

// MondayOfISOWeek returns the Monday of ISO week `week` in the given year.
func MondayOfISOWeek(year, week int) time.Time {
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, time.UTC)
	weekday := int(jan4.Weekday())
	daysToMonday := (weekday + 6) % 7
	week1Monday := jan4.AddDate(0, 0, -daysToMonday)
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

func renderProgressBar(percentage float64, totalBlocks int) string {
	if percentage < 0 {
		percentage = 0
	}
	filled := int((percentage / 100) * float64(totalBlocks))
	if filled > totalBlocks {
		filled = totalBlocks
	}
	empty := totalBlocks - filled
	return strings.Repeat("█", filled) + strings.Repeat("░", empty)
}

type WeeklyStats struct {
	TotalDistanceKM    float64
	DistanceBySport    map[string]float64
	DistanceByAthlete  map[string]float64
	ElevationByAthlete map[string]float64
	TimeByAthlete      map[string]int
	TotalElevation     float64
	TotalMovingTime    int
	MountainGoat       string
	MachineAthlete     string
	EpicActivityName   string
	EpicAthlete        string
	EpicActivityKM     float64
}

func formatDuration(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 {
		return fmt.Sprintf("%dh %02dmin", hours, minutes)
	}
	return fmt.Sprintf("%dmin", minutes)
}

func BuildPostText(weekNumber, totalWeeks int, annualKM float64, goalKM int, stats WeeklyStats) string {
	annualPct := annualKM / float64(goalKM) * 100
	weekPct := float64(weekNumber) / float64(totalWeeks) * 100
	onPaceKM := (float64(goalKM) / float64(totalWeeks)) * float64(weekNumber)

	lines := []string{
		fmt.Sprintf("Semana %d/%d (%.1f%%)", weekNumber, totalWeeks, weekPct),
		"",
		fmt.Sprintf("Total semanal: %.1f km", stats.TotalDistanceKM),
	}
	lines = append(lines, formatWeeklyBySportLines(stats.DistanceBySport)...)
	lines = append(lines, "")

	if stats.TotalMovingTime > 0 {
		lines = append(lines, fmt.Sprintf("⏱️ Tempo em movimento: %s", formatDuration(stats.TotalMovingTime)))
	}

	if stats.TotalElevation > 0 {
		lines = append(lines, fmt.Sprintf("⛰️ Subimos juntos: %.0f metros", stats.TotalElevation))
	}
	lines = append(lines, "")

	lines = append(lines, formatWeeklyByAthleteLines(stats)...)

	if numAthletes := len(stats.DistanceByAthlete); numAthletes > 0 {
		avgKM := stats.TotalDistanceKM / float64(numAthletes)
		avgTime := stats.TotalMovingTime / numAthletes
		avgElev := stats.TotalElevation / float64(numAthletes)

		lines = append(lines, fmt.Sprintf("Média por atleta: %.1f km | %s | +%.0fm alt.", avgKM, formatDuration(avgTime), avgElev))
		lines = append(lines, fmt.Sprintf("\n🔥 Destaque: A volta épica do(a) %s", stats.EpicAthlete))
		lines = append(lines, fmt.Sprintf(" '%s' com uns incríveis %.1f km!", stats.EpicActivityName, stats.EpicActivityKM))
	}

	bar := renderProgressBar(annualPct, 10)
	lines = append(lines,
		"",
		fmt.Sprintf("Objetivo Anual: [%s] %.1f%% (%.1f / %d km)", bar, annualPct, annualKM, goalKM),
		"",
		fmt.Sprintf("Por esta altura devíamos ter feito %.0f km", onPaceKM),
	)

	diff := annualKM - onPaceKM
	if diff >= 0 {
		lines = append(lines, fmt.Sprintf("🚀 Estamos +%.1f km acima do ritmo! Máquinas!", diff))
	} else {
		lines = append(lines, fmt.Sprintf("🚨 Estamos -%.1f km abaixo do ritmo! Bora mexer essas pernas!", -diff))
	}

	return strings.Join(lines, "\n")
}

func formatWeeklyByAthleteLines(stats WeeklyStats) []string {
	if len(stats.DistanceByAthlete) == 0 {
		return []string{"Leaderboard:", "└─ -"}
	}

	type athleteScore struct {
		Name string
		KM   float64
	}

	var items []athleteScore
	for name, km := range stats.DistanceByAthlete {
		items = append(items, athleteScore{Name: name, KM: km})
	}

	// Sort descending by KM
	sort.Slice(items, func(i, j int) bool {
		return items[i].KM > items[j].KM
	})

	medals := []string{"🥇", "🥈", "🥉"}

	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "Leaderboard da Semana:")
	for i, item := range items {
		prefix := "├─"
		if i == len(items)-1 {
			prefix = "└─"
		}

		rank := "🐢"
		if i < len(medals) {
			rank = medals[i]
		}

		badge := ""
		if item.KM > 50 {
			badge += " 🚀 Lenda"
		}
		if item.Name == stats.MountainGoat && stats.MountainGoat != "" {
			badge += " 🐐 Cabra da Montanha"
		}
		if item.Name == stats.MachineAthlete && stats.MachineAthlete != "" {
			badge += " 🤖 Papa-Treinos"
		}

		extras := ""
		if time, ok := stats.TimeByAthlete[item.Name]; ok && time > 0 {
			extras += formatDuration(time)
		}
		if elev, ok := stats.ElevationByAthlete[item.Name]; ok && elev > 0 {
			if extras != "" {
				extras += " | "
			}
			extras += fmt.Sprintf("+%.0fm alt.", elev)
		}

		if extras != "" {
			extras = fmt.Sprintf(" (%s)", extras)
		}

		if badge != "" {
			badge = fmt.Sprintf(" (%s)", strings.TrimSpace(badge))
		}

		lines = append(lines, fmt.Sprintf("%s %s %s: %.1f km%s%s", prefix, rank, item.Name, item.KM, extras, badge))
	}

	return lines
}

func formatWeeklyBySportLines(weeklyKMBySport map[string]float64) []string {
	if len(weeklyKMBySport) == 0 {
		return []string{"Por modalidade:", "└─ -"}
	}

	items := make([]sportBreakdown, 0, len(weeklyKMBySport))
	for sportType, km := range weeklyKMBySport {
		items = append(items, sportBreakdown{
			Label: translateSportTypeToPT(sportType),
			KM:    km,
		})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Label < items[j].Label
	})

	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "Por modalidade:")
	for i, item := range items {
		prefix := "├─"
		if i == len(items)-1 {
			prefix = "└─"
		}
		lines = append(lines, fmt.Sprintf("%s %s: %.1f km", prefix, item.Label, item.KM))
	}

	return lines
}

func translateSportTypeToPT(sportType string) string {
	key := strings.ToLower(strings.TrimSpace(sportType))
	if translated, ok := sportTypePT[key]; ok {
		return translated
	}
	return sportType
}
