// Command spacer is a thin CLI over the spacer library: add notes,
// grade reviews, and see what's due, all backed by one JSON file.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"spacer"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "spacer:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: spacer <add|review|due|list> ...")
	}

	path := deckPath()
	cmd, rest := args[0], args[1:]

	switch cmd {
	case "add":
		return cmdAdd(path, rest)
	case "review":
		return cmdReview(path, rest)
	case "due":
		return cmdDue(path, rest)
	case "list":
		return cmdList(path, rest)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func deckPath() string {
	if p := os.Getenv("SPACER_DECK"); p != "" {
		return p
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "spacer-deck.json"
	}
	return filepath.Join(home, ".spacer", "deck.json")
}

func cmdAdd(path string, args []string) error {
	fs := flag.NewFlagSet("add", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 3 {
		return fmt.Errorf("usage: spacer add <id> <front> <back>")
	}

	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	if err := deck.Add(fs.Arg(0), fs.Arg(1), fs.Arg(2), time.Now()); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return deck.Save(path)
}

func cmdReview(path string, args []string) error {
	fs := flag.NewFlagSet("review", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: spacer review <id> <again|hard|good|easy>")
	}
	rating, ok := spacer.ParseRating(fs.Arg(1))
	if !ok {
		return fmt.Errorf("invalid rating %q (want again, hard, good, or easy)", fs.Arg(1))
	}

	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	note, ok := deck.Notes[fs.Arg(0)]
	if !ok {
		return fmt.Errorf("no note %q", fs.Arg(0))
	}

	note.Card = note.Card.Review(rating, time.Now())
	fmt.Printf("%s: next review in %.0f day(s), due %s\n",
		note.ID, note.Card.Interval, note.Card.Due.Format("2006-01-02"))
	return deck.Save(path)
}

func cmdDue(path string, args []string) error {
	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	for _, n := range deck.Due(time.Now()) {
		fmt.Printf("%s\t%s\n", n.ID, n.Front)
	}
	return nil
}

func cmdList(path string, args []string) error {
	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	for _, n := range deck.All() {
		fmt.Printf("%s\t%s\tdue %s\n", n.ID, n.Front, n.Card.Due.Format("2006-01-02"))
	}
	return nil
}
