package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestRenameColumn(t *testing.T) {
	board := setupTestBoard(t)
	backlog := board.Columns[0]
	if _, err := createCard(backlog, "tarefa"); err != nil {
		t.Fatal(err)
	}
	renamed, err := renameColumn(backlog, "planejamento")
	if err != nil {
		t.Fatal(err)
	}
	if renamed.Name != "planejamento" {
		t.Fatalf("renamed name = %q, want planejamento", renamed.Name)
	}
	if _, err := os.Stat(filepath.Join(renamed.Path, "tarefa.md")); err != nil {
		t.Fatalf("card missing after column rename: %v", err)
	}
	if _, err := os.Stat(backlog.Path); !os.IsNotExist(err) {
		t.Fatalf("old column dir still exists")
	}
}

func TestRenameColumnCollision(t *testing.T) {
	board := setupTestBoard(t)
	if _, err := renameColumn(board.Columns[0], board.Columns[1].Name); err == nil {
		t.Fatalf("expected error renaming onto existing column")
	}
}

func TestDeleteColumn(t *testing.T) {
	board := setupTestBoard(t)
	backlog := board.Columns[0]
	if _, err := createCard(backlog, "tarefa"); err != nil {
		t.Fatal(err)
	}
	if err := deleteColumn(backlog); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(backlog.Path); !os.IsNotExist(err) {
		t.Fatalf("column dir still exists after delete")
	}
}

func TestMoveColumnReordersViaOrderFile(t *testing.T) {
	board := setupTestBoard(t) // columns: backlog, fazendo (alphabetical)
	if err := moveColumn(board, 1, 0); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(board.Path, ".order"))
	if err != nil {
		t.Fatalf("order file missing: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != "fazendo\nbacklog" {
		t.Fatalf("order file = %q, want fazendo/backlog", got)
	}
	// re-scan and confirm the order sticks
	scanned, err := scanBoard(board.Path)
	if err != nil {
		t.Fatal(err)
	}
	if scanned.Columns[0].Name != "fazendo" || scanned.Columns[1].Name != "backlog" {
		t.Fatalf("rescan order = %s, %s; want fazendo, backlog",
			scanned.Columns[0].Name, scanned.Columns[1].Name)
	}
}

func TestScannerUsesOrderFile(t *testing.T) {
	dir := t.TempDir()
	boardDir := filepath.Join(dir, "dev")
	for _, col := range []string{"zeta", "alfa"} {
		if err := os.MkdirAll(filepath.Join(boardDir, col), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(boardDir, ".order"), []byte("zeta\nalfa\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	board, err := scanBoard(boardDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(board.Columns) != 2 || board.Columns[0].Name != "zeta" || board.Columns[1].Name != "alfa" {
		t.Fatalf("order not respected: %+v", board.Columns)
	}
}

func TestCreateColumnKeepsOrderFile(t *testing.T) {
	board := setupTestBoard(t)
	if err := moveColumn(board, 0, 1); err != nil {
		t.Fatal(err)
	}
	if err := createColumn(board, "novo"); err != nil {
		t.Fatal(err)
	}
	scanned, err := scanBoard(board.Path)
	if err != nil {
		t.Fatal(err)
	}
	if scanned.Columns[len(scanned.Columns)-1].Name != "novo" {
		t.Fatalf("new column not appended to order: %+v", scanned.Columns)
	}
}

func TestRenameColumnUpdatesOrderFile(t *testing.T) {
	board := setupTestBoard(t)
	if err := moveColumn(board, 0, 1); err != nil {
		t.Fatal(err)
	}
	// board.Columns is stale after moveColumn (it only writes .order); the
	// column first in the new order is fazendo (old Columns[1]).
	if _, err := renameColumn(board.Columns[1], "renomeada"); err != nil {
		t.Fatal(err)
	}
	scanned, err := scanBoard(board.Path)
	if err != nil {
		t.Fatal(err)
	}
	if scanned.Columns[0].Name != "renomeada" {
		t.Fatalf("renamed column lost its order position: %+v", scanned.Columns)
	}
}

func TestParseFrontMatter(t *testing.T) {
	due, body := parseFrontMatter("---\ndue: 2026-08-20\n---\n# title\n")
	if got := due.Format("2006-01-02"); got != "2026-08-20" {
		t.Fatalf("due = %s, want 2026-08-20", got)
	}
	if body != "# title\n" {
		t.Fatalf("body = %q, want %q", body, "# title\n")
	}

	due, body = parseFrontMatter("# sem front matter\n")
	if !due.IsZero() || body != "# sem front matter\n" {
		t.Fatalf("plain body mangled: due=%v body=%q", due, body)
	}

	due, body = parseFrontMatter("---\ndue: not-a-date\n---\n# x\n")
	if !due.IsZero() {
		t.Fatalf("malformed due should be ignored")
	}
	if body != "# x\n" {
		t.Fatalf("body = %q, want %q", body, "# x\n")
	}
}

func TestSetCardDue(t *testing.T) {
	board := setupTestBoard(t)
	card, err := createCard(board.Columns[0], "tarefa")
	if err != nil {
		t.Fatal(err)
	}
	due := time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local)
	updated, err := setCardDue(card, due)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(updated.Path)
	if !strings.Contains(string(raw), "due: 2026-08-20") {
		t.Fatalf("due not written: %q", raw)
	}

	scanned, err := scanBoard(board.Path)
	if err != nil {
		t.Fatal(err)
	}
	sc := scanned.Columns[0].Cards[0]
	if got := sc.Due.Format("2006-01-02"); got != "2026-08-20" {
		t.Fatalf("rescanned due = %s", got)
	}

	cleared, err := setCardDue(sc, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = os.ReadFile(cleared.Path)
	if strings.Contains(string(raw), "due:") {
		t.Fatalf("due not cleared: %q", raw)
	}
}

func TestUpdateCardBody(t *testing.T) {
	board := setupTestBoard(t)
	card, err := createCard(board.Columns[0], "tarefa")
	if err != nil {
		t.Fatal(err)
	}
	due := time.Date(2026, 8, 20, 0, 0, 0, 0, time.Local)
	if card, err = setCardDue(card, due); err != nil {
		t.Fatal(err)
	}

	newBody := "# tarefa\n\nProgresso novo do digest.\n"
	updated, err := updateCardBody(card, newBody)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(updated.Path)
	if !strings.Contains(string(raw), "due: 2026-08-20") {
		t.Fatalf("due lost on body update: %q", raw)
	}
	if !strings.Contains(string(raw), "Progresso novo do digest.") {
		t.Fatalf("new body not written: %q", raw)
	}

	scanned, err := scanBoard(board.Path)
	if err != nil {
		t.Fatal(err)
	}
	sc := scanned.Columns[0].Cards[0]
	if sc.Body != newBody {
		t.Fatalf("rescanned body = %q, want %q", sc.Body, newBody)
	}
	if got := sc.Due.Format("2006-01-02"); got != "2026-08-20" {
		t.Fatalf("rescanned due = %s", got)
	}
}

func TestUpdateCardBodyKeepsFrontMatterWhenNone(t *testing.T) {
	board := setupTestBoard(t)
	card, err := createCard(board.Columns[0], "sem-due")
	if err != nil {
		t.Fatal(err)
	}
	updated, err := updateCardBody(card, "# sem-due\n\nsó corpo\n")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(updated.Path)
	if strings.Contains(string(raw), "---") {
		t.Fatalf("front matter invented for a card without due: %q", raw)
	}
	if !strings.Contains(string(raw), "só corpo") {
		t.Fatalf("body not written: %q", raw)
	}
}
