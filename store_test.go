package spacer

import (
	"strings"
	"testing"
	"time"
)

func TestImportCSVNewNotesFromMinimalColumns(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	csv := "id,front,back\n" +
		"capital-france,capital of France?,Paris\n" +
		"capital-japan,capital of Japan?,Tokyo\n"

	added, skipped, err := deck.ImportCSV(strings.NewReader(csv), now)
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if added != 2 || skipped != 0 {
		t.Fatalf("added = %d, skipped = %d, want 2, 0", added, skipped)
	}

	note, ok := deck.Notes["capital-france"]
	if !ok {
		t.Fatal("capital-france not imported")
	}
	if note.Front != "capital of France?" || note.Back != "Paris" {
		t.Errorf("note = %+v, want front/back set from csv", note)
	}
	if note.Card.EaseFactor != startEase || !note.Card.Due.Equal(now) {
		t.Errorf("imported note without state columns should start fresh, got %+v", note.Card)
	}
}

func TestImportCSVSkipsExistingIDs(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("capital-france", "capital of France?", "Paris", now); err != nil {
		t.Fatal(err)
	}
	csv := "id,front,back\n" +
		"capital-france,capital of France?,Paris\n" +
		"capital-japan,capital of Japan?,Tokyo\n"

	added, skipped, err := deck.ImportCSV(strings.NewReader(csv), now)
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if added != 1 || skipped != 1 {
		t.Fatalf("added = %d, skipped = %d, want 1, 1", added, skipped)
	}
}

func TestImportCSVMissingRequiredColumn(t *testing.T) {
	deck := NewDeck()
	csv := "id,front\ncapital-france,capital of France?\n"

	_, _, err := deck.ImportCSV(strings.NewReader(csv), time.Now())
	if err == nil {
		t.Fatal("ImportCSV with no back column: want error, got nil")
	}
}

func TestImportCSVEmptyInput(t *testing.T) {
	deck := NewDeck()
	added, skipped, err := deck.ImportCSV(strings.NewReader(""), time.Now())
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if added != 0 || skipped != 0 {
		t.Fatalf("added = %d, skipped = %d, want 0, 0", added, skipped)
	}
}

func TestExportImportCSVRoundTrip(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("capital-france", "capital of France?", "Paris", now); err != nil {
		t.Fatal(err)
	}
	deck.Notes["capital-france"].Card = deck.Notes["capital-france"].Card.Review(Good, now)

	var buf strings.Builder
	if err := deck.ExportCSV(&buf); err != nil {
		t.Fatalf("ExportCSV: %v", err)
	}

	restored := NewDeck()
	added, skipped, err := restored.ImportCSV(strings.NewReader(buf.String()), now)
	if err != nil {
		t.Fatalf("ImportCSV: %v", err)
	}
	if added != 1 || skipped != 0 {
		t.Fatalf("added = %d, skipped = %d, want 1, 0", added, skipped)
	}

	want := deck.Notes["capital-france"]
	got := restored.Notes["capital-france"]
	if got == nil {
		t.Fatal("capital-france missing after round trip")
	}
	if got.Front != want.Front || got.Back != want.Back {
		t.Errorf("note content = %+v, want %+v", got, want)
	}
	if got.Card.Interval != want.Card.Interval ||
		got.Card.EaseFactor != want.Card.EaseFactor ||
		got.Card.Repetitions != want.Card.Repetitions ||
		!got.Card.Due.Equal(want.Card.Due) {
		t.Errorf("card state = %+v, want %+v", got.Card, want.Card)
	}
}
