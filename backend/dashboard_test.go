package main

import (
	"strings"
	"testing"
	"time"
)

func TestParseDateRange(t *testing.T) {
	from, to, err := parseDateRange("2026-01-02", "2026-01-03")
	if err != nil {
		t.Fatal(err)
	}
	if from.Location() != time.UTC || to.Sub(from) != 48*time.Hour {
		t.Fatalf("unexpected UTC range: %v to %v", from, to)
	}
	if _, _, err := parseDateRange("not-a-date", ""); err == nil {
		t.Fatal("expected malformed date to fail")
	}
	if _, _, err := parseDateRange("2026-01-04", "2026-01-03"); err == nil {
		t.Fatal("expected reversed range to fail")
	}
	if _, _, err := parseDateRange("2024-01-01", "2026-01-01"); err == nil {
		t.Fatal("expected oversized range to fail")
	}
}

func TestOptionalDateStringConversion(t *testing.T) {
	if got := fromDateString(time.Time{}); got != "" {
		t.Fatalf("zero start date = %q, want empty", got)
	}
	if got := toDateString(time.Time{}); got != "" {
		t.Fatalf("zero end date = %q, want empty", got)
	}
	_, inclusiveEnd, err := parseDateRange("", "2026-01-03")
	if err != nil {
		t.Fatal(err)
	}
	if got := toDateString(inclusiveEnd); got != "2026-01-03" {
		t.Fatalf("inclusive end date = %q, want 2026-01-03", got)
	}
}

func TestVotePercentHandlesZeroVotes(t *testing.T) {
	if got := votePercent(0, 0); got != 0 {
		t.Fatalf("zero total percent = %v, want 0", got)
	}
	if got := votePercent(2, 0); got != 0 {
		t.Fatalf("nonzero count with zero total percent = %v, want 0", got)
	}
	if got := votePercent(1, 4); got != 25 {
		t.Fatalf("percent = %v, want 25", got)
	}
}

func TestAssistantFallbackUsesActualAggregateAndLabelsMode(t *testing.T) {
	blank := assistantFallback("Summarize my analytics", AnalyticsResponse{})
	if !strings.Contains(blank, "don't have any polls") {
		t.Fatalf("empty account response was not empty-state guidance: %q", blank)
	}
	response := assistantFallback("Which poll has the most votes?", AnalyticsResponse{
		TotalPolls: 2, ActivePolls: 1, ClosedPolls: 1, TotalVotes: 7, AverageVotesPerPoll: 3.5,
		HighestParticipation: &PollMetric{Question: "Preferred lunch?", TotalVotes: 5},
		MostSelectedOption:   &OptionMetric{Text: "Salad", Votes: 3},
	})
	for _, expected := range []string{"2 polls", "7 total votes", "Preferred lunch?", "Salad"} {
		if !strings.Contains(response, expected) {
			t.Errorf("response %q missing %q", response, expected)
		}
	}
}
