<div align="center">

# TabelaKanban

**A kanban TUI over plain text files** — every card is a `.md`, every column is a
folder, every board is a folder of folders. Move a card and you move a file; edit
it in your `$EDITOR` and git takes care of the rest.

**English** · [Português](README.pt-BR.md)

[![Go Version](https://img.shields.io/github/go-mod/go-version/TabelaDev/tabelakanban?style=flat-square&logo=go&logoColor=white&color=00ADD8)](go.mod)
[![Built with Bubble Tea](https://img.shields.io/badge/built%20with-Bubble%20Tea-ff69b4?style=flat-square)](https://github.com/charmbracelet/bubbletea)
[![Powered by tabelatuiui](https://img.shields.io/badge/theme-tabelatuiui-d6b4f7?style=flat-square)](https://github.com/TabelaDev/tabelatuiui)
[![License: AGPL-3.0](https://img.shields.io/badge/license-AGPL--3.0-blue?style=flat-square)](LICENSE)

[![ko-fi](https://ko-fi.com/img/githubbutton_sm.svg)](https://ko-fi.com/ianptkcs)

</div>

---

## What it is

A [Bubble Tea](https://github.com/charmbracelet/bubbletea) kanban TUI that lives
on top of folders and markdown files, with no database at all. The structure on
disk IS the board:

```
~/kanban/
  exemplo/
    backlog/
      configurar-ci.md
      escrever-readme.md
    fazendo/
      extrair-lib.md
    feito/
      renomear.md
```

Board = folder, column = subfolder, card = `.md` file. No separate metadata:
moving a card is a file `mv` (which git can follow), the title is the filename and
the body is the content. Press `enter` to open it in your `$EDITOR` and the card is
just the markdown you already write every day.

![screenshot](screenshot.png)

## Contents

- [Installation](#installation)
- [Usage](#usage)
- [IPC](#ipc)
- [Configuration](#configuration)
- [License](#license)

## Installation

Requires Go 1.26+.

```bash
go install github.com/ianptkcs/tabelakanban@latest
```

Or building from source:

```bash
git clone https://github.com/TabelaDev/tabelakanban.git
cd tabelakanban
go build -o tabelakanban .
```

To use it as a global command (needs `~/.local/bin` on your `PATH`):

```bash
go build -o ~/.local/bin/tabelakanban .
```

## Usage

```
tabelakanban         # opens the TUI
tabelakanban list    # plain-text dump, no TTY — useful for scripting
```

Inside the TUI:

- `h`/`l` move between columns — on the first column, `h` enters the boards
  sidebar, where `j`/`k` switch board and `enter`/`l` go back to the columns.
- `j`/`k` move between the cards of the focused column.
- `H`/`L` move the selected card to the neighbouring column.
- `ctrl+h`/`ctrl+l` reorder the focused column (moving the column itself).
- `n` creates a card, `N` a column, `B` a board (the new board lands in the first
  root), `r` renames the selected card, `R` the column, `d` deletes the card and
  `D` the column (all with a confirmation or modal prompt).
- `t` sets or clears the selected card's due date (`DD/MM`, empty clears it). The
  due date lives in the `.md`'s front matter and shows in the card's footer (red
  when overdue); creating a card already lets you fill it in.
- `o` opens and closes a side column with the markdown preview of the selected
  card; `enter` opens the card in `$EDITOR` (`nvim` by default).
- `g` opens the activity log page (what has been done); `q`/`esc`/`h`/`l` closes
  it.
- `ctrl+e` collapses and expands the boards sidebar (visible by default),
  `ctrl+r` rescans, `q` quits.

Cards within a column appear in alphabetical order of title.

## IPC

For scripts, or for an LLM to ask "what is in the queue and which column is each
thing in" without opening the TUI, `tabelakanban` exposes the same
`ipc <method> --json` subcommand as `djobs`/`tabelaradar`:

```bash
tabelakanban ipc boards.list --json                 # every board, column and card
tabelakanban ipc boards.list name=exemplo --json    # a single board
tabelakanban ipc boards.next --json                 # the card tabelakanban itself would prioritise
tabelakanban ipc cards.create board=exemplo column=a-fazer title=nova --json
tabelakanban ipc cards.move board=exemplo from=a-fazer to=feito title=nova --json
tabelakanban ipc cards.update board=exemplo column=feito title=nova 'body=...' --json
```

`cards.update` replaces a card's body (its `due:` front matter survives) — the
write counterpart to `cards.create`/`cards.move`, for an external editor (the
tabelaradar digest) to append progress notes or rewrite a checklist.

`boards.next` returns the first card of the first column that does not look
"done" (by name: `done`, `feito`, `conclu...`) — in the same spirit as
tabelaradar's `projects.next`.

## Configuration

Everything lives in `~/.config/tabelakanban/config.toml` (overridable through
`TABELAKANBAN_CONFIG`). The file is optional and partial: only the keys present
override anything, the rest stay on their defaults. `f5` reloads without
restarting.

```toml
# Folders whose children are boards. Order matters: a new board is born in the 1st.
roots = ["~/kanban", "~/codigo/pessoal/meu-time"]

[layout]
card_lines   = 3   # height of a card (title + preview + spacer)
panel_gap    = 1
sidebar_width = 18 # content; border and padding add 4

[display]
notice_timeout = "2s"
log_capacity   = 200

[ipc]
# Only `boards.next` uses this — the TUI has no notion of a "done" column.
done_column_markers = ["done", "feito", "conclu"]

[general]
editor = "nvim"  # empty = use $EDITOR, then nvim
```

With no file at all, it scans only `TABELAKANBAN_ROOT` (or `~/kanban`).

### Migrating from the old format

The config used to be `~/.config/tabelakanban/config`, one folder per line. **That
file is still read** when no `config.toml` exists, with a warning on the notice
line. The translation is direct:

```
~/kanban                    →  roots = ["~/kanban"]
~/codigo/pessoal/meu-time      (both in the same list, in the same order)
```

Once `config.toml` exists, it takes over on its own.

### Other variables

- `TABELAKANBAN_ROOT` — the root scanned when no config exists at all (`~/kanban`
  by default).
- `TABELAKANBAN_ACCENT` — a manual Catppuccin Mocha accent, used only when
  DankMaterialShell is not installed or configured (`mauve` by default).
- `TABELAKANBAN_DMS_SETTINGS` — path to the DMS `settings.json`, when it is not the
  default one.

The theme and the shared chrome (header/footer/panels, ANSI-aware padding, IPC
helpers) come from
[`tabelatuiui`](https://github.com/TabelaDev/tabelatuiui).

## Development

```bash
go test ./...
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for the version history.

## Support the project

- **Global**: [ko-fi.com/ianptkcs](https://ko-fi.com/ianptkcs)
- **Brazil (Pix)**: scan the QR below or copy the code

  <img src="pix-qr.png" alt="Pix QR" width="200" />

  <details><summary>Pix code (copy)</summary>

  ```
  00020126580014BR.GOV.BCB.PIX01365ad933b0-dcdc-4525-a736-0759902aeec65204000053039865802BR5925Ian Patrick da Costa Soar6009SAO PAULO62140510tQA85x6Dov63041FB6
  ```

  </details>

## License

[GNU AGPL-3.0](LICENSE) — free and open source. If you run a modified version of
this project, including as a network service, you also have to make the modified
source available under the same license.
