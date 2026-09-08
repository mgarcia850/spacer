package spacer

import "time"

// Stats summarizes a deck's current standing and review history.
type Stats struct {
	TotalNotes    int
	DueNow        int
	TotalReviews  int
	Retention     float64 // fraction of logged reviews rated above Again
	ReviewsPerDay float64 // average logged reviews per calendar day since the first one
}

// ComputeStats summarizes d as of now. Retention and ReviewsPerDay are left
// at zero when there's no review history yet, rather than dividing by zero.
func (d *Deck) ComputeStats(now time.Time) Stats {
	s := Stats{
		TotalNotes: len(d.Notes),
		DueNow:     len(d.Due(now)),
	}
	if len(d.History) == 0 {
		return s
	}
	s.TotalReviews = len(d.History)

	passed := 0
	first, last := d.History[0].Time, d.History[0].Time
	for _, e := range d.History {
		if e.Rating != Again {
			passed++
		}
		if e.Time.Before(first) {
			first = e.Time
		}
		if e.Time.After(last) {
			last = e.Time
		}
	}
	s.Retention = float64(passed) / float64(s.TotalReviews)

	// A single day (or a deck reviewed entirely within one) shouldn't
	// inflate the average by dividing by less than a day.
	days := last.Sub(first).Hours() / 24
	if days < 1 {
		days = 1
	}
	s.ReviewsPerDay = float64(s.TotalReviews) / days

	return s
}
