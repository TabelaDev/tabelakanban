package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
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

// setCardDue writes (or clears, when due is zero) the card's due date into
// its YAML front matter, preserving the body.
func setCardDue(card Card, due time.Time) (Card, error) {
	content := renderFrontMatter(due) + card.Body
	if err := os.WriteFile(card.Path, []byte(content), 0o644); err != nil {
		return Card{}, err
	}
	card.Due = due
	return card, nil
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
	dir := filepath.Join(board.Path, name)
	if err := os.Mkdir(dir, 0o755); err != nil {
		return err
	}
	return appendColumnOrder(board, name)
}

// appendColumnOrder keeps an existing .order file in sync when a column is
// created. Boards without a .order file stay alphabetical.
func appendColumnOrder(board Board, name string) error {
	orderPath := filepath.Join(board.Path, ".order")
	if _, err := os.Stat(orderPath); os.IsNotExist(err) {
		return nil
	}
	f, err := os.OpenFile(orderPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(name + "\n")
	return err
}

// renameColumn renames the column directory, keeping its cards.
func renameColumn(column Column, newName string) (Column, error) {
	newName = sanitizeTitle(newName)
	if newName == "" {
		return Column{}, fmt.Errorf("nome vazio")
	}
	newPath := filepath.Join(filepath.Dir(column.Path), newName)
	if newPath == column.Path {
		return column, nil
	}
	if _, err := os.Stat(newPath); err == nil {
		return Column{}, fmt.Errorf("coluna %q já existe", newName)
	}
	if err := os.Rename(column.Path, newPath); err != nil {
		return Column{}, err
	}
	if err := replaceColumnOrder(filepath.Dir(column.Path), column.Name, newName); err != nil {
		return Column{}, err
	}
	column.Name = newName
	column.Path = newPath
	return column, nil
}

// replaceColumnOrder swaps oldName for newName in the board's .order file
// (no-op when there's no .order file).
func replaceColumnOrder(boardDir, oldName, newName string) error {
	orderPath := filepath.Join(boardDir, ".order")
	data, err := os.ReadFile(orderPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) == oldName {
			lines[i] = newName
		}
	}
	return os.WriteFile(orderPath, []byte(strings.Join(lines, "\n")), 0o644)
}

// moveColumn reorders a board's columns so the column at `from` lands at
// `to`, by materializing the order in the board's .order file (column
// directories are name-ordered by default, so an explicit order file is what
// makes "moving a column" meaningful without renaming it).
func moveColumn(board Board, from, to int) error {
	n := len(board.Columns)
	if n == 0 || from < 0 || from >= n || to < 0 || to >= n || from == to {
		return nil
	}
	order := make([]string, n)
	for i, c := range board.Columns {
		order[i] = c.Name
	}
	name := order[from]
	order = append(order[:from], order[from+1:]...)
	order = append(order[:to], append([]string{name}, order[to:]...)...)
	orderPath := filepath.Join(board.Path, ".order")
	return os.WriteFile(orderPath, []byte(strings.Join(order, "\n")+"\n"), 0o644)
}

// deleteColumn removes the column directory and everything in it.
func deleteColumn(column Column) error {
	return os.RemoveAll(column.Path)
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
