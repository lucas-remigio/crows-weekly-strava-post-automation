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
	values, err := sc.getValues(athletesSheet)
	if err != nil {
		slog.Warn("Athletes worksheet not found or empty — skipping roast", "error", err)
		return nil
	}

	if len(values) < 2 {
		slog.Warn("Athletes worksheet is empty — skipping roast.")
		return nil
	}

	headers := values[0]
	nameIdx := headerIndex(headers, "nome", "name")
	charIdx := headerIndex(headers, "caracteristica", "característica", "characteristic")

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

func headerIndex(headers []string, aliases ...string) int {
	allowed := make(map[string]struct{}, len(aliases))
	for _, a := range aliases {
		allowed[strings.ToLower(strings.TrimSpace(a))] = struct{}{}
	}

	for i, h := range headers {
		normalized := strings.ToLower(strings.TrimSpace(h))
		if _, ok := allowed[normalized]; ok {
			return i
		}
	}

	return -1
}
