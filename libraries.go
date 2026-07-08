package main

import (
	"log/slog"
	"time"
)

func runLibrariesDaemon(cfg Config, loc *time.Location) {
	slog.Info("Starting Libraries Promo checker daemon mode...")

	for {
		now := time.Now().In(loc)
		target := calculateNextDailyRunTime(now, loc)

		slog.Info("Libraries Daemon sleeping until next run...", "target", target.Format(time.RFC3339))
		time.Sleep(target.Sub(now))

		slog.Info("Libraries Daemon wake up! Checking promotions for all configured libraries.")
		checkLibrariesPromo(cfg)

		time.Sleep(1 * time.Minute) // Prevent double-triggering in the same minute
	}
}

func calculateNextDailyRunTime(now time.Time, loc *time.Location) time.Time {
	// We want every day at 09:00:00 AM
	target := time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, loc)

	if now.After(target) || now.Equal(target) {
		target = target.AddDate(0, 0, 1)
	}
	return target
}

func checkLibrariesPromo(cfg Config) {
	slog.Info("Checking Wook promos...")
	checkWookPromo(cfg)

	slog.Info("Checking Fnac promos...")
	checkFnacPromo(cfg)
}
