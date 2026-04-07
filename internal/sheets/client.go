package sheets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
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
	apiBase        = "https://sheets.googleapis.com/v4/spreadsheets"
	scope          = "https://www.googleapis.com/auth/spreadsheets"
	AthletesSheet  = "Atletas"
	weekHeader     = "Semana"
	weekCol        = 0
	annualTotalCol = 4
	dateLayout     = "02-01-2006"
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

type Client struct {
	client          *http.Client
	sheetID         string
	firstSheetTitle string
}

func NewClient(serviceAccountJSON, sheetID string, timeoutSeconds int) (*Client, error) {
	creds, err := google.CredentialsFromJSON(context.Background(), []byte(serviceAccountJSON), scope)
	if err != nil {
		return nil, fmt.Errorf("google credentials: %w", err)
	}

	httpClient := oauth2.NewClient(context.Background(), creds.TokenSource)
	httpClient.Timeout = time.Duration(timeoutSeconds) * time.Second

	sc := &Client{client: httpClient, sheetID: sheetID}
	title, err := sc.fetchFirstSheetTitle()
	if err != nil {
		return nil, fmt.Errorf("fetch sheet title: %w", err)
	}
	sc.firstSheetTitle = title
	return sc, nil
}

func (sc *Client) fetchFirstSheetTitle() (string, error) {
	apiURL := fmt.Sprintf("%s/%s?fields=sheets.properties.title", apiBase, sc.sheetID)
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

func (sc *Client) GetValues(rangeName string) ([][]string, error) {
	apiURL := fmt.Sprintf("%s/%s/values/%s", apiBase, sc.sheetID, url.PathEscape(rangeName))
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

func (sc *Client) appendRow(rangeName string, row []any) error {
	body, err := json.Marshal(map[string]any{
		"range":          rangeName,
		"majorDimension": "ROWS",
		"values":         [][]any{row},
	})
	if err != nil {
		return err
	}

	apiURL := fmt.Sprintf("%s/%s/values/%s:append?valueInputOption=USER_ENTERED", apiBase, sc.sheetID, url.PathEscape(rangeName))
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

func (sc *Client) GetLastAnnualTotal() (float64, error) {
	values, err := sc.GetValues(sc.firstSheetTitle)
	if err != nil {
		return 0, err
	}

	var dataRows [][]string
	for _, row := range values {
		if isWeekDataRow(row) {
			dataRows = append(dataRows, row)
		}
	}

	if len(dataRows) == 0 {
		slog.Info("No existing rows in sheet — starting from 0 km.")
		return 0, nil
	}

	last := dataRows[len(dataRows)-1]
	if len(last) <= annualTotalCol {
		slog.Warn("Last row too short to read annual total", "row", last)
		return 0, nil
	}

	total, err := strconv.ParseFloat(last[annualTotalCol], 64)
	if err != nil {
		slog.Warn("Could not parse annual total", "value", last[annualTotalCol])
		return 0, nil
	}

	slog.Info("Last annual total from sheet", "km", total)
	return total, nil
}

func (sc *Client) HasEntryForWeek(weekNumber int) (bool, error) {
	values, err := sc.GetValues(sc.firstSheetTitle)
	if err != nil {
		return false, err
	}
	if len(values) < 2 {
		return false, nil
	}

	for _, row := range values[1:] {
		if len(row) == 0 {
			continue
		}
		v, err := strconv.Atoi(row[weekCol])
		if err != nil {
			continue
		}
		if v == weekNumber {
			return true, nil
		}
	}
	return false, nil
}

func (sc *Client) AppendWeeklyEntry(
	weekNumber int,
	weekStart, weekEnd time.Time,
	weeklyKM, annualKM float64,
	annualGoal int,
	postText string,
) error {
	row := []any{
		weekNumber,
		weekStart.Format(dateLayout),
		weekEnd.Format(dateLayout),
		math.Round(weeklyKM*100) / 100,
		math.Round(annualKM*100) / 100,
		annualGoal,
		postText,
		time.Now().UTC().Format(dateLayout + " 15:04 UTC"),
	}

	if err := sc.appendRow(sc.firstSheetTitle, row); err != nil {
		return err
	}
	slog.Info("Appended row to sheet", "week", weekNumber)
	return nil
}

func (sc *Client) EnsureHeaderExists() error {
	values, err := sc.GetValues(sc.firstSheetTitle)
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

func isWeekDataRow(row []string) bool {
	return len(row) > weekCol && row[weekCol] != "" && row[weekCol] != weekHeader
}

func checkStatus(resp *http.Response) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("HTTP %s: %s", resp.Status, body)
}
