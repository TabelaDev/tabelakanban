package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "list":
			os.Exit(runList())
		case "ipc":
			os.Exit(runIPC(os.Args[2:]))
		}
	}

	p := tea.NewProgram(newModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

// runList prints a plain-text dump of the boards/columns/cards and exits —
// no TTY/altscreen needed, useful for piping into other tools.
func runList() int {
	roots, cfgWarning := loadRootsConfig()
	boards, warnings := scanBoards(roots)
	if cfgWarning != "" {
		fmt.Fprintln(os.Stderr, "aviso:", cfgWarning)
	}
	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "aviso:", w)
	}
	for _, b := range boards {
		fmt.Printf("%s (%d colunas)\n", b.Name, len(b.Columns))
		for _, c := range b.Columns {
			fmt.Printf("  %s (%d cards)\n", c.Name, len(c.Cards))
			for _, card := range c.Cards {
				fmt.Printf("    - %s\n", card.Title)
			}
		}
	}
	return 0
}
