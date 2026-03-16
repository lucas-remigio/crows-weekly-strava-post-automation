package main

import (
	"log/slog"
	"strings"
)

type Athlete struct {
	Name           string
	Characteristic string
}

func getAthletes(sc *sheetsClient) []Athlete {
	values, err := sc.getValues("Atletas")
	if err != nil {
		slog.Warn("Athletes worksheet not found or empty — skipping roast", "error", err)
		return nil
	}

	if len(values) < 2 {
		slog.Warn("Athletes worksheet is empty — skipping roast.")
		return nil
	}

	headers := values[0]
	nameIdx, charIdx := -1, -1

	nameOptions := []string{"nome", "name"}
	charOptions := []string{"caracteristica", "característica", "characteristic"}

	for i, h := range headers {
		lower := strings.ToLower(strings.TrimSpace(h))
		for _, opt := range nameOptions {
			if lower == opt {
				nameIdx = i
			}
		}
		for _, opt := range charOptions {
			if lower == opt {
				charIdx = i
			}
		}
	}

	if nameIdx == -1 || charIdx == -1 {
		slog.Warn("Athletes worksheet missing Nome or Caracteristica columns.")
		return nil
	}

	var athletes []Athlete
	for _, row := range values[1:] {
		name := safeGet(row, nameIdx)
		if name == "" {
			continue
		}
		athletes = append(athletes, Athlete{
			Name:           name,
			Characteristic: safeGet(row, charIdx),
		})
	}

	if len(athletes) == 0 {
		slog.Warn("Athletes worksheet has no valid rows — skipping roast.")
	}
	return athletes
}

func safeGet(row []string, idx int) string {
	if idx < 0 || idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}
