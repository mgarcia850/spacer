package spacer

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Note pairs a Card's scheduling state with the content being studied.
type Note struct {
	ID    string `json:"id"`
	Front string `json:"front"`
	Back  string `json:"back"`
	Card  Card   `json:"card"`
}

// ReviewEvent records one graded review. Cards only keep their current
// scheduling state, so this log is the only place review history (and
// therefore anything the stats command reports) can be recovered from.
// PrevCard is the card's state immediately before this review, kept so
// Undo can restore it exactly rather than trying to recompute it.
type ReviewEvent struct {
	NoteID   string    `json:"note_id"`
	Rating   Rating    `json:"rating"`
	Time     time.Time `json:"time"`
	PrevCard Card      `json:"prev_card"`
}

// Deck is a collection of notes, keyed by ID, persisted as one JSON file.
type Deck struct {
	Notes   map[string]*Note `json:"notes"`
	History []ReviewEvent    `json:"history,omitempty"`
	Params  Params           `json:"params"`
}

func NewDeck() *Deck {
	return &Deck{Notes: make(map[string]*Note), Params: DefaultParams()}
}

// LoadDeck reads a deck from disk, returning an empty deck if the file
// doesn't exist yet — a fresh deck shouldn't require a setup step.
func LoadDeck(path string) (*Deck, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return NewDeck(), nil
	}
	if err != nil {
		return nil, err
	}
	deck := NewDeck()
	if err := json.Unmarshal(data, deck); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	if deck.Notes == nil {
		deck.Notes = make(map[string]*Note)
	}
	// Files written before Params existed unmarshal it as the zero value,
	// which isn't usable, so fall back to the defaults it used to hardcode.
	if deck.Params == (Params{}) {
		deck.Params = DefaultParams()
	}
	return deck, nil
}

func (d *Deck) Save(path string) error {
	data, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func (d *Deck) Add(id, front, back string, now time.Time) error {
	if _, exists := d.Notes[id]; exists {
		return fmt.Errorf("note %q already exists", id)
	}
	d.Notes[id] = &Note{ID: id, Front: front, Back: back, Card: NewCardWithParams(now, d.Params)}
	return nil
}

// Grade looks up a note, applies a review to its card, and logs the
// outcome so ComputeStats can report on it later and Undo can revert it.
func (d *Deck) Grade(id string, rating Rating, now time.Time) (*Note, error) {
	note, ok := d.Notes[id]
	if !ok {
		return nil, fmt.Errorf("no note %q", id)
	}
	prev := note.Card
	note.Card = note.Card.ReviewWithParams(rating, now, d.Params)
	d.History = append(d.History, ReviewEvent{NoteID: id, Rating: rating, Time: now, PrevCard: prev})
	return note, nil
}

// Undo reverts the most recently logged review, restoring its note's card
// to the state it had beforehand and removing the entry from History so
// it no longer counts toward stats or a further undo.
//
// Review history logged before PrevCard existed has no recorded prior
// state; PrevCard unmarshals as the zero Card in that case, which is
// never a real card's state since EaseFactor is always positive.
func (d *Deck) Undo() (*Note, error) {
	if len(d.History) == 0 {
		return nil, fmt.Errorf("no reviews to undo")
	}
	last := d.History[len(d.History)-1]
	note, ok := d.Notes[last.NoteID]
	if !ok {
		return nil, fmt.Errorf("note %q from last review no longer exists", last.NoteID)
	}
	if last.PrevCard.EaseFactor == 0 {
		return nil, fmt.Errorf("cannot undo: review predates undo support")
	}
	note.Card = last.PrevCard
	d.History = d.History[:len(d.History)-1]
	return note, nil
}

// Due returns notes whose card is due at or before now, earliest first.
func (d *Deck) Due(now time.Time) []*Note {
	var due []*Note
	for _, n := range d.Notes {
		if !n.Card.Due.After(now) {
			due = append(due, n)
		}
	}
	sort.Slice(due, func(i, j int) bool { return due[i].Card.Due.Before(due[j].Card.Due) })
	return due
}

func (d *Deck) All() []*Note {
	all := make([]*Note, 0, len(d.Notes))
	for _, n := range d.Notes {
		all = append(all, n)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].ID < all[j].ID })
	return all
}

var csvColumns = []string{"id", "front", "back", "interval", "ease", "reps", "due"}

// ExportCSV writes the whole deck, including scheduling state, so it can
// be inspected in a spreadsheet or re-imported elsewhere with ImportCSV.
func (d *Deck) ExportCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(csvColumns); err != nil {
		return err
	}
	for _, n := range d.All() {
		record := []string{
			n.ID,
			n.Front,
			n.Back,
			strconv.FormatFloat(n.Card.Interval, 'f', -1, 64),
			strconv.FormatFloat(n.Card.EaseFactor, 'f', -1, 64),
			strconv.Itoa(n.Card.Repetitions),
			n.Card.Due.Format(time.RFC3339),
		}
		if err := cw.Write(record); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// ImportCSV adds notes from CSV records with a header row naming its
// columns. "id", "front", and "back" are required; if "interval", "ease",
// "reps", and "due" are also present (as written by ExportCSV) each note
// is restored with that scheduling state, otherwise it starts fresh as of
// now. Rows whose ID already exists in the deck are skipped rather than
// erroring, so re-running an import to pick up new rows is safe.
func (d *Deck) ImportCSV(r io.Reader, now time.Time) (added, skipped int, err error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1

	header, err := cr.Read()
	if err == io.EOF {
		return 0, 0, nil
	}
	if err != nil {
		return 0, 0, err
	}
	col := make(map[string]int, len(header))
	for i, h := range header {
		col[strings.ToLower(strings.TrimSpace(h))] = i
	}
	for _, want := range []string{"id", "front", "back"} {
		if _, ok := col[want]; !ok {
			return 0, 0, fmt.Errorf("csv missing required column %q", want)
		}
	}
	hasState := true
	for _, want := range []string{"interval", "ease", "reps", "due"} {
		if _, ok := col[want]; !ok {
			hasState = false
			break
		}
	}

	for {
		record, err := cr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return added, skipped, err
		}

		id := record[col["id"]]
		if _, exists := d.Notes[id]; exists {
			skipped++
			continue
		}

		note := &Note{ID: id, Front: record[col["front"]], Back: record[col["back"]], Card: NewCardWithParams(now, d.Params)}
		if hasState {
			card, err := parseCardCSV(record, col)
			if err != nil {
				return added, skipped, fmt.Errorf("row %q: %w", id, err)
			}
			note.Card = card
		}
		d.Notes[id] = note
		added++
	}
	return added, skipped, nil
}

func parseCardCSV(record []string, col map[string]int) (Card, error) {
	interval, err := strconv.ParseFloat(record[col["interval"]], 64)
	if err != nil {
		return Card{}, fmt.Errorf("invalid interval: %w", err)
	}
	ease, err := strconv.ParseFloat(record[col["ease"]], 64)
	if err != nil {
		return Card{}, fmt.Errorf("invalid ease: %w", err)
	}
	reps, err := strconv.Atoi(record[col["reps"]])
	if err != nil {
		return Card{}, fmt.Errorf("invalid reps: %w", err)
	}
	due, err := time.Parse(time.RFC3339, record[col["due"]])
	if err != nil {
		return Card{}, fmt.Errorf("invalid due: %w", err)
	}
	return Card{Interval: interval, EaseFactor: ease, Repetitions: reps, Due: due}, nil
}
