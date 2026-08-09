package main

import (
	"github.com/ianptkcs/tabelatuiui"
)

// App-specific styles on top of tabelatuiui's shared chrome (called as
// theme.Header/Footer/Panel/Title/Dim directly). Colors live in the theme
// resolved in theme.go (Catppuccin Mocha + the DMS accent).

// Layout helpers come straight from the lib.

func padLines(s string, width int) string {
	if width < 0 {
		width = 0
	}
	return tuiui.PadLines(s, width)
}
func wrapText(s string, width int) string { return tuiui.WrapText(s, width) }
func padToHeight(s string, lines int) string {
	if lines < 0 {
		lines = 0
	}
	return tuiui.PadToHeight(s, lines)
}
