package strava

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const (
	authURL   = "https://www.strava.com/oauth/token"
	apiBase   = "https://www.strava.com/api/v3"
	perPage   = 200
	pageDelay = 500 * time.Millisecond
)

type Client struct {
	clientID     string
	clientSecret string
	refreshToken string
	clubID       string
	httpClient   *http.Client
}

type TokenRefreshResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at"`
}

type Activity struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Distance  float64 `json:"distance"`
	SportType string  `json:"sport_type"`
	Type      string  `json:"type"`
	StartDate string  `json:"start_date"`
}

func (a Activity) EffectiveSportType() string {
	if a.SportType != "" {
		return a.SportType
	}
	return a.Type
}

func NewClient(clientID, clientSecret, refreshToken, clubID string, timeoutSeconds int) *Client {
	return &Client{
		clientID:     clientID,
		clientSecret: clientSecret,
		refreshToken: refreshToken,
		clubID:       clubID,
		httpClient:   &http.Client{Timeout: time.Duration(timeoutSeconds) * time.Second},
	}
}

func (c *Client) RefreshToken() (string, error) {
	params := url.Values{
		"client_id":     {c.clientID},
		"client_secret": {c.clientSecret},
		"grant_type":    {"refresh_token"},
		"refresh_token": {c.refreshToken},
	}

	resp, err := c.httpClient.Post(authURL, "application/x-www-form-urlencoded", strings.NewReader(params.Encode()))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return "", fmt.Errorf("strava token refresh: %w", err)
	}

	var result TokenRefreshResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	slog.Info("Strava token refreshed", "expires_at", result.ExpiresAt)
	return result.AccessToken, nil
}

// WeekStartEpoch returns the Unix timestamp of Monday 00:00:00 UTC for the week of t.
func WeekStartEpoch(t time.Time) int64 {
	weekday := int(t.Weekday())
	daysToMonday := (weekday + 6) % 7
	monday := t.AddDate(0, 0, -daysToMonday)
	monday = time.Date(monday.Year(), monday.Month(), monday.Day(), 0, 0, 0, 0, time.UTC)
	return monday.Unix()
}

func (c *Client) FetchClubActivities(accessToken string, forDate time.Time) ([]Activity, error) {
	after := WeekStartEpoch(forDate)

	slog.Info("Fetching club activities",
		"club_id", c.clubID,
		"after_epoch", after,
		"date", time.Unix(after, 0).UTC().Format("2006-01-02"),
	)

	var all []Activity
	for page := 1; ; page++ {
		reqURL := fmt.Sprintf("%s/clubs/%s/activities?after=%d&page=%d&per_page=%d",
			apiBase, c.clubID, after, page, perPage)

		req, err := http.NewRequest(http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+accessToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, err
		}

		if err := checkStatus(resp); err != nil {
			resp.Body.Close()
			return nil, fmt.Errorf("strava activities page %d: %w", page, err)
		}

		var pageData []Activity
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

		if len(pageData) < perPage {
			break
		}

		time.Sleep(pageDelay)
	}

	slog.Info("Total activities this week", "count", len(all))
	logWeeklyDistanceBySportType(all)
	return all, nil
}

func logWeeklyDistanceBySportType(activities []Activity) {
	if len(activities) == 0 {
		return
	}

	metersByType := make(map[string]float64)
	for _, activity := range activities {
		sportType := activity.EffectiveSportType()
		if sportType == "" {
			sportType = "Unknown"
		}
		metersByType[sportType] += activity.Distance
	}

	sportTypes := make([]string, 0, len(metersByType))
	for sportType := range metersByType {
		sportTypes = append(sportTypes, sportType)
	}
	sort.Strings(sportTypes)

	for _, sportType := range sportTypes {
		km := math.Round((metersByType[sportType]/1000)*100) / 100
		slog.Info("Weekly distance by sport type", "sport_type", sportType, "km", km)
	}
}

func SumWeeklyDistanceKM(activities []Activity, sportTypes []string) float64 {
	var totalMeters float64
	allowedSports := make(map[string]struct{}, len(sportTypes))
	for _, sport := range sportTypes {
		allowedSports[sport] = struct{}{}
	}

	for _, activity := range activities {
		sportType := activity.EffectiveSportType()
		if len(allowedSports) > 0 {
			if _, ok := allowedSports[sportType]; !ok {
				slog.Debug("Skipping activity type (filtered)", "type", sportType)
				continue
			}
		}
		totalMeters += activity.Distance
	}

	km := math.Round(totalMeters/1000*100) / 100
	filter := "all types"
	if len(sportTypes) > 0 {
		filter = fmt.Sprintf("filtered to %v", sportTypes)
	}
	slog.Info("Summed distance", "km", km, "filter", filter)
	return km
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("HTTP %s: %s", resp.Status, body)
}
