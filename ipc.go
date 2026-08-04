package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/ianptkcs/tabelatuiui"
)

// cardJSON is the wire format for the ipc subcommand.
type cardJSON struct {
	Title string `json:"title"`
	Path  string `json:"path"`
	Body  string `json:"body,omitempty"`
}

type columnJSON struct {
	Name  string     `json:"name"`
	Path  string     `json:"path"`
	Cards []cardJSON `json:"cards"`
}

type boardJSON struct {
	Name    string       `json:"name"`
	Path    string       `json:"path"`
	Columns []columnJSON `json:"columns"`
}

func boardsToJSON(boards []Board) []boardJSON {
	out := make([]boardJSON, 0, len(boards))
	for _, b := range boards {
		bj := boardJSON{Name: b.Name, Path: b.Path}
		for _, c := range b.Columns {
			cj := columnJSON{Name: c.Name, Path: c.Path}
			for _, card := range c.Cards {
				cj.Cards = append(cj.Cards, cardJSON{Title: card.Title, Path: card.Path, Body: card.Body})
			}
			bj.Columns = append(bj.Columns, cj)
		}
		out = append(out, bj)
	}
	return out
}

// runIPC implements `tabelakanban ipc <método> [key=value...] --json`, the
// same scriptable-data-source convention as dcal/djobs/tabelaradar.
func runIPC(args []string) int {
	parsed, err := tuiui.ParseIPCArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "uso: tabelakanban ipc <método> [key=value...] --json")
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	roots, cfgWarning := loadRootsConfig()
	boards, warnings := scanBoards(roots)
	if cfgWarning != "" {
		warnings = append([]string{cfgWarning}, warnings...)
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "aviso:", w)
	}

	switch parsed.Method {
	case "boards.list":
		out := boardsToJSON(boards)
		if name, ok := parsed.Filters["name"]; ok {
			filtered := out[:0]
			for _, b := range out {
				if b.Name == name {
					filtered = append(filtered, b)
				}
			}
			out = filtered
		}
		return tuiui.WriteJSON(out)
	case "boards.next":
		return ipcBoardsNext(boards)
	default:
		fmt.Fprintf(os.Stderr, "método desconhecido: %q\n", parsed.Method)
		return 1
	}
}

// ipcBoardsNext returns the single card tabelakanban itself would put first:
// the top card of the first column that isn't a "done"-ish column (by name),
// falling back to the first card of the first column if every column looks
// done.
func ipcBoardsNext(boards []Board) int {
	if len(boards) == 0 {
		return tuiui.WriteJSON(nil)
	}
	b := boards[0]
	for _, c := range b.Columns {
		if isDoneColumn(c.Name) || len(c.Cards) == 0 {
			continue
		}
		return tuiui.WriteJSON(boardJSON{Name: b.Name, Path: b.Path, Columns: []columnJSON{{
			Name: c.Name, Path: c.Path, Cards: []cardJSON{{Title: c.Cards[0].Title, Path: c.Cards[0].Path, Body: c.Cards[0].Body}},
		}}})
	}
	for _, c := range b.Columns {
		if len(c.Cards) == 0 {
			continue
		}
		return tuiui.WriteJSON(boardJSON{Name: b.Name, Path: b.Path, Columns: []columnJSON{{
			Name: c.Name, Path: c.Path, Cards: []cardJSON{{Title: c.Cards[0].Title, Path: c.Cards[0].Path, Body: c.Cards[0].Body}},
		}}})
	}
	return tuiui.WriteJSON(nil)
}

func isDoneColumn(name string) bool {
	for _, marker := range []string{"done", "feito", "conclu"} {
		if strings.Contains(strings.ToLower(name), marker) {
			return true
		}
	}
	return false
}
