package main

import (
	"errors"
	"log/slog"
	"time"
)

func runDaemon() {
	slog.Info("Starting Strava Crows Weekly Post daemon mode...")
	loc := getLisbonTimezone()

	for {
		now := time.Now().In(loc)
		target := calculateNextRunTime(now, loc)

		slog.Info("Sleeping until next run...", "target", target.Format(time.RFC3339))
		time.Sleep(target.Sub(now))

		slog.Info("Wake up! Triggering weekly post.")
		executeScheduledRun()

		time.Sleep(1 * time.Minute) // Prevent double-triggering in the same minute
	}
}

func getLisbonTimezone() *time.Location {
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		slog.Warn("Failed to load Lisbon timezone. Falling back to UTC.", "error", err)
		return time.UTC
	}
	return loc
}

func calculateNextRunTime(now time.Time, loc *time.Location) time.Time {
	// We want Monday at 06:00:00 AM (perfect morning time to catch all late Sunday runs)
	target := time.Date(now.Year(), now.Month(), now.Day(), 6, 0, 0, 0, loc)

	// Shift forward until we hit a future Monday
	for target.Weekday() != time.Monday || now.After(target) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

func executeScheduledRun() {
	// Trick the run() function into evaluating the week that *just* ended
	// by passing yesterday's date (Sunday), avoiding the brand new empty Monday week.
	evalDate := time.Now().Add(-24 * time.Hour)

	if err := run(false, 0, evalDate); err != nil {
		if errors.Is(err, errDuplicateWeek) {
			slog.Warn("Week already exists in the sheet — skipping duplicate alert.")
		} else {
			slog.Error("Scheduled run failed", "error", err)
		}
	}
}
