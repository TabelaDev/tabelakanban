package main

import (
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

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

	send := func(key string) {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}
	sendEnter := func() {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}
	sendConfirm := func() {
		nm, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
		switch v := nm.(type) {
		case appModel:
			m = v
		case *appModel:
			m = *v
		}
	}

	// create card in backlog (col 0)
	send("n")
	m.input.SetValue("tarefa-nova")
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

	// rename it
	send("r")
	m.input.SetValue("tarefa-renomeada")
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
