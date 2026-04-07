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

func BuildPostText(weekNumber, totalWeeks int, weeklyKM, annualKM float64, goalKM int, weeklyKMBySport, weeklyKMByAthlete map[string]float64) string {
	annualPct := annualKM / float64(goalKM) * 100
	weekPct := float64(weekNumber) / float64(totalWeeks) * 100
	onPaceKM := (float64(goalKM) / float64(totalWeeks)) * float64(weekNumber)

	lines := []string{
		fmt.Sprintf("Semana %d/%d (%.1f%%)", weekNumber, totalWeeks, weekPct),
		"",
		fmt.Sprintf("Total semanal: %.1f km", weeklyKM),
	}

	lines = append(lines, formatWeeklyBySportLines(weeklyKMBySport)...)
	lines = append(lines, "")
	lines = append(lines, formatWeeklyByAthleteLines(weeklyKMByAthlete)...)
	lines = append(lines,
		"",
		fmt.Sprintf("Total anual: %.1f / %d km (%.1f%%)", annualKM, goalKM, annualPct),
		"",
		fmt.Sprintf("Por esta altura devíamos ter feito %.0f km", onPaceKM),
	)

	diff := annualKM - onPaceKM
	if diff >= 0 {
		lines = append(lines, fmt.Sprintf("Estamos +%.1f km acima do ritmo. Muito bom!", diff))
	} else {
		lines = append(lines, fmt.Sprintf("Estamos -%.1f km abaixo do ritmo. Vamos lá!", -diff))
	}

	return strings.Join(lines, "\n")
}

func formatWeeklyByAthleteLines(weeklyKMByAthlete map[string]float64) []string {
	if len(weeklyKMByAthlete) == 0 {
		return []string{"Leaderboard:", "└─ -"}
	}

	type athleteScore struct {
		Name string
		KM   float64
	}

	var items []athleteScore
	for name, km := range weeklyKMByAthlete {
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

		rank := "💩"
		if i < len(medals) {
			rank = medals[i]
		}

		lines = append(lines, fmt.Sprintf("%s %s %s: %.1f km", prefix, rank, item.Name, item.KM))
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
