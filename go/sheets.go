package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	sheetsAPIBase = "https://sheets.googleapis.com/v4/spreadsheets"
	sheetsScope   = "https://www.googleapis.com/auth/spreadsheets"
	athletesSheet = "Atletas"
)

var headerRow = []any{
	"Semana",
	"Início da Semana",
	"Fim da Semana",
	"KM Semanal",
	"Total Anual KM",
	"Objetivo Anual KM",
	"Texto do Post",
	"Executado Em",
}

type sheetsClient struct {
	client          *http.Client
	sheetID         string
	firstSheetTitle string
}

func newSheetsClient(cfg Config) (*sheetsClient, error) {
	creds, err := google.CredentialsFromJSON(
		context.Background(),
		[]byte(cfg.GoogleServiceAccountJSON),
		sheetsScope,
	)
	if err != nil {
		return nil, fmt.Errorf("google credentials: %w", err)
	}

	client := oauth2.NewClient(context.Background(), creds.TokenSource)
	client.Timeout = time.Duration(cfg.HTTPTimeoutSeconds) * time.Second

	sc := &sheetsClient{client: client, sheetID: cfg.GoogleSheetID}
	title, err := sc.fetchFirstSheetTitle()
	if err != nil {
		return nil, fmt.Errorf("fetch sheet title: %w", err)
	}
	sc.firstSheetTitle = title
	return sc, nil
}

// fetchFirstSheetTitle returns the title of the first worksheet in the spreadsheet.
func (sc *sheetsClient) fetchFirstSheetTitle() (string, error) {
	apiURL := fmt.Sprintf("%s/%s?fields=sheets.properties.title", sheetsAPIBase, sc.sheetID)
	resp, err := sc.client.Get(apiURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if err := checkStatus(resp); err != nil {
		return "", fmt.Errorf("spreadsheet metadata: %w", err)
	}
	var meta struct {
		Sheets []struct {
			Properties struct {
				Title string `json:"title"`
			} `json:"properties"`
		} `json:"sheets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
		return "", err
	}
	if len(meta.Sheets) == 0 {
		return "", fmt.Errorf("spreadsheet has no sheets")
	}
	return meta.Sheets[0].Properties.Title, nil
}

// getValues fetches all rows from the given sheet/range name.
func (sc *sheetsClient) getValues(rangeName string) ([][]string, error) {
	apiURL := fmt.Sprintf("%s/%s/values/%s", sheetsAPIBase, sc.sheetID, url.PathEscape(rangeName))

	resp, err := sc.client.Get(apiURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return nil, fmt.Errorf("sheets getValues(%s): %w", rangeName, err)
	}

	var result struct {
		Values [][]string `json:"values"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return result.Values, nil
}

// appendRow appends a single row to the given sheet/range name.
func (sc *sheetsClient) appendRow(rangeName string, row []any) error {
	body, err := json.Marshal(map[string]any{
		"range":          rangeName,
		"majorDimension": "ROWS",
		"values":         [][]any{row},
	})
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=USER_ENTERED",
		sheetsAPIBase, sc.sheetID, url.PathEscape(rangeName))

	resp, err := sc.client.Post(apiURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := checkStatus(resp); err != nil {
		return fmt.Errorf("sheets appendRow(%s): %w", rangeName, err)
	}
	return nil
}

func (sc *sheetsClient) getLastAnnualTotal() (float64, error) {
	values, err := sc.getValues(sc.firstSheetTitle)
	if err != nil {
		return 0, err
	}

	var dataRows [][]string
	for _, row := range values {
		if len(row) > 0 && row[0] != "" && row[0] != "Semana" {
			dataRows = append(dataRows, row)
		}
	}

	if len(dataRows) == 0 {
		slog.Info("No existing rows in sheet — starting from 0 km.")
		return 0, nil
	}

	last := dataRows[len(dataRows)-1]
	if len(last) < 5 {
		slog.Warn("Last row too short to read annual total", "row", last)
		return 0, nil
	}

	total, err := strconv.ParseFloat(last[4], 64)
	if err != nil {
		slog.Warn("Could not parse annual total", "value", last[4])
		return 0, nil
	}

	slog.Info("Last annual total from sheet", "km", total)
	return total, nil
}

func (sc *sheetsClient) hasEntryForWeek(weekNumber int) (bool, error) {
	values, err := sc.getValues(sc.firstSheetTitle)
	if err != nil {
		return false, err
	}

	if len(values) < 2 {
		return false, nil
	}

	for _, row := range values[1:] { // skip header
		if len(row) == 0 {
			continue
		}
		v, err := strconv.Atoi(row[0])
		if err != nil {
			continue
		}
		if v == weekNumber {
			return true, nil
		}
	}
	return false, nil
}

func (sc *sheetsClient) appendWeeklyEntry(
	weekNumber int,
	weekStart, weekEnd time.Time,
	weeklyKM, annualKM float64,
	annualGoal int,
	postText string,
) error {
	row := []any{
		weekNumber,
		weekStart.Format("02-01-2006"),
		weekEnd.Format("02-01-2006"),
		math.Round(weeklyKM*100) / 100,
		math.Round(annualKM*100) / 100,
		annualGoal,
		postText,
		time.Now().UTC().Format("02-01-2006 15:04 UTC"),
	}

	if err := sc.appendRow(sc.firstSheetTitle, row); err != nil {
		return err
	}
	slog.Info("Appended row to sheet", "week", weekNumber)
	return nil
}

func (sc *sheetsClient) ensureHeaderExists() error {
	values, err := sc.getValues(sc.firstSheetTitle)
	if err != nil {
		return err
	}
	if len(values) == 0 || len(values[0]) == 0 {
		if err := sc.appendRow(sc.firstSheetTitle, headerRow); err != nil {
			return err
		}
		slog.Info("Header row written.")
	}
	return nil
}
