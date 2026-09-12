package spacer

import (
	"math"
	"testing"
	"time"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func TestNewCard(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCard(now)

	if c.Interval != 0 {
		t.Errorf("Interval = %v, want 0", c.Interval)
	}
	if c.EaseFactor != startEase {
		t.Errorf("EaseFactor = %v, want %v", c.EaseFactor, startEase)
	}
	if c.Repetitions != 0 {
		t.Errorf("Repetitions = %v, want 0", c.Repetitions)
	}
	if !c.Due.Equal(now) {
		t.Errorf("Due = %v, want %v", c.Due, now)
	}
}

func TestReviewGoodFirstThreeSteps(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCard(now)

	c = c.Review(Good, now)
	if !almostEqual(c.Interval, 1) {
		t.Fatalf("after 1st good, Interval = %v, want 1", c.Interval)
	}
	if c.Repetitions != 1 {
		t.Fatalf("after 1st good, Repetitions = %v, want 1", c.Repetitions)
	}

	c = c.Review(Good, now)
	if !almostEqual(c.Interval, 6) {
		t.Fatalf("after 2nd good, Interval = %v, want 6", c.Interval)
	}
	if c.Repetitions != 2 {
		t.Fatalf("after 2nd good, Repetitions = %v, want 2", c.Repetitions)
	}

	c = c.Review(Good, now)
	wantInterval := 6 * startEase
	if !almostEqual(c.Interval, wantInterval) {
		t.Fatalf("after 3rd good, Interval = %v, want %v", c.Interval, wantInterval)
	}
	if c.Repetitions != 3 {
		t.Fatalf("after 3rd good, Repetitions = %v, want 3", c.Repetitions)
	}
	// Good reviews shouldn't touch the ease factor.
	if c.EaseFactor != startEase {
		t.Fatalf("after 3rd good, EaseFactor = %v, want unchanged %v", c.EaseFactor, startEase)
	}
}

func TestReviewAgainResetsStreak(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCard(now)
	c = c.Review(Good, now)
	c = c.Review(Good, now)
	c = c.Review(Good, now) // reps=3, interval=6*2.5=15, ease=2.5

	before := c
	c = c.Review(Again, now)

	if c.Repetitions != 0 {
		t.Errorf("Repetitions = %v, want 0", c.Repetitions)
	}
	if !almostEqual(c.Interval, 1) {
		t.Errorf("Interval = %v, want 1", c.Interval)
	}
	wantEase := before.EaseFactor - 0.20
	if !almostEqual(c.EaseFactor, wantEase) {
		t.Errorf("EaseFactor = %v, want %v", c.EaseFactor, wantEase)
	}
	wantDue := now.AddDate(0, 0, 1)
	if !c.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", c.Due, wantDue)
	}
}

func TestReviewAgainClampsEaseAtMinimum(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCard(now)
	c.EaseFactor = minEase + 0.05

	c = c.Review(Again, now)

	if c.EaseFactor != minEase {
		t.Errorf("EaseFactor = %v, want clamped to %v", c.EaseFactor, minEase)
	}
}

func TestReviewHardClampsEaseAtMinimum(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCard(now)
	c.EaseFactor = minEase + 0.05
	c.Interval = 10

	c = c.Review(Hard, now)

	if c.EaseFactor != minEase {
		t.Errorf("EaseFactor = %v, want clamped to %v", c.EaseFactor, minEase)
	}
	wantInterval := 10 * 1.2
	if !almostEqual(c.Interval, wantInterval) {
		t.Errorf("Interval = %v, want %v", c.Interval, wantInterval)
	}
}

func TestReviewHardGrowsIntervalModestly(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := Card{Interval: 20, EaseFactor: 2.0, Repetitions: 4, Due: now}

	next := c.Review(Hard, now)

	wantInterval := 20 * 1.2
	if !almostEqual(next.Interval, wantInterval) {
		t.Errorf("Interval = %v, want %v", next.Interval, wantInterval)
	}
	wantEase := 2.0 - 0.15
	if !almostEqual(next.EaseFactor, wantEase) {
		t.Errorf("EaseFactor = %v, want %v", next.EaseFactor, wantEase)
	}
	if next.Repetitions != 5 {
		t.Errorf("Repetitions = %v, want 5", next.Repetitions)
	}
}

func TestReviewEasyGrowsIntervalAndEase(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := Card{Interval: 10, EaseFactor: 2.0, Repetitions: 3, Due: now}

	next := c.Review(Easy, now)

	wantEase := 2.0 + 0.15
	if !almostEqual(next.EaseFactor, wantEase) {
		t.Errorf("EaseFactor = %v, want %v", next.EaseFactor, wantEase)
	}
	wantInterval := 10 * wantEase * 1.3
	if !almostEqual(next.Interval, wantInterval) {
		t.Errorf("Interval = %v, want %v", next.Interval, wantInterval)
	}
	if next.Repetitions != 4 {
		t.Errorf("Repetitions = %v, want 4", next.Repetitions)
	}
}

func TestReviewIntervalNeverBelowOneDay(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// A fresh card's interval is 0, so even a small multiplicative step
	// (Hard, Easy) must still clamp up to at least one day.
	hard := NewCard(now).Review(Hard, now)
	if hard.Interval < 1 {
		t.Errorf("Hard from zero interval = %v, want >= 1", hard.Interval)
	}

	easy := NewCard(now).Review(Easy, now)
	if easy.Interval < 1 {
		t.Errorf("Easy from zero interval = %v, want >= 1", easy.Interval)
	}
}

func TestReviewDueDateRoundsIntervalToWholeDays(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := Card{Interval: 10, EaseFactor: 2.36, Repetitions: 5, Due: now}

	next := c.Review(Hard, now) // interval = 10 * 1.2 = 12, exact

	wantDue := now.AddDate(0, 0, 12)
	if !next.Due.Equal(wantDue) {
		t.Errorf("Due = %v, want %v", next.Due, wantDue)
	}
}

func TestReviewDoesNotMutateReceiver(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := NewCard(now)
	original := c

	c.Review(Good, now)

	if c != original {
		t.Errorf("Review mutated the receiver: got %+v, want %+v", c, original)
	}
}

func TestReviewWithParamsUsesCustomConstants(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	params := Params{
		StartEase:              3.0,
		MinEase:                1.0,
		AgainEasePenalty:       0.5,
		HardEasePenalty:        0.3,
		HardIntervalMultiplier: 1.5,
		EasyEaseBonus:          0.4,
		EasyIntervalMultiplier: 2.0,
	}
	c := NewCardWithParams(now, params)
	if c.EaseFactor != params.StartEase {
		t.Fatalf("EaseFactor = %v, want %v", c.EaseFactor, params.StartEase)
	}

	hard := Card{Interval: 10, EaseFactor: 2.0, Repetitions: 3, Due: now}.ReviewWithParams(Hard, now, params)
	if wantEase := 2.0 - params.HardEasePenalty; hard.EaseFactor != wantEase {
		t.Errorf("Hard EaseFactor = %v, want %v", hard.EaseFactor, wantEase)
	}
	if wantInterval := 10 * params.HardIntervalMultiplier; !almostEqual(hard.Interval, wantInterval) {
		t.Errorf("Hard Interval = %v, want %v", hard.Interval, wantInterval)
	}

	easy := Card{Interval: 10, EaseFactor: 2.0, Repetitions: 3, Due: now}.ReviewWithParams(Easy, now, params)
	wantEase := 2.0 + params.EasyEaseBonus
	if easy.EaseFactor != wantEase {
		t.Errorf("Easy EaseFactor = %v, want %v", easy.EaseFactor, wantEase)
	}
	if wantInterval := 10 * wantEase * params.EasyIntervalMultiplier; !almostEqual(easy.Interval, wantInterval) {
		t.Errorf("Easy Interval = %v, want %v", easy.Interval, wantInterval)
	}

	again := Card{Interval: 10, EaseFactor: params.MinEase + 0.1, Repetitions: 3, Due: now}.ReviewWithParams(Again, now, params)
	if again.EaseFactor != params.MinEase {
		t.Errorf("Again EaseFactor = %v, want clamped to %v", again.EaseFactor, params.MinEase)
	}
}

func TestReviewMatchesReviewWithDefaultParams(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	c := Card{Interval: 10, EaseFactor: 2.0, Repetitions: 3, Due: now}

	for _, r := range []Rating{Again, Hard, Good, Easy} {
		got := c.Review(r, now)
		want := c.ReviewWithParams(r, now, DefaultParams())
		if got != want {
			t.Errorf("Review(%v) = %+v, want %+v", r, got, want)
		}
	}
}

func TestParseRatingRoundTrip(t *testing.T) {
	for _, r := range []Rating{Again, Hard, Good, Easy} {
		got, ok := ParseRating(r.String())
		if !ok {
			t.Errorf("ParseRating(%q) not ok", r.String())
		}
		if got != r {
			t.Errorf("ParseRating(%q) = %v, want %v", r.String(), got, r)
		}
	}
}

func TestParseRatingInvalid(t *testing.T) {
	if _, ok := ParseRating("bogus"); ok {
		t.Error("ParseRating(\"bogus\") = ok, want not ok")
	}
}
