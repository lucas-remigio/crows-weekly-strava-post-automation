package main

import (
	"errors"
	"flag"
	"log/slog"
	"os"
	"time"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Fetch and print the post, but skip Sheets write and Telegram send.")
	week := flag.Int("week", 0, "ISO week number to process (0 = current week). Use to recover a missed run.")
	daemon := flag.Bool("daemon", true, "Run continuously as a daemon and trigger automatically every Sunday at 22:00")
	flag.Parse()

	if *daemon && !*dryRun && *week == 0 {
		runDaemon()
	} else {
		// Manual one-off run
		if err := run(*dryRun, *week, time.Now()); err != nil {
			if errors.Is(err, errDuplicateWeek) {
				slog.Warn("Week already exists in the sheet — exiting to avoid duplicate.")
				os.Exit(0)
			}
			slog.Error("Run failed", "error", err)
			os.Exit(1)
		}
		slog.Info("Done.")
	}
}
