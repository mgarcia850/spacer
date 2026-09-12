package spacer

import (
	"os"
	"path/filepath"
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

func TestLoadDeckFillsDefaultParamsForFileWithoutThem(t *testing.T) {
	path := filepath.Join(t.TempDir(), "deck.json")
	old := `{"notes":{}}`
	if err := os.WriteFile(path, []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}

	deck, err := LoadDeck(path)
	if err != nil {
		t.Fatalf("LoadDeck: %v", err)
	}
	if deck.Params != DefaultParams() {
		t.Errorf("Params = %+v, want %+v", deck.Params, DefaultParams())
	}
}

func TestDeckParamsControlNewCardsAndReviews(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	deck.Params.StartEase = 3.0
	deck.Params.AgainEasePenalty = 0.5

	if err := deck.Add("a", "front", "back", now); err != nil {
		t.Fatal(err)
	}
	if got := deck.Notes["a"].Card.EaseFactor; got != 3.0 {
		t.Errorf("new card EaseFactor = %v, want 3.0 from configured StartEase", got)
	}

	note, err := deck.Grade("a", Again, now)
	if err != nil {
		t.Fatal(err)
	}
	if wantEase := 3.0 - 0.5; note.Card.EaseFactor != wantEase {
		t.Errorf("EaseFactor after again = %v, want %v", note.Card.EaseFactor, wantEase)
	}
}

func TestGradeUpdatesCardAndLogsHistory(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	deck := NewDeck()
	if err := deck.Add("capital-france", "capital of France?", "Paris", now); err != nil {
		t.Fatal(err)
	}

	note, err := deck.Grade("capital-france", Good, now)
	if err != nil {
		t.Fatalf("Grade: %v", err)
	}
	if note.Card.Repetitions != 1 {
		t.Errorf("Repetitions = %v, want 1", note.Card.Repetitions)
	}
	if len(deck.History) != 1 {
		t.Fatalf("len(History) = %d, want 1", len(deck.History))
	}
	got := deck.History[0]
	if got.NoteID != "capital-france" || got.Rating != Good || !got.Time.Equal(now) {
		t.Errorf("History[0] = %+v, want {capital-france Good %v}", got, now)
	}
}

func TestGradeUnknownNote(t *testing.T) {
	deck := NewDeck()
	if _, err := deck.Grade("nope", Good, time.Now()); err == nil {
		t.Fatal("Grade on unknown note: want error, got nil")
	}
	if len(deck.History) != 0 {
		t.Errorf("len(History) = %d, want 0 after failed grade", len(deck.History))
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
