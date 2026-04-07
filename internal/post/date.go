package post

import (
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

// MaxISOWeek returns the total number of ISO weeks in a given year.
// According to the ISO 8601 standard, December 28th always falls in the last week of its year.
func MaxISOWeek(year int) int {
	_, maxWeek := time.Date(year, 12, 28, 0, 0, 0, 0, time.UTC).ISOWeek()
	return maxWeek
}
