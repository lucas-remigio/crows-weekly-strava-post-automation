package main

import (
	"flag"
	"fmt"
	"log/slog"
	"math"
	"os"
	"strings"
	"time"
)

func main() {
	dryRun := flag.Bool("dry-run", false, "Fetch and print the post, but skip Sheets write and Telegram send.")
	week := flag.Int("week", 0, "ISO week number to process (0 = current week). Use to recover a missed run.")
	flag.Parse()

	cfg := loadConfig()

	if !*dryRun {
		if err := cfg.validate(); err != nil {
			slog.Error("Configuration error", "error", err)
			os.Exit(1)
		}
	}

	var forDate time.Time
	if *week > 0 {
		forDate = mondayOfISOWeek(time.Now().Year(), *week)
		slog.Info("Targeting past week", "week", *week, "derived_date", forDate.Format("2006-01-02"))
	} else {
		forDate = time.Now()
	}

	bounds := getWeekBounds(forDate)
	slog.Info("Running for week",
		"week", bounds.WeekNumber,
		"from", bounds.Monday.Format("2006-01-02"),
		"to", bounds.Sunday.Format("2006-01-02"),
	)

	sc, err := newSheetsClient(cfg)
	if err != nil {
		slog.Error("Failed to create Sheets client", "error", err)
		os.Exit(1)
	}

	if !*dryRun {
		exists, err := sc.hasEntryForWeek(bounds.WeekNumber)
		if err != nil {
			slog.Error("Failed to check for duplicate week", "error", err)
			os.Exit(1)
		}
		if exists {
			slog.Warn("Week already exists in the sheet — exiting to avoid duplicate.", "week", bounds.WeekNumber)
			os.Exit(0)
		}
	}

	// Step 1: Fetch Strava activities
	accessToken, err := refreshStravaToken(cfg)
	if err != nil {
		slog.Error("Failed to refresh Strava token", "error", err)
		os.Exit(1)
	}

	activities, err := fetchClubActivities(cfg, accessToken, forDate)
	if err != nil {
		slog.Error("Failed to fetch Strava activities", "error", err)
		os.Exit(1)
	}

	weeklyKM := sumWeeklyDistanceKM(cfg, activities)

	// Step 2: Read annual total from Sheets
	lastAnnualKM, err := sc.getLastAnnualTotal()
	if err != nil {
		slog.Error("Failed to read annual total from sheet", "error", err)
		os.Exit(1)
	}

	newAnnualKM := math.Round((lastAnnualKM+weeklyKM)*100) / 100
	slog.Info("Distance summary",
		"weekly_km", weeklyKM,
		"previous_annual_km", lastAnnualKM,
		"new_annual_km", newAnnualKM,
	)

	// Step 3: Build post text
	onPaceKM := (float64(cfg.AnnualGoalKM) / float64(cfg.TotalWeeks)) * float64(bounds.WeekNumber)
	athletes := getAthletes(sc)
	roast := generateWeeklyRoast(cfg, athletes, newAnnualKM >= onPaceKM, math.Abs(newAnnualKM-onPaceKM))

	postText := buildPostText(bounds.WeekNumber, cfg.TotalWeeks, weeklyKM, newAnnualKM, cfg.AnnualGoalKM)
	if roast != "" {
		postText = roast + "\n\n" + postText
	}

	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("WEEKLY POST TEXT")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(postText)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()

	if *dryRun {
		slog.Info("DRY RUN — skipping Sheets write and Telegram send.")
		return
	}

	// Step 4: Write to Google Sheets
	if err := sc.ensureHeaderExists(); err != nil {
		slog.Error("Failed to ensure header exists", "error", err)
		os.Exit(1)
	}

	if err := sc.appendWeeklyEntry(
		bounds.WeekNumber,
		bounds.Monday, bounds.Sunday,
		weeklyKM, newAnnualKM,
		cfg.AnnualGoalKM,
		postText,
	); err != nil {
		slog.Error("Failed to append to sheet", "error", err)
		os.Exit(1)
	}

	// Step 5: Send Telegram
	sendTelegramMessage(cfg, postText)

	slog.Info("Done.")
}
