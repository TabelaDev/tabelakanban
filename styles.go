package main

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/ianptkcs/tabelatuiui"
)

// Thin wrappers over tabelatuiui's shared chrome, so the model/view code
// keeps calling the same short helpers the other TUIs do.

func headerStyle(width int) lipgloss.Style { return theme.Header(width) }
func footerStyle(width int) lipgloss.Style { return theme.Footer(width) }
func panelStyle(focused bool) lipgloss.Style {
	return theme.Panel(focused)
}
func titleStyle() lipgloss.Style { return theme.Title() }
func dimStyle() lipgloss.Style   { return theme.Dim() }

// Layout helpers come straight from the lib.

func padLines(s string, width int) string { return tuiui.PadLines(s, width) }
func wrapText(s string, width int) string { return tuiui.WrapText(s, width) }

// cardStyle is the per-card block inside a column: highlighted when the
// card is selected (inverted to primary), plain text otherwise.
func cardStyle(selected bool) lipgloss.Style {
	if selected {
		return lipgloss.NewStyle().Foreground(colBase).Background(colPrimary).Bold(true)
	}
	return lipgloss.NewStyle().Foreground(colText).Background(colBase)
}
