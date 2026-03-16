package strava

import (
	"testing"
	"time"
)

func TestWeekStartEpoch(t *testing.T) {
	forDate := time.Date(2026, time.March, 20, 17, 30, 0, 0, time.UTC)
	epoch := WeekStartEpoch(forDate)
	got := time.Unix(epoch, 0).UTC().Format("2006-01-02")

	if got != "2026-03-16" {
		t.Fatalf("expected Monday 2026-03-16, got %s", got)
	}
}

func TestSumWeeklyDistanceKMAllTypes(t *testing.T) {
	activities := []Activity{
		{Distance: 10000, SportType: "Run"},
		{Distance: 2500, SportType: "Walk"},
		{Distance: 1500, Type: "Run"},
	}

	km := SumWeeklyDistanceKM(activities, nil)
	if km != 14 {
		t.Fatalf("expected 14.00 km, got %.2f", km)
	}
}

func TestSumWeeklyDistanceKMFilteredTypes(t *testing.T) {
	activities := []Activity{
		{Distance: 10000, SportType: "Run"},
		{Distance: 2500, SportType: "Walk"},
		{Distance: 1500, Type: "Run"},
	}

	km := SumWeeklyDistanceKM(activities, []string{"Run"})
	if km != 11.5 {
		t.Fatalf("expected 11.50 km, got %.2f", km)
	}
}
