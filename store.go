package spacer

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"
)

// Note pairs a Card's scheduling state with the content being studied.
type Note struct {
	ID    string `json:"id"`
	Front string `json:"front"`
	Back  string `json:"back"`
	Card  Card   `json:"card"`
}

// Deck is a collection of notes, keyed by ID, persisted as one JSON file.
type Deck struct {
	Notes map[string]*Note `json:"notes"`
}

func NewDeck() *Deck {
	return &Deck{Notes: make(map[string]*Note)}
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
	d.Notes[id] = &Note{ID: id, Front: front, Back: back, Card: NewCard(now)}
	return nil
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
