package main

import (
	"errors"
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

var errDuplicateWeek = errors.New("week already exists")

func run(dryRun bool, week int, now time.Time) error {
	cfg := loadConfig()

	if !dryRun {
		if err := cfg.validate(); err != nil {
			return fmt.Errorf("configuration error: %w", err)
		}
	}

	forDate, err := resolveRunDate(week, now)
	if err != nil {
		return err
	}

	bounds := getWeekBounds(forDate)
	slog.Info("Running for week",
		"week", bounds.WeekNumber,
		"from", bounds.Monday.Format("2006-01-02"),
		"to", bounds.Sunday.Format("2006-01-02"),
	)

	sc, err := newSheetsClient(cfg)
	if err != nil {
		return fmt.Errorf("create Sheets client: %w", err)
	}

	if !dryRun {
		exists, err := sc.hasEntryForWeek(bounds.WeekNumber)
		if err != nil {
			return fmt.Errorf("check duplicate week: %w", err)
		}
		if exists {
			return errDuplicateWeek
		}
	}

	accessToken, err := refreshStravaToken(cfg)
	if err != nil {
		return fmt.Errorf("refresh Strava token: %w", err)
	}

	activities, err := fetchClubActivities(cfg, accessToken, forDate)
	if err != nil {
		return fmt.Errorf("fetch Strava activities: %w", err)
	}
	weeklyKM := sumWeeklyDistanceKM(cfg, activities)

	lastAnnualKM, err := sc.getLastAnnualTotal()
	if err != nil {
		return fmt.Errorf("read annual total from sheet: %w", err)
	}

	newAnnualKM := math.Round((lastAnnualKM+weeklyKM)*100) / 100
	slog.Info("Distance summary",
		"weekly_km", weeklyKM,
		"previous_annual_km", lastAnnualKM,
		"new_annual_km", newAnnualKM,
	)

	onPaceKM := (float64(cfg.AnnualGoalKM) / float64(cfg.TotalWeeks)) * float64(bounds.WeekNumber)
	athletes := getAthletes(sc)
	roast := generateWeeklyRoast(cfg, athletes, newAnnualKM >= onPaceKM, math.Abs(newAnnualKM-onPaceKM))

	postText := buildPostText(bounds.WeekNumber, cfg.TotalWeeks, weeklyKM, newAnnualKM, cfg.AnnualGoalKM)
	if roast != "" {
		postText = roast + "\n\n" + postText
	}

	printPostText(postText)

	if dryRun {
		slog.Info("DRY RUN — skipping Sheets write and Telegram send.")
		return nil
	}

	if err := sc.ensureHeaderExists(); err != nil {
		return fmt.Errorf("ensure header exists: %w", err)
	}

	if err := sc.appendWeeklyEntry(
		bounds.WeekNumber,
		bounds.Monday, bounds.Sunday,
		weeklyKM, newAnnualKM,
		cfg.AnnualGoalKM,
		postText,
	); err != nil {
		return fmt.Errorf("append to sheet: %w", err)
	}

	sendTelegramMessage(cfg, postText)
	return nil
}

func resolveRunDate(week int, now time.Time) (time.Time, error) {
	if week == 0 {
		return now, nil
	}
	if week < 1 || week > 53 {
		return time.Time{}, fmt.Errorf("invalid ISO week: %d", week)
	}
	forDate := mondayOfISOWeek(now.Year(), week)
	slog.Info("Targeting past week", "week", week, "derived_date", forDate.Format("2006-01-02"))
	return forDate, nil
}

func printPostText(postText string) {
	fmt.Println()
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println("WEEKLY POST TEXT")
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println(postText)
	fmt.Println(strings.Repeat("=", 50))
	fmt.Println()
}
