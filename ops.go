package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// sanitizeTitle turns a user-typed title into a safe filename stem: trimmed,
// no slashes (which would escape the column directory), no empty result.
func sanitizeTitle(title string) string {
	title = strings.TrimSpace(strings.ReplaceAll(title, "/", "-"))
	return title
}

// createCard writes a new .md file into column's directory. If a card with
// that title already exists, it gets a " (2)", " (3)"... suffix.
func createCard(column Column, title string) (Card, error) {
	title = sanitizeTitle(title)
	if title == "" {
		return Card{}, fmt.Errorf("título vazio")
	}
	path, n := uniquePath(column.Path, title, ".md")
	if n > 1 {
		title = fmt.Sprintf("%s (%d)", title, n)
	}
	body := "# " + title + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return Card{}, err
	}
	return Card{Title: title, Path: path, Body: body}, nil
}

// renameCard moves the card file to a new title, preserving the body.
func renameCard(card Card, newTitle string) (Card, error) {
	newTitle = sanitizeTitle(newTitle)
	if newTitle == "" {
		return Card{}, fmt.Errorf("título vazio")
	}
	dir := filepath.Dir(card.Path)
	path, n := uniquePath(dir, newTitle, ".md")
	if n > 1 {
		newTitle = fmt.Sprintf("%s (%d)", newTitle, n)
	}
	if err := os.Rename(card.Path, path); err != nil {
		return Card{}, err
	}
	return Card{Title: newTitle, Path: path, Body: card.Body}, nil
}

// moveCard relocates a card file into another column's directory.
func moveCard(card Card, toColumn Column) (Card, error) {
	dir := toColumn.Path
	path, n := uniquePath(dir, strings.TrimSuffix(filepath.Base(card.Path), ".md"), ".md")
	title := strings.TrimSuffix(filepath.Base(path), ".md")
	if n > 1 && title == strings.TrimSuffix(filepath.Base(card.Path), ".md") {
		title = fmt.Sprintf("%s (%d)", title, n)
	}
	if err := os.Rename(card.Path, path); err != nil {
		return Card{}, err
	}
	return Card{Title: title, Path: path, Body: card.Body}, nil
}

func deleteCard(card Card) error {
	return os.Remove(card.Path)
}

// createColumn makes a new column directory inside the board.
func createColumn(board Board, name string) error {
	name = sanitizeTitle(name)
	if name == "" {
		return fmt.Errorf("nome vazio")
	}
	return os.Mkdir(filepath.Join(board.Path, name), 0o755)
}

// createBoard makes a new board directory under the first configured root.
func createBoard(roots []string, name string) (string, error) {
	name = sanitizeTitle(name)
	if name == "" {
		return "", fmt.Errorf("nome vazio")
	}
	if len(roots) == 0 {
		return "", fmt.Errorf("nenhuma raiz configurada")
	}
	dir := filepath.Join(roots[0], name)
	return dir, os.Mkdir(dir, 0o755)
}

// uniquePath returns a path in dir for stem+suffix that doesn't collide with
// an existing file, appending " (2)", " (3)"... as needed, plus the number
// that was used (1 = no collision).
func uniquePath(dir, stem, suffix string) (string, int) {
	for n := 1; ; n++ {
		name := stem
		if n > 1 {
			name = fmt.Sprintf("%s (%d)", stem, n)
		}
		path := filepath.Join(dir, name+suffix)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return path, n
		}
	}
}
