package main

import (
	"path/filepath"

	"github.com/charmbracelet/bubbles/key"
	"github.com/ianptkcs/tabelatuiui"
)

// reg is tabelakanban's single source of truth for keybindings: defaults
// registered below, overrides persisted to ~/.config/tabelakanban/keybindings.json.
// Resolve() returns the effective binding, shared by dispatch, footer and
// help modal — a user rebind via the settings modal applies to all at once.
var reg = tuiui.NewKeyRegistry(filepath.Join(tuiui.ConfigDir(), "tabelakanban", "keybindings.json"))

func init() {
	reg.RegisterMany(
		tuiui.Action{ID: "quit", Help: "sair", Keys: []string{"q", "ctrl+c"}},
		tuiui.Action{ID: "help", Help: "keybindings", Keys: []string{"?"}},
		tuiui.Action{ID: "settings", Help: "rebind keys", Keys: []string{","}},
		tuiui.Action{ID: "refresh", Help: "recarregar", Keys: []string{"ctrl+r"}},
		tuiui.Action{ID: "sidebar", Help: "sidebar", Keys: []string{"ctrl+e"}},
		tuiui.Action{ID: "logs", Help: "logs", Keys: []string{"g"}},
		tuiui.Action{ID: "new-card", Help: "novo card", Keys: []string{"n"}},
		tuiui.Action{ID: "new-column", Help: "nova coluna", Keys: []string{"N"}},
		tuiui.Action{ID: "new-board", Help: "novo board", Keys: []string{"B"}},
		tuiui.Action{ID: "rename-card", Help: "renomear card", Keys: []string{"r"}},
		tuiui.Action{ID: "rename-column", Help: "renomear coluna", Keys: []string{"R"}},
		tuiui.Action{ID: "due", Help: "due date", Keys: []string{"t"}},
		tuiui.Action{ID: "delete-card", Help: "del card", Keys: []string{"d"}},
		tuiui.Action{ID: "delete-column", Help: "del coluna", Keys: []string{"D"}},
		tuiui.Action{ID: "preview", Help: "preview", Keys: []string{"o"}},
		tuiui.Action{ID: "open", Help: "editar", Keys: []string{"enter"}},
		tuiui.Action{ID: "move-col-left", Help: "coluna esq", Keys: []string{"h", "left"}, Label: "h"},
		tuiui.Action{ID: "move-col-right", Help: "coluna dir", Keys: []string{"l", "right"}, Label: "l"},
		tuiui.Action{ID: "shift-card-left", Help: "mover card esq", Keys: []string{"H"}},
		tuiui.Action{ID: "shift-card-right", Help: "mover card dir", Keys: []string{"L"}},
		tuiui.Action{ID: "reorder-col-left", Help: "coluna esq", Keys: []string{"ctrl+h"}},
		tuiui.Action{ID: "reorder-col-right", Help: "coluna dir", Keys: []string{"ctrl+l"}},
		tuiui.Action{ID: "card-down", Help: "card baixo", Keys: []string{"j", "down"}, Label: "j"},
		tuiui.Action{ID: "card-up", Help: "card cima", Keys: []string{"k", "up"}, Label: "k"},
	)
}

// resolve is a short alias so Update reads like the old named keys.
func resolve(id string) key.Binding { return reg.Resolve(id) }
