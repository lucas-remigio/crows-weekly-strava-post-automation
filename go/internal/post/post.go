package post

import (
	"fmt"
	"strings"
	"time"
)

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

func BuildPostText(weekNumber, totalWeeks int, weeklyKM, annualKM float64, goalKM int) string {
	annualPct := annualKM / float64(goalKM) * 100
	weekPct := float64(weekNumber) / float64(totalWeeks) * 100
	onPaceKM := (float64(goalKM) / float64(totalWeeks)) * float64(weekNumber)

	lines := []string{
		fmt.Sprintf("Semana %d/%d (%.1f%%)", weekNumber, totalWeeks, weekPct),
		"",
		fmt.Sprintf("Total semanal: %.1f km", weeklyKM),
		fmt.Sprintf("Total anual: %.1f / %d km (%.1f%%)", annualKM, goalKM, annualPct),
		"",
		fmt.Sprintf("Por esta altura devíamos ter feito %.0f km", onPaceKM),
	}

	diff := annualKM - onPaceKM
	if diff >= 0 {
		lines = append(lines, fmt.Sprintf("Estamos +%.1f km acima do ritmo. Muito bom!", diff))
	} else {
		lines = append(lines, fmt.Sprintf("Estamos -%.1f km abaixo do ritmo. Vamos lá!", -diff))
	}

	return strings.Join(lines, "\n")
}
