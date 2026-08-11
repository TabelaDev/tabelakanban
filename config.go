package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ianptkcs/tabelatuiui"
)

// config is tabelakanban's settings schema, read from
// ~/.config/tabelakanban/config.toml. Every field falls back to
// defaultConfig when the file leaves it out.
type config struct {
	// Roots are the board-group directories to scan. Order matters: a new
	// board is always created under the first one (see ops.go).
	Roots   []string      `toml:"roots"`
	Layout  layoutConfig  `toml:"layout"`
	Display displayConfig `toml:"display"`
	IPC     ipcConfig     `toml:"ipc"`
	General generalConfig `toml:"general"`
}

type layoutConfig struct {
	// CardLines is the height of one card block (title + preview + spacer).
	CardLines int `toml:"card_lines"`
	PanelGap  int `toml:"panel_gap"`
	// SidebarWidth is the board sidebar's content width; its border and
	// padding add 4 on top (see sidebarTotal).
	SidebarWidth int `toml:"sidebar_width"`
}

type displayConfig struct {
	// NoticeTimeout is how long the notice line holds a message.
	NoticeTimeout duration `toml:"notice_timeout"`
	// LogCapacity is how many activity entries are kept in memory.
	LogCapacity int `toml:"log_capacity"`
}

type ipcConfig struct {
	// DoneColumnMarkers are matched case-insensitively against column names
	// to decide what counts as "done". Only `ipc boards.next` uses this — the
	// TUI itself has no notion of a done column.
	DoneColumnMarkers []string `toml:"done_column_markers"`
}

type generalConfig struct {
	// Editor overrides $EDITOR when opening a card.
	Editor string `toml:"editor"`
}

// duration wraps time.Duration so TOML can express it as "2s" instead of a
// raw nanosecond count.
type duration struct{ time.Duration }

func (d *duration) UnmarshalText(text []byte) error {
	parsed, err := time.ParseDuration(string(text))
	if err != nil {
		return err
	}
	d.Duration = parsed
	return nil
}

func defaultConfig() config {
	return config{
		Roots: []string{tuiui.EnvOr("TABELAKANBAN_ROOT", filepath.Join(tuiui.HomeDir(), "kanban"))},
		Layout: layoutConfig{
			CardLines:    3,
			PanelGap:     1,
			SidebarWidth: 18,
		},
		Display: displayConfig{
			NoticeTimeout: duration{2 * time.Second},
			LogCapacity:   200,
		},
		IPC:     ipcConfig{DoneColumnMarkers: []string{"done", "feito", "conclu"}},
		General: generalConfig{Editor: ""}, // empty = fall back to $EDITOR, then nvim
	}
}

// configPath is resolved lazily, not in a package-level var: an init-time var
// would freeze TABELAKANBAN_CONFIG/XDG_CONFIG_HOME before main (or a test)
// could set them.
func configPath() string {
	return tuiui.EnvOr("TABELAKANBAN_CONFIG", tuiui.ConfigPath("tabelakanban", "config.toml"))
}

// legacyConfigPath is the pre-TOML file: one board-root per line. Still read
// when no config.toml exists, so an existing install keeps working.
func legacyConfigPath() string {
	return tuiui.ConfigPath("tabelakanban", "config")
}

// settings is the normalized snapshot the app reads from.
var settings = defaultConfig()

// normalize clamps values the renderer cannot survive: a card height of zero
// divides by zero when computing how many cards fit in a column.
func normalize(c config) config {
	d := defaultConfig()
	if c.Layout.CardLines < 1 {
		c.Layout.CardLines = d.Layout.CardLines
	}
	if c.Layout.PanelGap < 0 {
		c.Layout.PanelGap = d.Layout.PanelGap
	}
	if c.Layout.SidebarWidth < 1 {
		c.Layout.SidebarWidth = d.Layout.SidebarWidth
	}
	if c.Display.NoticeTimeout.Duration <= 0 {
		c.Display.NoticeTimeout = d.Display.NoticeTimeout
	}
	if c.Display.LogCapacity < 1 {
		c.Display.LogCapacity = d.Display.LogCapacity
	}
	if len(c.IPC.DoneColumnMarkers) == 0 {
		c.IPC.DoneColumnMarkers = d.IPC.DoneColumnMarkers
	}
	if len(c.Roots) == 0 {
		c.Roots = d.Roots
	}
	return c
}

// refreshSettings re-reads config.toml from disk and returns a warning string
// (never an error) — a bad config file must not stop the scan.
// The Config is built per call rather than kept in a package var: both the
// path (TABELAKANBAN_CONFIG) and the defaults (TABELAKANBAN_ROOT) come from
// the environment, and a cached instance would freeze whatever they were on
// the first call. Nothing is lost — this app re-reads on every rescan anyway
// and never consults Reload's "changed" flag.
func refreshSettings() string {
	path := configPath()

	// No config.toml yet: fall back to the pre-TOML file if it's still there,
	// so an existing install keeps its roots after upgrading.
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if legacy, ok := loadLegacyConfig(); ok {
			settings = normalize(legacy)
			return fmt.Sprintf("lendo o config antigo %s — migre pra %s", legacyConfigPath(), path)
		}
	}

	cfg := tuiui.NewConfig(path, defaultConfig())
	err := cfg.Load()
	settings = normalize(cfg.Get())
	if err != nil {
		return fmt.Sprintf("erro lendo %s: %v", path, err)
	}
	return ""
}

// loadLegacyConfig reads the pre-TOML format: one board-root per line, "#"
// for comments. Reports false when the file is absent or has no entries.
func loadLegacyConfig() (config, bool) {
	f, err := os.Open(legacyConfigPath())
	if err != nil {
		return config{}, false
	}
	defer f.Close()

	c := defaultConfig()
	c.Roots = nil
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		c.Roots = append(c.Roots, line)
	}
	if scanner.Err() != nil || len(c.Roots) == 0 {
		return config{}, false
	}
	return c, true
}

// loadRootsConfig re-reads the config file and returns the board roots plus a
// warning string. It re-reads on every call so an external edit is picked up
// by a rescan without a restart.
func loadRootsConfig() ([]string, string) {
	warn := refreshSettings()
	roots := make([]string, 0, len(settings.Roots))
	for _, r := range settings.Roots {
		roots = append(roots, tuiui.ExpandHome(r))
	}
	return roots, warn
}
