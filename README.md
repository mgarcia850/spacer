# spacer

A spaced repetition scheduler: a small Go library plus a CLI that uses it.

## The problem

If you're trying to remember something long-term — vocabulary, API
signatures, whatever — reviewing it on a fixed schedule wastes time on
stuff you already know and under-reviews stuff you're about to forget.
Spaced repetition fixes this by tracking, per item, how well you
recalled it last time and picking the next review date so it lands
right before you'd have forgotten it. Get the recall grading right and
the intervals stretch out on their own: things you know well get
reviewed every few months, things you keep flubbing come back daily.

`spacer` implements the scheduling half of that (an SM-2 variant, the
algorithm behind Anki and the original SuperMemo). It doesn't do
flashcard UI, sync, or media — just: given a card's history and how
you rated your recall, when should it come back?

## Library

```go
import "spacer"

c := spacer.NewCard(time.Now())
c = c.Review(spacer.Good, time.Now()) // due tomorrow
c = c.Review(spacer.Good, time.Now()) // due in 6 days
c = c.Review(spacer.Good, time.Now()) // due in ~15 days (ease-driven)
```

Four ratings: `Again`, `Hard`, `Good`, `Easy`. `Again` resets the
streak and schedules the card for the next day. The others grow the
interval, with the ease factor shifting slightly each time based on
how the review went.

`Deck` (in `store.go`) is a thin JSON-backed collection of notes if
you want persistence without writing your own storage layer.

## CLI

```sh
go build -o spacer ./cmd/spacer

./spacer add capital-france "capital of France?" "Paris"
./spacer add capital-japan  "capital of Japan?"  "Tokyo"

./spacer due
# capital-france   capital of France?
# capital-japan    capital of Japan?

./spacer review capital-france good
# capital-france: next review in 1 day(s), due 2026-08-26

./spacer list
# capital-france   capital of France?   due 2026-08-26
# capital-japan    capital of Japan?    due 2026-08-25
```

Deck data lives at `~/.spacer/deck.json` by default; set `SPACER_DECK`
to point somewhere else.

## Status

Early. The scheduling core and CLI work end to end; see the roadmap
in the repo for what's missing (bulk import, a `stats` command,
configurable ease/interval constants).

## License

MIT, see LICENSE.
