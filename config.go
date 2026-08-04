package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/ianptkcs/tabelatuiui"
)

// configPath is the settings file listing which board-root directories to
// scan. TABELAKANBAN_CONFIG overrides it, same pattern as tabelaradar.
var configPath = tuiui.EnvOr("TABELAKANBAN_CONFIG", filepath.Join(tuiui.ConfigDir(), "tabelakanban", "config"))

// defaultRoots scans TABELAKANBAN_ROOT, or ~/kanban if unset — the same
// behavior as before any config file existed.
func defaultRoots() []string {
	return []string{tuiui.EnvOr("TABELAKANBAN_ROOT", filepath.Join(tuiui.HomeDir(), "kanban"))}
}

// loadRootsConfig reads configPath, one board-root per line:
//
//	~/kanban
//	~/codigo/pessoal/meu-time  # another group of boards, each subdir is a board
//
// Blank lines and lines starting with "#" are ignored. Returns a non-empty
// warning string (instead of an error) when the file exists but couldn't be
// read in full — scanning still proceeds with whatever roots were parsed
// before the failure, or the default root if none were.
func loadRootsConfig() ([]string, string) {
	f, err := os.Open(configPath)
	if err != nil {
		return defaultRoots(), ""
	}
	defer f.Close()

	var roots []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		roots = append(roots, tuiui.ExpandHome(line))
	}
	if err := scanner.Err(); err != nil {
		return roots, fmt.Sprintf("erro lendo %s: %v", configPath, err)
	}
	if len(roots) == 0 {
		return defaultRoots(), ""
	}
	return roots, ""
}
