package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// configDir points XDG_CONFIG_HOME at a temp dir and returns
// <tmp>/tabelakanban, where both the legacy "config" and the new
// "config.toml" live.
func configDir(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("TABELAKANBAN_CONFIG", "")
	t.Setenv("TABELAKANBAN_ROOT", "")
	settings = defaultConfig()

	dir := filepath.Join(base, "tabelakanban")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeCfg(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestConfigTOMLRoots(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config.toml"), `roots = ["/tmp/a", "/tmp/b"]`+"\n")

	roots, warn := loadRootsConfig()
	if warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	// Order is load-bearing: ops.go creates a new board under roots[0].
	if len(roots) != 2 || roots[0] != "/tmp/a" || roots[1] != "/tmp/b" {
		t.Fatalf("roots = %v, want [/tmp/a /tmp/b] in order", roots)
	}
}

func TestConfigFallsBackToLegacyFile(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config"), "# comentário\n/tmp/kanban\n/tmp/trabalho\n")

	roots, warn := loadRootsConfig()
	if !strings.Contains(warn, "config antigo") {
		t.Fatalf("warning = %q, want a migration hint", warn)
	}
	if len(roots) != 2 || roots[0] != "/tmp/kanban" {
		t.Fatalf("roots = %v, want the 2 legacy roots", roots)
	}
}

func TestConfigTOMLBeatsLegacy(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config"), "/tmp/legacy\n")
	writeCfg(t, filepath.Join(dir, "config.toml"), `roots = ["/tmp/novo"]`+"\n")

	roots, warn := loadRootsConfig()
	if warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	if len(roots) != 1 || roots[0] != "/tmp/novo" {
		t.Fatalf("roots = %v, want [/tmp/novo]", roots)
	}
}

func TestConfigPartialOverride(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config.toml"), "[layout]\ncard_lines = 5\n")

	if _, warn := loadRootsConfig(); warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	if cardLines() != 5 {
		t.Fatalf("cardLines() = %d, want 5", cardLines())
	}
	// Untouched keys keep their defaults.
	if panelGap() != 1 {
		t.Fatalf("panelGap() = %d, want the default 1", panelGap())
	}
	if sidebarTotal() != 22 {
		t.Fatalf("sidebarTotal() = %d, want the default 18+4", sidebarTotal())
	}
	if settings.Display.LogCapacity != 200 {
		t.Fatalf("LogCapacity = %d, want the default 200", settings.Display.LogCapacity)
	}
}

func TestConfigNoticeTimeoutParsesDuration(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config.toml"), `[display]`+"\n"+`notice_timeout = "750ms"`+"\n")

	if _, warn := loadRootsConfig(); warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	if got := settings.Display.NoticeTimeout.Duration; got != 750*time.Millisecond {
		t.Fatalf("NoticeTimeout = %v, want 750ms", got)
	}
}

func TestConfigDoneColumnMarkers(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config.toml"), `[ipc]`+"\n"+`done_column_markers = ["pronto"]`+"\n")

	if _, warn := loadRootsConfig(); warn != "" {
		t.Fatalf("warning = %q, want none", warn)
	}
	if !isDoneColumn("PRONTO") {
		t.Fatal(`isDoneColumn("PRONTO") = false, want true (match is case-insensitive)`)
	}
	// The list replaces the defaults outright.
	if isDoneColumn("done") {
		t.Fatal(`isDoneColumn("done") = true, want false — the configured list replaces the defaults`)
	}
}

func TestConfigNormalizeClampsUnusableValues(t *testing.T) {
	// card_lines = 0 would divide by zero when computing how many cards fit.
	got := normalize(config{Layout: layoutConfig{CardLines: 0, SidebarWidth: 0}})
	if got.Layout.CardLines != 3 {
		t.Fatalf("CardLines = %d, want the default 3", got.Layout.CardLines)
	}
	if got.Layout.SidebarWidth != 18 {
		t.Fatalf("SidebarWidth = %d, want the default 18", got.Layout.SidebarWidth)
	}
	if got.Display.LogCapacity != 200 {
		t.Fatalf("LogCapacity = %d, want the default 200", got.Display.LogCapacity)
	}
	if got.Display.NoticeTimeout.Duration != 2*time.Second {
		t.Fatalf("NoticeTimeout = %v, want the default 2s", got.Display.NoticeTimeout.Duration)
	}
}

func TestConfigMalformedTOMLWarnsAndKeepsScanning(t *testing.T) {
	dir := configDir(t)
	writeCfg(t, filepath.Join(dir, "config.toml"), "roots = [\nbroken")

	roots, warn := loadRootsConfig()
	if warn == "" {
		t.Fatal("warning = empty, want the parse error surfaced")
	}
	if len(roots) == 0 {
		t.Fatal("roots = empty, want the defaults so the scan still runs")
	}
}
