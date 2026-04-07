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

	postpkg "strava-weekly-post/internal/post"
	"strava-weekly-post/internal/sheets"
	"strava-weekly-post/internal/strava"
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

func runDaemon() {
	slog.Info("Starting Strava Crows Weekly Post daemon mode...")
	loc, err := time.LoadLocation("Europe/Lisbon")
	if err != nil {
		slog.Warn("Failed to load Lisbon timezone. Falling back to UTC.", "error", err)
		loc = time.UTC
	}

	for {
		now := time.Now().In(loc)
		// We want Sunday at 22:00:00
		daysUntilSunday := int(time.Sunday - now.Weekday())
		if daysUntilSunday < 0 {
			daysUntilSunday += 7 // next week
		}

		target := time.Date(now.Year(), now.Month(), now.Day()+daysUntilSunday, 22, 0, 0, 0, loc)

		// If today is Sunday but it's already past 22:00, schedule for *next* Sunday
		if now.After(target) {
			target = target.AddDate(0, 0, 7)
		}

		sleepDuration := target.Sub(now)
		slog.Info("Sleeping until next run...", "target", target.Format(time.RFC3339))
		time.Sleep(sleepDuration)

		slog.Info("Wake up! Triggering weekly post.")
		if err := run(false, 0, time.Now()); err != nil {
			if errors.Is(err, errDuplicateWeek) {
				slog.Warn("Week already exists in the sheet — skipping duplicate alert.")
			} else {
				slog.Error("Scheduled run failed", "error", err)
			}
		}

		// Sleep slightly to avoid double firing in the same minute
		time.Sleep(1 * time.Minute)
	}
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

	bounds := postpkg.GetWeekBounds(forDate)
	slog.Info("Running for week",
		"week", bounds.WeekNumber,
		"from", bounds.Monday.Format("2006-01-02"),
		"to", bounds.Sunday.Format("2006-01-02"),
	)

	sc, err := sheets.NewClient(cfg.GoogleServiceAccountJSON, cfg.GoogleSheetID, cfg.HTTPTimeoutSeconds)
	if err != nil {
		return fmt.Errorf("create Sheets client: %w", err)
	}

	if !dryRun {
		exists, err := sc.HasEntryForWeek(bounds.WeekNumber)
		if err != nil {
			return fmt.Errorf("check duplicate week: %w", err)
		}
		if exists {
			return errDuplicateWeek
		}
	}

	stravaClient := strava.NewClient(
		cfg.StravaClientID,
		cfg.StravaClientSecret,
		cfg.StravaRefreshToken,
		cfg.StravaClubID,
		cfg.HTTPTimeoutSeconds,
	)

	accessToken, err := stravaClient.RefreshToken()
	if err != nil {
		return fmt.Errorf("refresh Strava token: %w", err)
	}

	activities, err := stravaClient.FetchClubActivities(accessToken, forDate)
	if err != nil {
		return fmt.Errorf("fetch Strava activities: %w", err)
	}
	weeklyKM := strava.SumWeeklyDistanceKM(activities, cfg.SportTypes)
	weeklyKMBySport := strava.SumWeeklyDistanceBySportKM(activities, cfg.SportTypes)

	lastAnnualKM, err := sc.GetLastAnnualTotal()
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

	postText := postpkg.BuildPostText(bounds.WeekNumber, cfg.TotalWeeks, weeklyKM, newAnnualKM, cfg.AnnualGoalKM, weeklyKMBySport)
	if roast != "" {
		postText = roast + "\n\n" + postText
	}

	printPostText(postText)

	if dryRun {
		slog.Info("DRY RUN — skipping Sheets write and Telegram send.")
		return nil
	}

	if err := sc.EnsureHeaderExists(); err != nil {
		return fmt.Errorf("ensure header exists: %w", err)
	}

	if err := sc.AppendWeeklyEntry(
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
	forDate := postpkg.MondayOfISOWeek(now.Year(), week)
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
