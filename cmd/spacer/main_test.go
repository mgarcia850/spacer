package main

import (
	"path/filepath"
	"testing"
)

func TestResolveDeckPathDefaultsToDefault(t *testing.T) {
	t.Setenv("SPACER_DECK_DIR", "/decks")
	t.Setenv("SPACER_DECK", "")

	path, err := resolveDeckPath("")
	if err != nil {
		t.Fatalf("resolveDeckPath: %v", err)
	}
	if want := filepath.Join("/decks", "default.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestResolveDeckPathUsesName(t *testing.T) {
	t.Setenv("SPACER_DECK_DIR", "/decks")

	path, err := resolveDeckPath("spanish")
	if err != nil {
		t.Fatalf("resolveDeckPath: %v", err)
	}
	if want := filepath.Join("/decks", "spanish.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestResolveDeckPathFallsBackToEnvVar(t *testing.T) {
	t.Setenv("SPACER_DECK_DIR", "/decks")
	t.Setenv("SPACER_DECK", "spanish")

	path, err := resolveDeckPath("")
	if err != nil {
		t.Fatalf("resolveDeckPath: %v", err)
	}
	if want := filepath.Join("/decks", "spanish.json"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
}

func TestResolveDeckPathRejectsPathTraversal(t *testing.T) {
	for _, name := range []string{"../escape", "a/b", "."} {
		if _, err := resolveDeckPath(name); err == nil {
			t.Errorf("resolveDeckPath(%q): want error, got nil", name)
		}
	}
}
