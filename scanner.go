package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Card is one .md file inside a column directory. Its Title is the filename
// without the .md suffix; Body is the file contents minus the YAML front
// matter; Due is the front-matter `due:` date (zero when unset).
type Card struct {
	Title string
	Path  string
	Body  string
	Due   time.Time
}

// Column is a subdirectory of a board. Its name is the column header; each
// card file inside is a column item.
type Column struct {
	Name  string
	Path  string
	Cards []Card
}

// Board is a subdirectory of a board root: columns are its subdirectories.
type Board struct {
	Name    string
	Path    string
	Columns []Column
}

// scanBoards walks every root, treating each child directory as a board.
// Sort order is by name so navigation is predictable.
func scanBoards(roots []string) ([]Board, []string) {
	var boards []Board
	var warnings []string
	for _, root := range roots {
		dirs, err := os.ReadDir(root)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("raiz %s: %v", root, err))
			continue
		}
		for _, d := range dirs {
			if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
				continue
			}
			board, err := scanBoard(filepath.Join(root, d.Name()))
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("board %s: %v", d.Name(), err))
				continue
			}
			boards = append(boards, board)
		}
	}
	sort.Slice(boards, func(i, j int) bool { return boards[i].Name < boards[j].Name })
	return boards, warnings
}

func scanBoard(dir string) (Board, error) {
	board := Board{Name: filepath.Base(dir), Path: dir}
	dirs, err := os.ReadDir(dir)
	if err != nil {
		return board, err
	}
	order := columnOrderFromFile(dir)
	colByName := make(map[string]Column)
	for _, d := range dirs {
		if !d.IsDir() || strings.HasPrefix(d.Name(), ".") {
			continue
		}
		column := Column{Name: d.Name(), Path: filepath.Join(dir, d.Name())}
		files, err := os.ReadDir(column.Path)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".md") {
				continue
			}
			cardPath := filepath.Join(column.Path, f.Name())
			raw, _ := os.ReadFile(cardPath)
			due, body := parseFrontMatter(string(raw))
			column.Cards = append(column.Cards, Card{
				Title: strings.TrimSuffix(f.Name(), ".md"),
				Path:  cardPath,
				Body:  body,
				Due:   due,
			})
		}
		sort.Slice(column.Cards, func(i, j int) bool { return column.Cards[i].Title < column.Cards[j].Title })
		colByName[d.Name()] = column
	}

	// Columns listed in .order come first (in that order); anything not in
	// the file follows alphabetically — which is also the full fallback when
	// there's no .order file, keeping old boards unchanged.
	seen := make(map[string]bool, len(colByName))
	var columns []Column
	for _, name := range order {
		if col, ok := colByName[name]; ok {
			columns = append(columns, col)
			seen[name] = true
		}
	}
	var rest []string
	for name := range colByName {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	for _, name := range rest {
		columns = append(columns, colByName[name])
	}
	board.Columns = columns
	return board, nil
}

// columnOrderFromFile reads the board's .order file (one column name per
// line; blank lines and # comments ignored). Missing file means nil, i.e.
// alphabetical order.
func columnOrderFromFile(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, ".order"))
	if err != nil {
		return nil
	}
	var order []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			order = append(order, line)
		}
	}
	return order
}

// parseFrontMatter extracts the `due:` date from a card's leading YAML block
// (---\ndue: YYYY-MM-DD\n---) and returns the body with that block stripped.
// A missing/malformed block yields a zero due and the body unchanged.
func parseFrontMatter(raw string) (time.Time, string) {
	if !strings.HasPrefix(raw, "---\n") {
		return time.Time{}, raw
	}
	end := strings.Index(raw, "\n---")
	if end < 0 {
		return time.Time{}, raw
	}
	block := raw[4:end]
	body := raw[end+4:]
	body = strings.TrimPrefix(body, "\n")

	var due time.Time
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "due:") {
			continue
		}
		if d, err := time.Parse("2006-01-02", strings.TrimSpace(strings.TrimPrefix(line, "due:"))); err == nil {
			due = d
		}
	}
	return due, body
}

// renderFrontMatter returns a card's YAML block for its due date, or "" when
// there's no due date.
func renderFrontMatter(due time.Time) string {
	if due.IsZero() {
		return ""
	}
	return "---\ndue: " + due.Format("2006-01-02") + "\n---\n"
}
