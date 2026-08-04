package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Card is one .md file inside a column directory. Its Title is the filename
// without the .md suffix; Body is the raw file contents.
type Card struct {
	Title string
	Path  string
	Body  string
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
			body, _ := os.ReadFile(cardPath)
			column.Cards = append(column.Cards, Card{
				Title: strings.TrimSuffix(f.Name(), ".md"),
				Path:  cardPath,
				Body:  string(body),
			})
		}
		sort.Slice(column.Cards, func(i, j int) bool { return column.Cards[i].Title < column.Cards[j].Title })
		board.Columns = append(board.Columns, column)
	}
	sort.Slice(board.Columns, func(i, j int) bool { return board.Columns[i].Name < board.Columns[j].Name })
	return board, nil
}
