package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	stravaAuthURL = "https://www.strava.com/oauth/token"
	stravaAPIBase = "https://www.strava.com/api/v3"
)

func refreshStravaToken(cfg Config) (string, error) {
	params := url.Values{
		"client_id":     {cfg.StravaClientID},
		"client_secret": {cfg.StravaClientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {cfg.StravaRefreshToken},
	}

	client := httpClient(cfg.HTTPTimeoutSeconds)
	resp, err := client.Post(stravaAuthURL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return "", fmt.Errorf("strava token refresh: %w", err)
	}

	var result struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	slog.Info("Strava token refreshed", "expires_at", result.ExpiresAt)
	return result.AccessToken, nil
}

// weekStartEpoch returns the Unix timestamp of Monday 00:00:00 UTC for the week of t.
func weekStartEpoch(t time.Time) int64 {
	weekday := int(t.Weekday()) // 0=Sun, 1=Mon, ..., 6=Sat
	daysToMonday := (weekday + 6) % 7
	monday := t.AddDate(0, 0, -daysToMonday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	return monday.Unix()
}

func fetchClubActivities(cfg Config, accessToken string, forDate time.Time) ([]map[string]any, error) {
	after := weekStartEpoch(forDate)
	client := httpClient(cfg.HTTPTimeoutSeconds)

	slog.Info("Fetching club activities",
		"club_id", cfg.StravaClubID,
		"after_epoch", after,
		"date", time.Unix(after, 0).UTC().Format("2006-01-02"),
	)

	var all []map[string]any
	for page := 1; ; page++ {
		reqURL := fmt.Sprintf("%s/clubs/%s/activities?after=%d&page=%d&per_page=200",
			stravaAPIBase, cfg.StravaClubID, after, page)

		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}

		if err := checkStatus(resp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("strava activities page %d: %w", page, err)
		}

		var pageData []map[string]any
		err = json.NewDecoder(resp.Body).Decode(&pageData)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}

		if len(pageData) == 0 {
			break
		}

		all = append(all, pageData...)
		slog.Info("Activities fetched", "page", page, "total", len(all))

		if len(pageData) < 200 {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}

	slog.Info("Total activities this week", "count", len(all))
	return all, nil
}

func sumWeeklyDistanceKM(cfg Config, activities []map[string]any) float64 {
	var totalMeters float64

	for _, act := range activities {
		sportType, _ := act["sport_type"].(string)
		if sportType == "" {
			sportType, _ = act["type"].(string)
		}

		if len(cfg.SportTypes) > 0 && !sliceContains(cfg.SportTypes, sportType) {
			slog.Debug("Skipping activity type (filtered)", "type", sportType)
			continue
		}

		if d, ok := act["distance"].(float64); ok {
			totalMeters += d
		}
	}

	km := math.Round(totalMeters/1000*100) / 100
	filter := "all types"
	if len(cfg.SportTypes) > 0 {
		filter = fmt.Sprintf("filtered to %v", cfg.SportTypes)
	}
	slog.Info("Summed distance", "km", km, "filter", filter)
	return km
}

func sliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
