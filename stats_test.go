package spacer

import (
	"testing"
	"time"
)

func TestComputeStatsEmptyDeck(t *testing.T) {
	deck := NewDeck()
	s := deck.ComputeStats(time.Now())

	if s.TotalNotes != 0 || s.DueNow != 0 || s.TotalReviews != 0 {
		t.Fatalf("Stats = %+v, want all zero", s)
	}
	if s.Retention != 0 || s.ReviewsPerDay != 0 {
		t.Errorf("Stats = %+v, want Retention and ReviewsPerDay zero with no history", s)
	}
}

func TestComputeStatsDueNow(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("due-already", "front", "back", now.AddDate(0, 0, -1)); err != nil {
		t.Fatal(err)
	}
	if err := deck.Add("due-later", "front", "back", now.AddDate(0, 0, 5)); err != nil {
		t.Fatal(err)
	}

	s := deck.ComputeStats(now)
	if s.TotalNotes != 2 {
		t.Errorf("TotalNotes = %d, want 2", s.TotalNotes)
	}
	if s.DueNow != 1 {
		t.Errorf("DueNow = %d, want 1", s.DueNow)
	}
}

func TestComputeStatsRetention(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("a", "front", "back", now); err != nil {
		t.Fatal(err)
	}

	// 3 passing reviews, 1 failing: retention should be 75%.
	if _, err := deck.Grade("a", Good, now); err != nil {
		t.Fatal(err)
	}
	if _, err := deck.Grade("a", Easy, now); err != nil {
		t.Fatal(err)
	}
	if _, err := deck.Grade("a", Again, now); err != nil {
		t.Fatal(err)
	}
	if _, err := deck.Grade("a", Hard, now); err != nil {
		t.Fatal(err)
	}

	s := deck.ComputeStats(now)
	if s.TotalReviews != 4 {
		t.Fatalf("TotalReviews = %d, want 4", s.TotalReviews)
	}
	wantRetention := 0.75
	if s.Retention != wantRetention {
		t.Errorf("Retention = %v, want %v", s.Retention, wantRetention)
	}
}

func TestComputeStatsReviewsPerDay(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("a", "front", "back", now); err != nil {
		t.Fatal(err)
	}

	// 4 reviews spread across exactly 2 days: 2 reviews/day.
	if _, err := deck.Grade("a", Good, now); err != nil {
		t.Fatal(err)
	}
	if _, err := deck.Grade("a", Good, now); err != nil {
		t.Fatal(err)
	}
	later := now.AddDate(0, 0, 2)
	if _, err := deck.Grade("a", Good, later); err != nil {
		t.Fatal(err)
	}
	if _, err := deck.Grade("a", Good, later); err != nil {
		t.Fatal(err)
	}

	s := deck.ComputeStats(later)
	wantPerDay := 2.0
	if s.ReviewsPerDay != wantPerDay {
		t.Errorf("ReviewsPerDay = %v, want %v", s.ReviewsPerDay, wantPerDay)
	}
}

func TestComputeStatsReviewsPerDaySameDayDoesNotInflate(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("a", "front", "back", now); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 3; i++ {
		if _, err := deck.Grade("a", Good, now); err != nil {
			t.Fatal(err)
		}
	}

	s := deck.ComputeStats(now)
	wantPerDay := 3.0
	if s.ReviewsPerDay != wantPerDay {
		t.Errorf("ReviewsPerDay = %v, want %v (all reviews same instant, min 1 day)", s.ReviewsPerDay, wantPerDay)
	}
}
