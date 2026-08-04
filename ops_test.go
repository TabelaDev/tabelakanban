package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestBoard(t *testing.T) Board {
	t.Helper()
	dir := t.TempDir()
	boardDir := filepath.Join(dir, "dev")
	for _, col := range []string{"backlog", "fazendo"} {
		if err := os.MkdirAll(filepath.Join(boardDir, col), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	board, err := scanBoard(boardDir)
	if err != nil {
		t.Fatal(err)
	}
	return board
}

func TestCreateAndMoveCard(t *testing.T) {
	board := setupTestBoard(t)
	backlog := board.Columns[0]
	card, err := createCard(backlog, "nova-tarefa")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(card.Path); err != nil {
		t.Fatalf("card file missing: %v", err)
	}

	fazendo := board.Columns[1]
	moved, err := moveCard(card, fazendo)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(moved.Path); err != nil {
		t.Fatalf("moved card missing: %v", err)
	}
	if _, err := os.Stat(card.Path); !os.IsNotExist(err) {
		t.Fatalf("original card still exists: %v", err)
	}
	if !strings.Contains(moved.Path, filepath.Join("dev", "fazendo")) {
		t.Fatalf("moved to wrong column: %s", moved.Path)
	}
}

func TestCreateCardCollision(t *testing.T) {
	board := setupTestBoard(t)
	backlog := board.Columns[0]
	if _, err := createCard(backlog, "mesmo-nome"); err != nil {
		t.Fatal(err)
	}
	second, err := createCard(backlog, "mesmo-nome")
	if err != nil {
		t.Fatal(err)
	}
	if second.Title != "mesmo-nome (2)" {
		t.Fatalf("collision title = %q, want 'mesmo-nome (2)'", second.Title)
	}
}

func TestSanitizeTitle(t *testing.T) {
	if got := sanitizeTitle("  a/b  "); got != "a-b" {
		t.Fatalf("sanitizeTitle = %q, want a-b", got)
	}
	if got := sanitizeTitle("   "); got != "" {
		t.Fatalf("sanitizeTitle(blank) = %q, want empty", got)
	}
}
