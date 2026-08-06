package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// driveKey feeds a message through the model and runs the form's returned
// commands (navigation), skipping the cosmetic cursor-blink cmd that sleeps.
func driveKey(m *appModel, msg tea.Msg) {
	for i := 0; i < 20; i++ {
		nm, cmd := m.Update(msg)
		switch v := nm.(type) {
		case appModel:
			*m = v
		case *appModel:
			*m = *v
		}
		if cmd == nil || m.mode != modeInput {
			return
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- cmd() }()
		select {
		case msg = <-done:
			if msg == nil {
				return
			}
		case <-time.After(50 * time.Millisecond):
			return
		}
	}
}

// TestKeyFlow drives the model through create/move/rename/delete with real
// KeyMsg messages on a throwaway board, verifying the TUI glue (not just the
// ops functions) wires keys to filesystem changes.
func TestKeyFlow(t *testing.T) {
	root := t.TempDir()
	os.Setenv("TABELAKANBAN_ROOT", root)
	os.Setenv("TABELAKANBAN_CONFIG", filepath.Join(root, "no-config"))

	// board "dev" with columns "backlog", "fazendo"
	for _, col := range []string{"backlog", "fazendo"} {
		if err := os.MkdirAll(filepath.Join(root, "dev", col), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	m := newModel()
	m.width, m.height = 100, 26
	m.reclamp()
	if m.currentBoardName() != "dev" {
		t.Fatalf("board = %q, want dev", m.currentBoardName())
	}

	// apply feeds a message through the model AND runs any returned commands
	// (the huh form returns cmds for field navigation, which bubbletea would
	// normally process in its event loop). Navigation cmds return immediately;
	// the cursor-blink cmd sleeps ~500ms, so it's run in a goroutine with a
	// short timeout and skipped — it's purely cosmetic.
	apply := func(msg tea.Msg) {
		for i := 0; i < 20; i++ {
			nm, cmd := m.Update(msg)
			switch v := nm.(type) {
			case appModel:
				m = v
			case *appModel:
				m = *v
			}
			if cmd == nil || m.mode != modeInput {
				return
			}
			done := make(chan tea.Msg, 1)
			go func() { done <- cmd() }()
			select {
			case msg = <-done:
				if msg == nil {
					return
				}
			case <-time.After(50 * time.Millisecond):
				return
			}
		}
	}
	send := func(key string) {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	}
	typeRunes := func(s string) {
		for _, r := range s {
			send(string(r))
		}
	}
	sendEnter := func() {
		apply(tea.KeyMsg{Type: tea.KeyEnter})
	}
	sendBackspace := func(n int) {
		for i := 0; i < n; i++ {
			apply(tea.KeyMsg{Type: tea.KeyBackspace})
		}
	}
	sendConfirm := func() {
		apply(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	}

	// create card in backlog (col 0) via the huh form: type the title, enter
	// past the (empty) due field, enter to submit
	send("n")
	typeRunes("tarefa-nova")
	sendEnter()
	sendEnter()
	if m.currentColumnName() != "backlog" {
		t.Fatalf("after create, col = %q", m.currentColumnName())
	}
	if _, err := os.Stat(filepath.Join(root, "dev", "backlog", "tarefa-nova.md")); err != nil {
		t.Fatalf("card file missing: %v", err)
	}

	// move it to "fazendo" (col 1) with L
	send("L")
	if _, err := os.Stat(filepath.Join(root, "dev", "fazendo", "tarefa-nova.md")); err != nil {
		t.Fatalf("moved card missing: %v", err)
	}

	// rename it (pre-filled "tarefa-nova"; clear then type the new name)
	send("r")
	sendBackspace(11)
	typeRunes("tarefa-renomeada")
	sendEnter()
	if _, err := os.Stat(filepath.Join(root, "dev", "fazendo", "tarefa-renomeada.md")); err != nil {
		t.Fatalf("renamed card missing: %v", err)
	}

	// delete it
	send("d")
	sendConfirm()
	if _, err := os.Stat(filepath.Join(root, "dev", "fazendo", "tarefa-renomeada.md")); !os.IsNotExist(err) {
		t.Fatalf("deleted card still exists: %v", err)
	}
}

func TestCardPreviewSkipsH1(t *testing.T) {
	body := "# configurar CI no repo\n\n- github actions\n- badges"
	if got := cardPreview(body); got != "github actions" {
		t.Fatalf("cardPreview = %q, want 'github actions'", got)
	}
	plain := "linha sem markdown"
	if got := cardPreview(plain); got != plain {
		t.Fatalf("cardPreview(plain) = %q, want %q", got, plain)
	}
	if got := cardPreview(""); got != "" {
		t.Fatalf("cardPreview(empty) = %q, want empty", got)
	}
}

func TestCtrlNavAndPreview(t *testing.T) {
	root := t.TempDir()
	os.Setenv("TABELAKANBAN_ROOT", root)
	os.Setenv("TABELAKANBAN_CONFIG", filepath.Join(root, "no-config"))
	for _, col := range []string{"a", "b"} {
		if err := os.MkdirAll(filepath.Join(root, "dev", col), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel()
	m.width, m.height = 120, 26
	m.reclamp()

	send := func(key string) {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}
	sendCtrl := func(k tea.KeyType) {
		nm, _ := m.Update(tea.KeyMsg{Type: k})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}

	if m.colIdx != 0 {
		t.Fatalf("initial colIdx = %d, want 0", m.colIdx)
	}

	// plain hjkl navigation
	send("l")
	if m.colIdx != 1 {
		t.Fatalf("after l colIdx = %d, want 1", m.colIdx)
	}
	send("h")
	if m.colIdx != 0 {
		t.Fatalf("after h colIdx = %d, want 0", m.colIdx)
	}

	// ctrl+h/l move (reorder) the focused column instead of navigating
	sendCtrl(tea.KeyCtrlL)
	if m.colIdx != 1 {
		t.Fatalf("after ctrl+l colIdx = %d, want 1 (column moved right)", m.colIdx)
	}
	board, err := scanBoard(filepath.Join(root, "dev"))
	if err != nil {
		t.Fatal(err)
	}
	if board.Columns[0].Name != "b" || board.Columns[1].Name != "a" {
		t.Fatalf("after ctrl+l column order = %s, %s; want b, a",
			board.Columns[0].Name, board.Columns[1].Name)
	}
	sendCtrl(tea.KeyCtrlH)
	if m.colIdx != 0 {
		t.Fatalf("after ctrl+h colIdx = %d, want 0 (column moved left)", m.colIdx)
	}

	if m.preview {
		t.Fatal("preview should start closed")
	}
	send("o")
	if !m.preview {
		t.Fatal("preview should open with o")
	}
	send("o")
	if m.preview {
		t.Fatal("preview should close with o again")
	}

	if m.mode != modeBoard {
		t.Fatalf("mode = %d, want modeBoard", m.mode)
	}
	send("g")
	if m.mode != modeLogs {
		t.Fatalf("mode after g = %d, want modeLogs", m.mode)
	}
	send("q")
	if m.mode != modeBoard {
		t.Fatalf("mode after closing logs = %d, want modeBoard", m.mode)
	}
}

func TestSidebarNavigation(t *testing.T) {
	root := t.TempDir()
	os.Setenv("TABELAKANBAN_ROOT", root)
	os.Setenv("TABELAKANBAN_CONFIG", filepath.Join(root, "no-config"))
	for _, board := range []string{"a", "b", "c"} {
		if err := os.MkdirAll(filepath.Join(root, board, "col"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	m := newModel()
	m.width, m.height = 120, 26
	m.reclamp()

	send := func(key string) {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}
	sendCtrl := func(k tea.KeyType) {
		nm, _ := m.Update(tea.KeyMsg{Type: k})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}

	if m.boardIdx != 0 || m.sidebarFocused {
		t.Fatalf("initial state: boardIdx=%d sidebarFocused=%v", m.boardIdx, m.sidebarFocused)
	}
	// h on the first column enters the sidebar (h/l are plain column nav now;
	// ctrl+h/l reorder columns)
	send("h")
	if !m.sidebarFocused {
		t.Fatal("h on first column should focus sidebar")
	}
	// j/k move through boards
	send("j")
	send("j")
	if m.boardIdx != 2 {
		t.Fatalf("after jj boardIdx = %d, want 2", m.boardIdx)
	}
	send("k")
	if m.boardIdx != 1 {
		t.Fatalf("after k boardIdx = %d, want 1", m.boardIdx)
	}
	// l leaves the sidebar back on the columns
	send("l")
	if m.sidebarFocused {
		t.Fatal("l should leave the sidebar")
	}
	// ctrl+e collapses and expands the sidebar
	sendCtrl(tea.KeyCtrlE)
	if m.sidebar {
		t.Fatal("ctrl+e should collapse the sidebar")
	}
	sendCtrl(tea.KeyCtrlE)
	if !m.sidebar {
		t.Fatal("ctrl+e should expand the sidebar")
	}
}

func TestWidthHelpers(t *testing.T) {
	m := appModel{width: 180, height: 40, preview: false}
	area := m.boardAreaWidth()
	if area != 180 {
		t.Fatalf("boardAreaWidth = %d, want 180", area)
	}
	if got := m.visibleCols(area); got < 3 {
		t.Fatalf("visibleCols on 180 wide = %d, want >= 3", got)
	}
	m.preview = true
	if m.columnsAvailWidth() >= area {
		t.Fatalf("preview should shrink columns: avail %d >= area %d", m.columnsAvailWidth(), area)
	}
}

// TestTinyTerminalNoPanic ensures View() survives absurdly small terminal
// sizes (negative body heights must not crash the render).
func TestTinyTerminalNoPanic(t *testing.T) {
	root := t.TempDir()
	os.Setenv("TABELAKANBAN_ROOT", root)
	os.Setenv("TABELAKANBAN_CONFIG", filepath.Join(root, "no-config"))
	if err := os.MkdirAll(filepath.Join(root, "dev", "col"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModel()
	for _, size := range [][2]int{{30, 4}, {40, 5}, {20, 3}, {80, 8}} {
		m.width, m.height = size[0], size[1]
		m.reclamp()
		if got := m.View(); got == "" {
			t.Fatalf("empty view at %dx%d", size[0], size[1])
		}
		m.preview = true
		m.View()
		m.preview = false
	}

	// empty states: a board with no columns, and no boards at all
	if err := os.MkdirAll(filepath.Join(root, "vazio"), 0o755); err != nil {
		t.Fatal(err)
	}
	m.rescan(false)
	for _, size := range [][2]int{{20, 3}, {60, 5}} {
		m.width, m.height = size[0], size[1]
		m.boardIdx = indexOfBoard(m.boards, "vazio")
		m.reclamp()
		m.View()
	}
}

func TestCardDueFlow(t *testing.T) {
	root := t.TempDir()
	os.Setenv("TABELAKANBAN_ROOT", root)
	os.Setenv("TABELAKANBAN_CONFIG", filepath.Join(root, "no-config"))
	if err := os.MkdirAll(filepath.Join(root, "dev", "backlog"), 0o755); err != nil {
		t.Fatal(err)
	}
	m := newModel()
	m.width, m.height = 100, 26
	m.reclamp()

	// create a card with a due date via the form: title, enter → due, type
	// the date, enter → submit
	driveKey(&m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	for _, r := range "com-prazo" {
		driveKey(&m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	driveKey(&m, tea.KeyMsg{Type: tea.KeyEnter})
	for _, r := range "20/12" {
		driveKey(&m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	driveKey(&m, tea.KeyMsg{Type: tea.KeyEnter})

	raw, err := os.ReadFile(filepath.Join(root, "dev", "backlog", "com-prazo.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "due: ") {
		t.Fatalf("due front matter missing: %q", raw)
	}

	m.rescan(false)
	card := m.currentCard()
	if card == nil || card.Due.IsZero() {
		t.Fatalf("due not parsed from front matter: %+v", card)
	}

	// edit the due with t: clear it (backspace the pre-filled value, submit)
	driveKey(&m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	for i := 0; i < 5; i++ {
		driveKey(&m, tea.KeyMsg{Type: tea.KeyBackspace})
	}
	driveKey(&m, tea.KeyMsg{Type: tea.KeyEnter})

	m.rescan(false)
	if c := m.currentCard(); c == nil || !c.Due.IsZero() {
		t.Fatalf("due not cleared: %+v", c)
	}
	raw, _ = os.ReadFile(filepath.Join(root, "dev", "backlog", "com-prazo.md"))
	if strings.Contains(string(raw), "due:") {
		t.Fatalf("front matter not removed after clearing due: %q", raw)
	}
}
