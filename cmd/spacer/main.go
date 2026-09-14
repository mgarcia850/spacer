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
		return fmt.Errorf("usage: spacer <add|review|undo|due|list|import|export|stats|config> ...")
	}

	path := deckPath()
	cmd, rest := args[0], args[1:]

	switch cmd {
	case "add":
		return cmdAdd(path, rest)
	case "review":
		return cmdReview(path, rest)
	case "undo":
		return cmdUndo(path, rest)
	case "due":
		return cmdDue(path, rest)
	case "list":
		return cmdList(path, rest)
	case "import":
		return cmdImport(path, rest)
	case "export":
		return cmdExport(path, rest)
	case "stats":
		return cmdStats(path, rest)
	case "config":
		return cmdConfig(path, rest)
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
	note, err := deck.Grade(fs.Arg(0), rating, time.Now())
	if err != nil {
		return err
	}

	fmt.Printf("%s: next review in %.0f day(s), due %s\n",
		note.ID, note.Card.Interval, note.Card.Due.Format("2006-01-02"))
	return deck.Save(path)
}

func cmdUndo(path string, args []string) error {
	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	note, err := deck.Undo()
	if err != nil {
		return err
	}
	fmt.Printf("%s: undone, back to interval %.0f day(s), due %s\n",
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

func cmdImport(path string, args []string) error {
	fs := flag.NewFlagSet("import", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() != 1 {
		return fmt.Errorf("usage: spacer import <file.csv>")
	}

	f, err := os.Open(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()

	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	added, skipped, err := deck.ImportCSV(f, time.Now())
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := deck.Save(path); err != nil {
		return err
	}
	fmt.Printf("imported %d note(s), skipped %d already present\n", added, skipped)
	return nil
}

func cmdExport(path string, args []string) error {
	fs := flag.NewFlagSet("export", flag.ExitOnError)
	fs.Parse(args)

	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}

	if fs.NArg() == 0 {
		return deck.ExportCSV(os.Stdout)
	}
	f, err := os.Create(fs.Arg(0))
	if err != nil {
		return err
	}
	defer f.Close()
	return deck.ExportCSV(f)
}

func cmdStats(path string, args []string) error {
	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}
	s := deck.ComputeStats(time.Now())

	fmt.Printf("notes:          %d\n", s.TotalNotes)
	fmt.Printf("due now:        %d\n", s.DueNow)
	fmt.Printf("reviews logged: %d\n", s.TotalReviews)
	if s.TotalReviews > 0 {
		fmt.Printf("retention:      %.0f%%\n", s.Retention*100)
		fmt.Printf("reviews/day:    %.1f\n", s.ReviewsPerDay)
	}
	return nil
}

// cmdConfig prints the deck's current scheduling parameters, and updates
// any for which a flag was passed. Flags default to 0, which is never a
// meaningful value for any of these fields, so a 0 reliably means "leave
// this one alone".
func cmdConfig(path string, args []string) error {
	fs := flag.NewFlagSet("config", flag.ExitOnError)
	startEase := fs.Float64("start-ease", 0, "ease factor assigned to new cards (default 2.5)")
	minEase := fs.Float64("min-ease", 0, "floor ease can't drop below (default 1.3)")
	againPenalty := fs.Float64("again-penalty", 0, "ease reduction on an again rating (default 0.20)")
	hardPenalty := fs.Float64("hard-penalty", 0, "ease reduction on a hard rating (default 0.15)")
	hardInterval := fs.Float64("hard-interval", 0, "interval multiplier on a hard rating (default 1.2)")
	easyBonus := fs.Float64("easy-bonus", 0, "ease increase on an easy rating (default 0.15)")
	easyInterval := fs.Float64("easy-interval", 0, "interval multiplier on an easy rating (default 1.3)")
	fs.Parse(args)

	deck, err := spacer.LoadDeck(path)
	if err != nil {
		return err
	}

	changed := false
	set := func(dst *float64, v float64) {
		if v != 0 {
			*dst = v
			changed = true
		}
	}
	set(&deck.Params.StartEase, *startEase)
	set(&deck.Params.MinEase, *minEase)
	set(&deck.Params.AgainEasePenalty, *againPenalty)
	set(&deck.Params.HardEasePenalty, *hardPenalty)
	set(&deck.Params.HardIntervalMultiplier, *hardInterval)
	set(&deck.Params.EasyEaseBonus, *easyBonus)
	set(&deck.Params.EasyIntervalMultiplier, *easyInterval)

	p := deck.Params
	fmt.Printf("start-ease:    %v\n", p.StartEase)
	fmt.Printf("min-ease:      %v\n", p.MinEase)
	fmt.Printf("again-penalty: %v\n", p.AgainEasePenalty)
	fmt.Printf("hard-penalty:  %v\n", p.HardEasePenalty)
	fmt.Printf("hard-interval: %v\n", p.HardIntervalMultiplier)
	fmt.Printf("easy-bonus:    %v\n", p.EasyEaseBonus)
	fmt.Printf("easy-interval: %v\n", p.EasyIntervalMultiplier)

	if !changed {
		return nil
	}
	return deck.Save(path)
}
