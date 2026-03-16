package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

// Config holds all runtime configuration loaded from environment variables.
type Config struct {
	StravaClientID           string
	StravaClientSecret       string
	StravaRefreshToken       string
	StravaClubID             string
	GoogleServiceAccountJSON string
	GoogleSheetID            string
	TelegramBotToken         string
	TelegramChatIDs          []string
	OpenAIAPIKey             string
	AnnualGoalKM             int
	TotalWeeks               int
	HTTPTimeoutSeconds       int
	SportTypes               []string
}

var requiredEnvKeys = []string{
	"STRAVA_CLIENT_ID",
	"STRAVA_CLIENT_SECRET",
	"STRAVA_REFRESH_TOKEN",
	"STRAVA_CLUB_ID",
	"GOOGLE_SERVICE_ACCOUNT_JSON",
	"GOOGLE_SHEET_ID",
	"TELEGRAM_BOT_TOKEN",
	"TELEGRAM_CHAT_IDS",
}

func loadConfig() Config {
	if err := godotenv.Load(); err != nil {
		_ = godotenv.Load("../.env")
	}
	return Config{
		StravaClientID:           os.Getenv("STRAVA_CLIENT_ID"),
		StravaClientSecret:       os.Getenv("STRAVA_CLIENT_SECRET"),
		StravaRefreshToken:       os.Getenv("STRAVA_REFRESH_TOKEN"),
		StravaClubID:             os.Getenv("STRAVA_CLUB_ID"),
		GoogleServiceAccountJSON: os.Getenv("GOOGLE_SERVICE_ACCOUNT_JSON"),
		GoogleSheetID:            os.Getenv("GOOGLE_SHEET_ID"),
		TelegramBotToken:         os.Getenv("TELEGRAM_BOT_TOKEN"),
		TelegramChatIDs:          splitNonEmpty(os.Getenv("TELEGRAM_CHAT_IDS")),
		OpenAIAPIKey:             os.Getenv("OPENAI_API_KEY"),
		AnnualGoalKM:             envInt("ANNUAL_GOAL_KM", 12000),
		TotalWeeks:               envInt("TOTAL_WEEKS", 52),
		HTTPTimeoutSeconds:       envInt("HTTP_TIMEOUT_SECONDS", 15),
		SportTypes:               splitNonEmpty(os.Getenv("SPORT_TYPES")),
	}
}

func (c Config) validate() error {
	var missing []string
	for _, k := range requiredEnvKeys {
		if os.Getenv(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}
	return nil
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func envInt(key string, def int) int {
	v, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return v
}
