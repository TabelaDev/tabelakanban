package main

import (
	"github.com/charmbracelet/bubbles/key"
)

// tabelakanban's keybindings, declared once and shared by the key dispatch in
// updateBoard (key.Matches), the footer hints (tuiui.Footer) and the help
// modal (tuiui.HelpModal) — the hints can never drift out of sync with what
// Update actually matches.
var (
	keyQuit       = key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "sair"))
	keyHelp       = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "keybindings"))
	keyRefresh    = key.NewBinding(key.WithKeys("ctrl+r"), key.WithHelp("ctrl+r", "recarregar"))
	keySidebar    = key.NewBinding(key.WithKeys("ctrl+e"), key.WithHelp("ctrl+e", "sidebar"))
	keyLogs       = key.NewBinding(key.WithKeys("g"), key.WithHelp("g", "logs"))
	keyNewCard    = key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "novo card"))
	keyNewColumn  = key.NewBinding(key.WithKeys("N"), key.WithHelp("N", "nova coluna"))
	keyNewBoard   = key.NewBinding(key.WithKeys("B"), key.WithHelp("B", "novo board"))
	keyRenameCard = key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "renomear card"))
	keyRenameCol  = key.NewBinding(key.WithKeys("R"), key.WithHelp("R", "renomear coluna"))
	keyDue        = key.NewBinding(key.WithKeys("t"), key.WithHelp("t", "due date"))
	keyDelCard    = key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "del card"))
	keyDelCol     = key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "del coluna"))
	keyPreview    = key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "preview"))
	keyOpen       = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "editar"))
	keyMoveColL   = key.NewBinding(key.WithKeys("h", "left"), key.WithHelp("h", "coluna esq"))
	keyMoveColR   = key.NewBinding(key.WithKeys("l", "right"), key.WithHelp("l", "coluna dir"))
	keyShiftCardL = key.NewBinding(key.WithKeys("H"), key.WithHelp("H", "mover card esq"))
	keyShiftCardR = key.NewBinding(key.WithKeys("L"), key.WithHelp("L", "mover card dir"))
	keyReorderL   = key.NewBinding(key.WithKeys("ctrl+h"), key.WithHelp("ctrl+h", "coluna esq"))
	keyReorderR   = key.NewBinding(key.WithKeys("ctrl+l"), key.WithHelp("ctrl+l", "coluna dir"))
	keyCardDown   = key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j", "card baixo"))
	keyCardUp     = key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k", "card cima"))
)

// appKeymap is the full list of bindings the footer hints and the help modal
// render from.
var appKeymap = []key.Binding{
	keyMoveColL, keyMoveColR, keyCardDown, keyCardUp,
	keyShiftCardL, keyShiftCardR, keyReorderL, keyReorderR,
	keyPreview, keyLogs, keyOpen, keyNewCard, keyNewColumn, keyNewBoard,
	keyRenameCard, keyRenameCol, keyDue, keyDelCard, keyDelCol,
	keySidebar, keyRefresh, keyHelp, keyQuit,
}
