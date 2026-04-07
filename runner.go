package main

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	postpkg "strava-weekly-post/internal/post"
	"strava-weekly-post/internal/sheets"
	"strava-weekly-post/internal/strava"
)

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
	slog.Info("Running for week", "week", bounds.WeekNumber, "from", bounds.Monday.Format("2006-01-02"))

	sc, err := sheets.NewClient(cfg.GoogleServiceAccountJSON, cfg.GoogleSheetID, cfg.HTTPTimeoutSeconds)
	if err != nil {
		return fmt.Errorf("create Sheets client: %w", err)
	}

	if !dryRun {
		if err := checkDuplicateWeek(sc, bounds.WeekNumber); err != nil {
			return err
		}
	}

	weeklyKM, weeklyKMBySport, weeklyKMByAthlete, err := fetchStravaStats(cfg, forDate)
	if err != nil {
		return err
	}

	newAnnualKM, err := calculateNewAnnualTotal(sc, weeklyKM)
	if err != nil {
		return err
	}

	postText := compilePost(cfg, sc, bounds, newAnnualKM, weeklyKM, weeklyKMBySport, weeklyKMByAthlete)
	printPostText(postText)

	if dryRun {
		slog.Info("DRY RUN — skipping Sheets write and Telegram send.")
		return nil
	}

	return publishResults(cfg, sc, bounds, postText, weeklyKM, newAnnualKM)
}

func checkDuplicateWeek(sc *sheets.Client, weekNumber int) error {
	exists, err := sc.HasEntryForWeek(weekNumber)
	if err != nil {
		return fmt.Errorf("check duplicate week: %w", err)
	}
	if exists {
		return errDuplicateWeek
	}
	return nil
}

func fetchStravaStats(cfg Config, forDate time.Time) (float64, map[string]float64, map[string]float64, error) {
	stravaClient := strava.NewClient(
		cfg.StravaClientID, cfg.StravaClientSecret, cfg.StravaRefreshToken,
		cfg.StravaClubID, cfg.HTTPTimeoutSeconds,
	)

	accessToken, err := stravaClient.RefreshToken()
	if err != nil {
		return 0, nil, nil, fmt.Errorf("refresh Strava token: %w", err)
	}

	activities, err := stravaClient.FetchClubActivities(accessToken, forDate)
	if err != nil {
		return 0, nil, nil, fmt.Errorf("fetch Strava activities: %w", err)
	}

	return strava.SumWeeklyDistanceKM(activities, cfg.SportTypes),
		strava.SumWeeklyDistanceBySportKM(activities, cfg.SportTypes),
		strava.SumWeeklyDistanceByAthleteKM(activities, cfg.SportTypes), nil
}

func calculateNewAnnualTotal(sc *sheets.Client, weeklyKM float64) (float64, error) {
	lastAnnualKM, err := sc.GetLastAnnualTotal()
	if err != nil {
		return 0, fmt.Errorf("read annual total from sheet: %w", err)
	}

	newAnnualKM := math.Round((lastAnnualKM+weeklyKM)*100) / 100
	slog.Info("Distance summary", "weekly_km", weeklyKM, "new_annual_km", newAnnualKM)
	return newAnnualKM, nil
}

func compilePost(cfg Config, sc *sheets.Client, bounds postpkg.WeekBounds, newAnnualKM, weeklyKM float64, bySport map[string]float64, byAthlete map[string]float64) string {
	onPaceKM := (float64(cfg.AnnualGoalKM) / float64(cfg.TotalWeeks)) * float64(bounds.WeekNumber)
	athletes := getAthletes(sc)

	roast := generateWeeklyRoast(cfg, athletes, newAnnualKM >= onPaceKM, math.Abs(newAnnualKM-onPaceKM))
	postText := postpkg.BuildPostText(bounds.WeekNumber, cfg.TotalWeeks, weeklyKM, newAnnualKM, cfg.AnnualGoalKM, bySport, byAthlete)

	if roast != "" {
		return roast + "\n\n" + postText
	}
	return postText
}

func publishResults(cfg Config, sc *sheets.Client, bounds postpkg.WeekBounds, postText string, weeklyKM, newAnnualKM float64) error {
	if err := sc.EnsureHeaderExists(); err != nil {
		return fmt.Errorf("ensure header exists: %w", err)
	}

	if err := sc.AppendWeeklyEntry(
		bounds.WeekNumber, bounds.Monday, bounds.Sunday,
		weeklyKM, newAnnualKM, cfg.AnnualGoalKM, postText,
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
