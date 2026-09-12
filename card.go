package spacer

import (
	"math"
	"time"
)

// Rating is the grade a person gives their own recall when reviewing a card.
type Rating int

const (
	Again Rating = iota
	Hard
	Good
	Easy
)

func (r Rating) String() string {
	switch r {
	case Again:
		return "again"
	case Hard:
		return "hard"
	case Good:
		return "good"
	case Easy:
		return "easy"
	default:
		return "unknown"
	}
}

func ParseRating(s string) (Rating, bool) {
	switch s {
	case "again":
		return Again, true
	case "hard":
		return Hard, true
	case "good":
		return Good, true
	case "easy":
		return Easy, true
	}
	return 0, false
}

// Card holds the scheduling state for one item. The zero value is not
// a valid starting point; use NewCard.
type Card struct {
	Interval    float64   `json:"interval"` // days until next review
	EaseFactor  float64   `json:"ease"`
	Repetitions int       `json:"reps"`
	Due         time.Time `json:"due"`
}

const (
	startEase = 2.5
	minEase   = 1.3
)

// Params holds the constants that drive how ease and interval move in
// response to a rating. The zero value is not usable; use DefaultParams,
// which reproduces the fixed behavior this package used to hardcode.
type Params struct {
	StartEase              float64
	MinEase                float64
	AgainEasePenalty       float64
	HardEasePenalty        float64
	HardIntervalMultiplier float64
	EasyEaseBonus          float64
	EasyIntervalMultiplier float64
}

func DefaultParams() Params {
	return Params{
		StartEase:              startEase,
		MinEase:                minEase,
		AgainEasePenalty:       0.20,
		HardEasePenalty:        0.15,
		HardIntervalMultiplier: 1.2,
		EasyEaseBonus:          0.15,
		EasyIntervalMultiplier: 1.3,
	}
}

func NewCard(now time.Time) Card {
	return NewCardWithParams(now, DefaultParams())
}

func NewCardWithParams(now time.Time, p Params) Card {
	return Card{Interval: 0, EaseFactor: p.StartEase, Repetitions: 0, Due: now}
}

// Review grades one recall attempt using the package's default constants.
// It's kept around as a convenience for callers that don't need per-deck
// tuning; see ReviewWithParams for the configurable form.
func (c Card) Review(rating Rating, now time.Time) Card {
	return c.ReviewWithParams(rating, now, DefaultParams())
}

// ReviewWithParams grades one recall attempt and returns the resulting
// card state.
//
// This follows SM-2's shape but swaps its 0-5 quality score for the
// four-button scheme Anki popularized, since nobody grading their own
// recall from a terminal wants to pick a number between 0 and 5.
func (c Card) ReviewWithParams(rating Rating, now time.Time, p Params) Card {
	next := c

	switch rating {
	case Again:
		next.Repetitions = 0
		next.Interval = 1
		next.EaseFactor = math.Max(p.MinEase, c.EaseFactor-p.AgainEasePenalty)
	case Hard:
		next.Repetitions = c.Repetitions + 1
		next.EaseFactor = math.Max(p.MinEase, c.EaseFactor-p.HardEasePenalty)
		next.Interval = math.Max(1, c.Interval*p.HardIntervalMultiplier)
	case Good:
		next.Repetitions = c.Repetitions + 1
		next.Interval = goodInterval(c.Interval, next.Repetitions, c.EaseFactor)
	case Easy:
		next.Repetitions = c.Repetitions + 1
		next.EaseFactor = c.EaseFactor + p.EasyEaseBonus
		next.Interval = math.Max(1, c.Interval*next.EaseFactor*p.EasyIntervalMultiplier)
	}

	next.Due = now.AddDate(0, 0, int(math.Round(next.Interval)))
	return next
}

// goodInterval mirrors SM-2's fixed first two steps (1 day, then 6 days)
// before the ease factor takes over driving growth.
func goodInterval(prevInterval float64, reps int, ease float64) float64 {
	switch reps {
	case 1:
		return 1
	case 2:
		return 6
	default:
		return prevInterval * ease
	}
}
