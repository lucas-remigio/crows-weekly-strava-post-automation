package post

import (
	"fmt"
	"sort"
	"strings"
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
	var lines []string

	lines = append(lines, buildHeader(weekNumber, totalWeeks, stats.TotalDistanceKM)...)
	lines = append(lines, formatWeeklyBySportLines(stats.DistanceBySport)...)
	lines = append(lines, "")
	lines = append(lines, buildMainStats(stats)...)
	lines = append(lines, formatWeeklyByAthleteLines(stats)...)
	lines = append(lines, buildAveragesRow(stats)...)
	lines = append(lines, buildFooter(annualKM, float64(goalKM), weekNumber, totalWeeks)...)

	return strings.Join(lines, "\n")
}

func buildHeader(weekNum, totalWeeks int, weeklyKM float64) []string {
	weekPct := float64(weekNum) / float64(totalWeeks) * 100
	return []string{
		fmt.Sprintf("Semana %d/%d (%.1f%%)", weekNum, totalWeeks, weekPct),
		"",
		fmt.Sprintf("Total semanal: %.1f km", weeklyKM),
	}
}

func buildMainStats(stats WeeklyStats) []string {
	var lines []string
	if stats.TotalMovingTime > 0 {
		lines = append(lines, fmt.Sprintf("⏱️ Tempo em movimento: %s", formatDuration(stats.TotalMovingTime)))
	}
	if stats.TotalElevation > 0 {
		lines = append(lines, fmt.Sprintf("⛰️ Subimos juntos: %.0f metros", stats.TotalElevation))
	}
	if len(lines) > 0 {
		lines = append(lines, "")
	}
	return lines
}

func buildAveragesRow(stats WeeklyStats) []string {
	numAthletes := len(stats.DistanceByAthlete)
	if numAthletes == 0 {
		return nil
	}

	avgKM := stats.TotalDistanceKM / float64(numAthletes)
	avgTime := stats.TotalMovingTime / numAthletes
	avgElev := stats.TotalElevation / float64(numAthletes)

	return []string{
		fmt.Sprintf("Média por atleta: %.1f km | %s | +%.0fm alt.", avgKM, formatDuration(avgTime), avgElev),
		fmt.Sprintf("\n🔥 Destaque: A volta épica do(a) %s", stats.EpicAthlete),
		fmt.Sprintf(" '%s' com uns incríveis %.1f km!", stats.EpicActivityName, stats.EpicActivityKM),
	}
}

func buildFooter(annualKM, goalKM float64, weekNum, totalWeeks int) []string {
	annualPct := (annualKM / goalKM) * 100
	onPaceKM := (goalKM / float64(totalWeeks)) * float64(weekNum)
	bar := renderProgressBar(annualPct, 10)

	lines := []string{
		"",
		fmt.Sprintf("Objetivo Anual: [%s] %.1f%% (%.1f / %.0f km)", bar, annualPct, annualKM, goalKM),
		"",
		fmt.Sprintf("Por esta altura devíamos ter feito %.0f km", onPaceKM),
	}

	diff := annualKM - onPaceKM
	if diff >= 0 {
		lines = append(lines, fmt.Sprintf("🚀 Estamos +%.1f km acima do ritmo! Máquinas!", diff))
	} else {
		lines = append(lines, fmt.Sprintf("🚨 Estamos -%.1f km abaixo do ritmo! Bora mexer essas pernas!", -diff))
	}

	return lines
}

type athleteScore struct {
	Name string
	KM   float64
}

func formatWeeklyByAthleteLines(stats WeeklyStats) []string {
	if len(stats.DistanceByAthlete) == 0 {
		return []string{"Leaderboard:", "└─ -"}
	}

	items := getSortedAthleteScores(stats.DistanceByAthlete)
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, "Leaderboard da Semana:")

	for i, item := range items {
		lines = append(lines, formatAthleteLine(i, len(items), item, stats))
	}

	return lines
}

func getSortedAthleteScores(distance map[string]float64) []athleteScore {
	var items []athleteScore
	for name, km := range distance {
		items = append(items, athleteScore{Name: name, KM: km})
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].KM > items[j].KM
	})
	return items
}

func formatAthleteLine(i, totalItems int, item athleteScore, stats WeeklyStats) string {
	prefix := "├─"
	if i == totalItems-1 {
		prefix = "└─"
	}

	rank := "🐢"
	if medals := []string{"🥇", "🥈", "🥉"}; i < len(medals) {
		rank = medals[i]
	}

	badge := getAthleteBadges(item.Name, item.KM, stats)
	extras := getAthleteExtras(item.Name, stats)

	if extras != "" {
		extras = fmt.Sprintf(" (%s)", extras)
	}
	if badge != "" {
		badge = fmt.Sprintf(" (%s)", strings.TrimSpace(badge))
	}

	return fmt.Sprintf("%s %s %s: %.1f km%s%s", prefix, rank, item.Name, item.KM, extras, badge)
}

func getAthleteBadges(name string, km float64, stats WeeklyStats) string {
	var badge string
	if km > 50 {
		badge += " 🚀 Lenda"
	}
	if name == stats.MountainGoat && stats.MountainGoat != "" {
		badge += " 🐐 Cabra da Montanha"
	}
	if name == stats.MachineAthlete && stats.MachineAthlete != "" {
		badge += " 🤖 Papa-Treinos"
	}
	return badge
}

func getAthleteExtras(name string, stats WeeklyStats) string {
	var extras string
	if timeVal, ok := stats.TimeByAthlete[name]; ok && timeVal > 0 {
		extras += formatDuration(timeVal)
	}
	if elev, ok := stats.ElevationByAthlete[name]; ok && elev > 0 {
		if extras != "" {
			extras += " | "
		}
		extras += fmt.Sprintf("+%.0fm alt.", elev)
	}
	return extras
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
