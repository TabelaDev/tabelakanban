package main

import (
	"github.com/ianptkcs/tabelatuiui"
)

// theme mirrors the installed DankMaterialShell's own configured accent
// (falling back to a manually chosen Catppuccin accent when DMS isn't
// installed/configured) — same lookup djobs and tabelaradar use, kept in
// sync so every tool's chrome matches. TABELAKANBAN_DMS_SETTINGS/
// TABELAKANBAN_ACCENT env vars override the defaults; see
// tabelatuiui.NewThemeFromEnv.
var theme = tuiui.NewThemeFromEnv("TABELAKANBAN")

var (
	colBase     = theme.Base
	colMantle   = theme.Mantle
	colSurface1 = theme.Surface1
	colOverlay0 = theme.Overlay0
	colOverlay1 = theme.Overlay1
	colText     = theme.Text
	colSubtext0 = theme.Subtext0
	colPrimary  = theme.Primary
	colRed      = theme.Red
)
