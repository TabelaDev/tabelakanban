package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/ianptkcs/tabelatuiui"
)

// mode tracks which interaction is on top of the kanban: normal board
// navigation, a text input modal, a delete confirmation, or the activity log.
type mode int

const (
	modeBoard mode = iota
	modeInput
	modeConfirm
	modeLogs
)

// inputKind says what the text input is creating/renaming.
type inputKind int

const (
	inputNewCard inputKind = iota
	inputNewColumn
	inputRenameCard
	inputNewBoard
	inputRenameColumn
	inputDue
)

const (
	// header/footer bands are single-line, flush against the terminal edges.
	headerLines = 1
	noticeLines = 1
	footerLines = 1
	// The frame has no outer margin; the header/footer's own horizontal
	// padding keeps the text off the edges.
	appMargin = 0
	// Column box: 2 border + 1 title + 1 blank = 4 lines of overhead; each
	// card block is 3 lines (title + preview + spacer).
	colBoxOverhead = 4
	cardLines      = 3
	panelGap       = 1
	minVisibleCols = 1
	minVisibleRows = 3
	// Smallest a column may shrink to before we just horizontally scroll.
	minColWidth = 20
	// Board sidebar width (content; border+padding add 4).
	sidebarInnerWidth = 18
	sidebarTotal      = sidebarInnerWidth + 4
)

// logEntry is one structured activity in the log: what happened to which
// board/card/column, with the source → destination columns when relevant.
type logEntry struct {
	at     time.Time
	kind   string // "card", "coluna", "board", "erro", "aviso"
	action string // "criado", "movido", "renomeado", "apagado", "reordenada", ...
	target string // name of the card/column/board
	board  string
	from   string
	to     string
	detail string
}

const logCap = 200

// summary is the one-line version shown in the notice line.
func (e logEntry) summary() string {
	switch e.kind {
	case "erro", "aviso":
		return e.detail
	}
	line := fmt.Sprintf("%s '%s' %s", e.kind, e.target, e.action)
	if e.to != "" {
		line += " para " + e.to
	}
	return line
}

// rich is the multi-line version shown on the logs page.
func (e logEntry) rich() []string {
	head := theme.Dim().Render(e.at.Format("15:04:05")) + "  " + e.summary()
	switch e.kind {
	case "erro", "aviso":
		return []string{head}
	}
	var parts []string
	if e.board != "" {
		parts = append(parts, "board "+e.board)
	}
	if e.from != "" && e.to != "" {
		parts = append(parts, e.from+" → "+e.to)
	} else if e.to != "" {
		parts = append(parts, "para "+e.to)
	}
	if e.detail != "" {
		parts = append(parts, e.detail)
	}
	if len(parts) == 0 {
		return []string{head}
	}
	return []string{head, "           " + strings.Join(parts, " · ")}
}

type appModel struct {
	roots    []string
	boards   []Board
	boardIdx int
	// colIdx is the focused column; cardIdx the focused card within it.
	colIdx  int
	cardIdx int
	// colScroll is the leftmost visible column — horizontal scroll for
	// boards with more columns than fit on screen.
	colScroll int

	// sidebar is the collapsible board column on the left (ctrl+e).
	sidebar        bool
	sidebarFocused bool
	preview        bool
	mode           mode
	form           *huh.Form
	inputKind      inputKind
	confirmCard    Card
	confirmCol     *Column
	width          int
	height         int
	notice         string
	noticeGen      int
	log            []logEntry

	// helpModal is the "?" overlay listing every keybinding — declared here
	// (not per-update) so its scroll position survives toggles.
	helpModal *tuiui.HelpModal
}

func newModel() appModel {
	m := appModel{
		sidebar: true,
		helpModal: tuiui.NewHelpModal(tuiui.HelpSection{
			Title:    "Atalhos",
			Bindings: appKeymap,
		}),
	}
	m.rescan(false)
	return m
}

// clearNoticeMsg clears the notice when its generation matches the current
// one — a newer notice supersedes an older clear.
type clearNoticeMsg struct{ gen int }

// notifyEntry records an activity in the log and shows its summary in the
// reserved notice line for a couple of seconds.
func (m *appModel) notifyEntry(e logEntry) tea.Cmd {
	m.log = append(m.log, e)
	if len(m.log) > logCap {
		m.log = m.log[len(m.log)-logCap:]
	}
	m.notice = e.summary()
	m.noticeGen++
	gen := m.noticeGen
	return tea.Tick(2*time.Second, func(time.Time) tea.Msg {
		return clearNoticeMsg{gen: gen}
	})
}

func (m *appModel) logAction(kind, action, target, board, from, to string) tea.Cmd {
	return m.notifyEntry(logEntry{at: time.Now(), kind: kind, action: action, target: target, board: board, from: from, to: to})
}

func (m *appModel) logInfo(detail string) tea.Cmd {
	return m.notifyEntry(logEntry{at: time.Now(), kind: "aviso", detail: detail})
}

func (m *appModel) logError(detail string) tea.Cmd {
	return m.notifyEntry(logEntry{at: time.Now(), kind: "erro", detail: detail})
}

// rescan reloads boards from disk. It keeps boardIdx/colIdx pinned to the
// same board/column by name when possible, so a refresh (e.g. after the
// editor closes) doesn't jump the cursor. feedback=true surfaces a
// confirmation notice (used by the explicit ctrl+r reload).
func (m *appModel) rescan(feedback bool) tea.Cmd {
	prevBoard, prevColumn := m.currentBoardName(), m.currentColumnName()
	roots, cfgWarning := loadRootsConfig()
	boards, warnings := scanBoards(roots)
	if cfgWarning != "" {
		warnings = append([]string{cfgWarning}, warnings...)
	}

	m.roots = roots
	m.boards = boards
	m.boardIdx = indexOfBoard(boards, prevBoard)
	m.colIdx = indexOfColumn(m.currentBoard(), prevColumn)

	if len(warnings) > 0 {
		return m.logInfo(strings.Join(warnings, "; "))
	}
	if feedback {
		return m.logInfo(fmt.Sprintf("recarregado — %d board(s)", len(boards)))
	}
	return nil
}

func (m *appModel) currentBoard() *Board {
	if m.boardIdx < 0 || m.boardIdx >= len(m.boards) {
		return nil
	}
	return &m.boards[m.boardIdx]
}

func (m *appModel) currentBoardName() string {
	if b := m.currentBoard(); b != nil {
		return b.Name
	}
	return ""
}

func (m *appModel) currentColumn() *Column {
	b := m.currentBoard()
	if b == nil || m.colIdx < 0 || m.colIdx >= len(b.Columns) {
		return nil
	}
	return &b.Columns[m.colIdx]
}

func (m *appModel) currentColumnName() string {
	if c := m.currentColumn(); c != nil {
		return c.Name
	}
	return ""
}

func (m *appModel) currentCard() *Card {
	c := m.currentColumn()
	if c == nil || m.cardIdx < 0 || m.cardIdx >= len(c.Cards) {
		return nil
	}
	return &c.Cards[m.cardIdx]
}

func indexOfBoard(boards []Board, name string) int {
	for i, b := range boards {
		if b.Name == name {
			return i
		}
	}
	return 0
}

func indexOfColumn(b *Board, name string) int {
	if b == nil {
		return 0
	}
	for i, c := range b.Columns {
		if c.Name == name {
			return i
		}
	}
	return 0
}

func (m appModel) Init() tea.Cmd { return nil }

func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if sizeMsg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width, m.height = sizeMsg.Width, sizeMsg.Height
		m.helpModal.SetSize(sizeMsg.Width, sizeMsg.Height)
		m.reclamp()
		return m, nil
	}
	if cm, ok := msg.(clearNoticeMsg); ok {
		if cm.gen == m.noticeGen {
			m.notice = ""
		}
		return m, nil
	}
	if _, ok := msg.(editorFinishedMsg); ok {
		return m, m.rescan(false)
	}

	// The help modal swallows all keys while it's open — the app must not
	// act on them (so "q" closes the modal instead of quitting, etc.).
	if m.helpModal.Update(msg) {
		return m, nil
	}

	switch m.mode {
	case modeInput:
		return m.updateInput(msg)
	case modeConfirm:
		return m.updateConfirm(msg)
	case modeLogs:
		return m.updateLogs(msg)
	default:
		return m.updateBoard(msg)
	}
}

func (m *appModel) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch {
	case key.Matches(keyMsg, keyQuit):
		return m, tea.Quit
	case key.Matches(keyMsg, keyHelp):
		m.helpModal.Toggle()
		return m, nil
	case key.Matches(keyMsg, keyRefresh):
		return m, m.rescan(true)
	case key.Matches(keyMsg, keySidebar):
		m.sidebar = !m.sidebar
		if !m.sidebar {
			m.sidebarFocused = false
		}
		return m, nil
	case key.Matches(keyMsg, keyLogs):
		m.mode = modeLogs
		return m, nil
	case key.Matches(keyMsg, keyNewCard):
		return m.startInput(inputNewCard, "")
	case key.Matches(keyMsg, keyNewColumn):
		return m.startInput(inputNewColumn, "")
	case key.Matches(keyMsg, keyNewBoard):
		return m.startInput(inputNewBoard, "")
	case key.Matches(keyMsg, keyRenameCard):
		if card := m.currentCard(); card != nil && !m.sidebarFocused {
			return m.startInput(inputRenameCard, card.Title)
		}
	case key.Matches(keyMsg, keyRenameCol):
		if col := m.currentColumn(); col != nil && !m.sidebarFocused {
			return m.startInput(inputRenameColumn, col.Name)
		}
	case key.Matches(keyMsg, keyDue):
		if card := m.currentCard(); card != nil && !m.sidebarFocused {
			return m.startInput(inputDue, dueDDMM(card.Due))
		}
	case key.Matches(keyMsg, keyDelCard):
		if card := m.currentCard(); card != nil && !m.sidebarFocused {
			m.confirmCard = *card
			m.confirmCol = nil
			m.mode = modeConfirm
		}
	case key.Matches(keyMsg, keyDelCol):
		if col := m.currentColumn(); col != nil && !m.sidebarFocused {
			m.confirmCol = col
			m.mode = modeConfirm
		}
	case key.Matches(keyMsg, keyPreview):
		if b := m.currentBoard(); b != nil {
			m.preview = !m.preview
		}
	case key.Matches(keyMsg, keyOpen):
		if m.sidebarFocused {
			m.sidebarFocused = false
		} else if card := m.currentCard(); card != nil {
			return m, openEditor(card.Path)
		}
	case key.Matches(keyMsg, keyMoveColL):
		if m.sidebarFocused {
			// already on the sidebar
		} else if m.colIdx == 0 && m.sidebar {
			m.sidebarFocused = true
			m.cardIdx = 0
		} else if m.colIdx > 0 {
			m.colIdx--
			m.cardIdx = 0
		}
	case key.Matches(keyMsg, keyMoveColR):
		if m.sidebarFocused {
			m.sidebarFocused = false
		} else if c := m.currentBoard(); c != nil && m.colIdx < len(c.Columns)-1 {
			m.colIdx++
			m.cardIdx = 0
		}
	case key.Matches(keyMsg, keyShiftCardL):
		return m, m.moveCardBetween(-1)
	case key.Matches(keyMsg, keyShiftCardR):
		return m, m.moveCardBetween(1)
	case key.Matches(keyMsg, keyReorderL):
		return m, m.moveColumnBetween(-1)
	case key.Matches(keyMsg, keyReorderR):
		return m, m.moveColumnBetween(1)
	case key.Matches(keyMsg, keyCardDown):
		if m.sidebarFocused {
			if m.boardIdx < len(m.boards)-1 {
				m.boardIdx++
				m.colIdx, m.cardIdx = 0, 0
			}
		} else if c := m.currentColumn(); c != nil && m.cardIdx < len(c.Cards)-1 {
			m.cardIdx++
		}
	case key.Matches(keyMsg, keyCardUp):
		if m.sidebarFocused {
			if m.boardIdx > 0 {
				m.boardIdx--
				m.colIdx, m.cardIdx = 0, 0
			}
		} else if m.cardIdx > 0 {
			m.cardIdx--
		}
	}
	m.reclamp()
	return m, nil
}

// moveCardBetween moves the selected card to the adjacent column.
func (m *appModel) moveCardBetween(delta int) tea.Cmd {
	card := m.currentCard()
	col := m.currentColumn()
	b := m.currentBoard()
	if card == nil || col == nil || b == nil || m.sidebarFocused {
		return nil
	}
	next := m.colIdx + delta
	if next < 0 || next >= len(b.Columns) {
		return m.logInfo("não há coluna nessa direção")
	}
	fromName, toName := col.Name, b.Columns[next].Name
	moved, err := moveCard(*card, b.Columns[next])
	if err != nil {
		return m.logError("erro movendo card: " + err.Error())
	}
	m.rescan(false)
	m.colIdx = next
	m.cardIdx = indexOfCard(m.boards, b.Columns[next].Path, moved.Title)
	m.reclamp()
	return m.logAction("card", "movido", moved.Title, b.Name, fromName, toName)
}

// moveColumnBetween reorders the board's columns so the focused column moves
// one position left or right.
func (m *appModel) moveColumnBetween(delta int) tea.Cmd {
	b := m.currentBoard()
	if b == nil || m.sidebarFocused {
		return nil
	}
	next := m.colIdx + delta
	if next < 0 || next >= len(b.Columns) {
		return m.logInfo("não há coluna nessa direção")
	}
	name := b.Columns[m.colIdx].Name
	if err := moveColumn(*b, m.colIdx, next); err != nil {
		return m.logError("erro movendo coluna: " + err.Error())
	}
	m.rescan(false)
	m.colIdx = next
	m.reclamp()
	return m.logAction("coluna", "reordenada", name, b.Name, "", "")
}

func (m *appModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "y":
		if m.confirmCol != nil {
			col := *m.confirmCol
			boardName := m.currentBoardName()
			m.confirmCol = nil
			if err := deleteColumn(col); err != nil {
				return m, m.logError("erro apagando coluna: " + err.Error())
			}
			m.mode = modeBoard
			m.rescan(false)
			m.reclamp()
			return m, m.logAction("coluna", "apagada", col.Name, boardName, "", "")
		}
		card := m.confirmCard
		boardName := m.currentBoardName()
		colName := m.currentColumnName()
		if err := deleteCard(card); err != nil {
			return m, m.logError("erro apagando: " + err.Error())
		}
		m.mode = modeBoard
		m.rescan(false)
		m.reclamp()
		return m, m.logAction("card", "apagado", card.Title, boardName, colName, "")
	case "n", "esc", "q":
		m.mode = modeBoard
		m.confirmCol = nil
	}
	return m, nil
}

func (m *appModel) updateLogs(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "g", "l", "h", "q", "esc", "ctrl+c":
			m.mode = modeBoard
		}
	}
	return m, nil
}

func (m *appModel) startInput(kind inputKind, value string) (tea.Model, tea.Cmd) {
	m.inputKind = kind
	m.form = newInputForm(kind, value)
	m.mode = modeInput
	return m, m.form.Init()
}

func (m *appModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.form == nil {
		m.mode = modeBoard
		return m, nil
	}
	if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
		m.form = nil
		m.mode = modeBoard
		return m, nil
	}
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}
	switch m.form.State {
	case huh.StateCompleted:
		m.commitInput()
		m.form = nil
		m.mode = modeBoard
	case huh.StateAborted:
		m.form = nil
		m.mode = modeBoard
	}
	return m, cmd
}

func (m *appModel) commitInput() tea.Cmd {
	if m.form == nil {
		return nil
	}
	board := m.currentBoard()
	col := m.currentColumn()
	switch m.inputKind {
	case inputNewCard:
		title := strings.TrimSpace(m.form.GetString("title"))
		if col == nil {
			return m.logError("nenhuma coluna focada")
		}
		card, err := createCard(*col, title)
		if err != nil {
			return m.logError("erro criando card: " + err.Error())
		}
		if due, _ := parseDueDDMM(m.form.GetString("due")); !due.IsZero() {
			if card, err = setCardDue(card, due); err != nil {
				return m.logError("erro salvando due date: " + err.Error())
			}
		}
		boardName := board.Name
		m.rescan(false)
		m.cardIdx = indexOfCard(m.boards, col.Path, card.Title)
		m.reclamp()
		return m.logAction("card", "criado", card.Title, boardName, col.Name, "")
	case inputNewColumn:
		name := strings.TrimSpace(m.form.GetString("title"))
		if board == nil {
			return m.logError("nenhum board")
		}
		if err := createColumn(*board, name); err != nil {
			return m.logError("erro criando coluna: " + err.Error())
		}
		m.rescan(false)
		m.colIdx = indexOfColumn(m.currentBoard(), name)
		m.reclamp()
		return m.logAction("coluna", "criada", name, board.Name, "", "")
	case inputRenameCard:
		title := strings.TrimSpace(m.form.GetString("title"))
		card := m.currentCard()
		if card == nil {
			return m.logError("nenhum card")
		}
		boardName := m.currentBoardName()
		colName := m.currentColumnName()
		renamed, err := renameCard(*card, title)
		if err != nil {
			return m.logError("erro renomeando: " + err.Error())
		}
		m.rescan(false)
		m.cardIdx = indexOfCard(m.boards, m.currentColumnPath(), renamed.Title)
		m.reclamp()
		return m.logAction("card", "renomeado", renamed.Title, boardName, colName, "")
	case inputRenameColumn:
		name := strings.TrimSpace(m.form.GetString("title"))
		column := m.currentColumn()
		if column == nil {
			return m.logError("nenhuma coluna")
		}
		boardName := m.currentBoardName()
		renamed, err := renameColumn(*column, name)
		if err != nil {
			return m.logError("erro renomeando coluna: " + err.Error())
		}
		m.rescan(false)
		m.colIdx = indexOfColumn(m.currentBoard(), renamed.Name)
		m.reclamp()
		return m.logAction("coluna", "renomeada", renamed.Name, boardName, "", "")
	case inputNewBoard:
		name := strings.TrimSpace(m.form.GetString("title"))
		dir, err := createBoard(m.roots, name)
		if err != nil {
			return m.logError("erro criando board: " + err.Error())
		}
		m.rescan(false)
		m.boardIdx = indexOfBoard(m.boards, name)
		m.reclamp()
		return m.logAction("board", "criado", name, name, "", fmt.Sprintf("em %s", dir))
	case inputDue:
		card := m.currentCard()
		if card == nil || col == nil || board == nil {
			return m.logError("nenhum card")
		}
		due, err := parseDueDDMM(m.form.GetString("due"))
		if err != nil {
			return m.logError("data inválida: " + err.Error())
		}
		boardName, colName := board.Name, col.Name
		updated, err := setCardDue(*card, due)
		if err != nil {
			return m.logError("erro salvando due date: " + err.Error())
		}
		m.rescan(false)
		m.cardIdx = indexOfCard(m.boards, col.Path, updated.Title)
		m.reclamp()
		if due.IsZero() {
			return m.logAction("card", "sem prazo", updated.Title, boardName, colName, "")
		}
		return m.logAction("card", "com prazo", updated.Title, boardName, colName, due.Format("02/01"))
	}
	m.reclamp()
	return nil
}

// currentColumnPath is the focused column's directory, after a rescan.
func (m *appModel) currentColumnPath() string {
	if c := m.currentColumn(); c != nil {
		return c.Path
	}
	return ""
}

// indexOfCard finds a card by title within a column path — the column's
// identity survives a rescan via its path.
func indexOfCard(boards []Board, columnPath, title string) int {
	for _, b := range boards {
		for _, c := range b.Columns {
			if c.Path != columnPath {
				continue
			}
			for i, card := range c.Cards {
				if card.Title == title {
					return i
				}
			}
		}
	}
	return 0
}

// reclamp fixes the cursor and horizontal scroll after any state change so
// the focused column is always visible.
func (m *appModel) reclamp() {
	b := m.currentBoard()
	if b == nil {
		m.colIdx, m.cardIdx, m.colScroll = 0, 0, 0
		return
	}
	if m.colIdx < 0 {
		m.colIdx = 0
	}
	if m.colIdx >= len(b.Columns) {
		m.colIdx = len(b.Columns) - 1
	}
	if len(b.Columns) == 0 {
		m.colScroll, m.cardIdx = 0, 0
		return
	}
	avail := m.columnsAvailWidth()
	visible := m.visibleCols(avail)
	remaining := len(b.Columns) - m.colScroll
	if visible > remaining {
		visible = remaining
	}
	if visible < minVisibleCols {
		visible = minVisibleCols
	}
	maxScroll := len(b.Columns) - visible
	if maxScroll < 0 {
		maxScroll = 0
	}
	if m.colScroll > maxScroll {
		m.colScroll = maxScroll
	}
	if m.colIdx < m.colScroll {
		m.colScroll = m.colIdx
	}
	if m.colIdx >= m.colScroll+visible {
		m.colScroll = m.colIdx - visible + 1
	}
	c := m.currentColumn()
	if c == nil {
		m.cardIdx = 0
		return
	}
	if m.cardIdx < 0 {
		m.cardIdx = 0
	}
	if m.cardIdx >= len(c.Cards) {
		m.cardIdx = len(c.Cards) - 1
	}
}

// boardAreaWidth is the content width inside the outer margin.
func (m appModel) boardAreaWidth() int {
	if w := m.width - 2*appMargin; w > 1 {
		return w
	}
	return 1
}

// baseColumnsAvail is the width the columns get with the sidebar taking its
// share (preview not yet accounted for).
func (m appModel) baseColumnsAvail() int {
	w := m.boardAreaWidth()
	if m.sidebar {
		w -= sidebarTotal + panelGap
	}
	if w < 1 {
		return 1
	}
	return w
}

// columnsAvailWidth is how much width the columns get — the board area minus
// the sidebar and the preview panel when they're open.
func (m appModel) columnsAvailWidth() int {
	w := m.baseColumnsAvail()
	if m.preview {
		w -= previewPanelWidth(m.boardAreaWidth()) + panelGap
	}
	if w < 1 {
		return 1
	}
	return w
}

// visibleCols is how many columns fit at the given available width given the
// smallest allowed column.
func (m appModel) visibleCols(avail int) int {
	if avail <= 0 {
		return 1
	}
	per := minColWidth + colBoxOverhead + panelGap
	if per <= 0 {
		return 1
	}
	if v := (avail + panelGap) / per; v >= minVisibleCols {
		return v
	}
	return minVisibleCols
}

// previewPanelWidth sizes the side preview panel: roughly 40% of the board
// area, clamped to a readable range.
func previewPanelWidth(area int) int {
	pw := area * 2 / 5
	if pw < 30 {
		pw = 30
	}
	if pw > 60 {
		pw = 60
	}
	if pw >= area {
		pw = area - 1
	}
	return pw
}

// columnInnerWidth turns "visible" columns sharing `avail` into the inner
// content width of each (subtracting border + padding overhead).
func columnInnerWidth(avail, visible int) int {
	colTotal := (avail - (visible-1)*panelGap) / visible
	inner := colTotal - colBoxOverhead
	if inner < 10 {
		inner = 10
	}
	return inner
}

func (m appModel) View() string {
	switch m.mode {
	case modeInput:
		return m.renderInputModal()
	case modeConfirm:
		return m.renderConfirm()
	case modeLogs:
		return m.renderLogs()
	}

	innerW := m.boardAreaWidth()
	innerH := m.height - 2*appMargin
	if innerH < 1 {
		innerH = 1
	}

	header := theme.Header(innerW).Render("TabelaKanban — " + m.currentBoardName())

	bodyHeight := innerH - headerLines - noticeLines - footerLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	body := m.renderBoard(bodyHeight, innerW)
	notice := m.renderNotice(innerW)
	footer := tuiui.NewFooter(appKeymap...).Render(innerW, theme)

	out := lipgloss.JoinVertical(lipgloss.Left, header, body, notice, footer)
	if m.helpModal.Visible() {
		return m.helpModal.View(theme)
	}
	return out
}

func (m appModel) renderBoard(bodyHeight, innerW int) string {
	b := m.currentBoard()
	if b == nil {
		content := theme.Title().Render("TabelaKanban") + "\n\n" + theme.Dim().Render("nenhum board ainda — pressione B para criar um")
		return m.renderEmptyState(bodyHeight, innerW, content)
	}
	if len(b.Columns) == 0 {
		content := theme.Title().Render(b.Name) + "\n\n" + theme.Dim().Render("sem colunas — pressione N para criar uma")
		return m.renderEmptyState(bodyHeight, innerW, content)
	}

	avail := m.columnsAvailWidth()
	showPreview := m.preview && avail > minColWidth
	if !showPreview {
		avail = m.baseColumnsAvail()
	}
	visible := m.visibleCols(avail)
	remaining := len(b.Columns) - m.colScroll
	if visible > remaining {
		visible = remaining
	}
	if visible < minVisibleCols {
		visible = minVisibleCols
	}
	innerW = columnInnerWidth(avail, visible)

	colHeight := bodyHeight
	innerColHeight := colHeight - colBoxOverhead
	if innerColHeight < cardLines*minVisibleRows {
		innerColHeight = cardLines * minVisibleRows
	}
	maxCards := innerColHeight / cardLines

	columns := b.Columns
	flat := make([]string, 0, visible*2)
	for i := m.colScroll; i < len(columns) && len(flat) < visible*2-1; i++ {
		if len(flat) > 0 {
			flat = append(flat, strings.Repeat(" ", panelGap))
		}
		flat = append(flat, m.renderColumn(columns[i], i == m.colIdx, innerW, maxCards, innerColHeight))
	}
	columnsBox := lipgloss.JoinHorizontal(lipgloss.Top, flat...)

	if m.sidebar {
		columnsBox = lipgloss.JoinHorizontal(lipgloss.Top,
			m.renderSidebar(bodyHeight, sidebarInnerWidth), strings.Repeat(" ", panelGap), columnsBox)
	}

	if showPreview {
		var card *Card
		if c := m.currentCard(); c != nil {
			card = c
		}
		pw := previewPanelWidth(m.boardAreaWidth())
		columnsBox = lipgloss.JoinHorizontal(lipgloss.Top, columnsBox, strings.Repeat(" ", panelGap), m.renderPreview(card, pw, bodyHeight))
	}

	return columnsBox
}

// renderEmptyState fills the whole body height so the footer stays anchored
// at the bottom instead of floating up right under the message.
func (m appModel) renderEmptyState(bodyHeight, innerW int, content string) string {
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	if m.sidebar {
		content = padToHeight(content, bodyHeight)
		left := m.renderSidebar(bodyHeight, sidebarInnerWidth)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, strings.Repeat(" ", panelGap),
			theme.Panel(false).Render(padLines(content, innerW-4-sidebarTotal-panelGap)))
	}
	content = padToHeight(content, bodyHeight)
	return theme.Panel(false).Render(padLines(content, innerW-4))
}

func (m appModel) renderColumn(col Column, focused bool, innerW, maxCards, contentHeight int) string {
	title := fmt.Sprintf("%s (%d)", col.Name, len(col.Cards))
	// Vertical scroll: if the focused column has more cards than fit, keep
	// the cursor visible by offsetting the slice.
	scroll := 0
	if focused && m.cardIdx >= maxCards {
		scroll = m.cardIdx - maxCards + 1
	}
	end := scroll + maxCards
	if end > len(col.Cards) {
		end = len(col.Cards)
	}
	var blocks []string
	for i := scroll; i < end; i++ {
		card := col.Cards[i]
		selected := focused && i == m.cardIdx
		blocks = append(blocks, renderCardBlock(card, selected, innerW))
	}
	content := strings.Join(blocks, "\n")
	// Pad the leftover partial card-row so the panel's bottom border lands
	// exactly on the body's last line.
	content = padToHeight(content, contentHeight)
	return theme.Panel(focused).Render(padLines(theme.Title().Render(title)+"\n\n"+content, innerW))
}

// renderCardBlock draws one 3-line card: title, preview, spacer. The spacer
// line gives each card a clear visual boundary and carries the due date when
// set (red when overdue). Text is padded BEFORE the style is applied so the
// background fills the whole block (a rectangle, not a staircase around the
// text).
func renderCardBlock(card Card, selected bool, width int) string {
	var titleStyle, previewStyle, spacerStyle lipgloss.Style
	if selected {
		titleStyle = lipgloss.NewStyle().Foreground(colBase).Background(colPrimary).Bold(true)
		previewStyle = lipgloss.NewStyle().Foreground(colBase).Background(colPrimary)
		spacerStyle = lipgloss.NewStyle().Background(colPrimary)
	} else {
		titleStyle = lipgloss.NewStyle().Foreground(colText).Background(colBase).Bold(true)
		previewStyle = lipgloss.NewStyle().Foreground(colOverlay0).Background(colBase)
		spacerStyle = lipgloss.NewStyle().Background(colBase)
	}
	preview := cardPreview(card.Body)
	if preview == "" {
		preview = " "
	}

	spacerText := " "
	if !card.Due.IsZero() {
		dueFg := lipgloss.Color(colOverlay0)
		if selected {
			dueFg = colBase
		} else if overdue := card.Due.Before(time.Now().Truncate(24 * time.Hour)); overdue {
			dueFg = colRed
		}
		spacerText = " · " + card.Due.Format("02/01")
		spacerStyle = spacerStyle.Foreground(dueFg)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		titleStyle.Render(padLines("▸ "+card.Title, width)),
		previewStyle.Render(padLines("  "+preview, width)),
		spacerStyle.Render(padLines(spacerText, width)),
	)
}

// cardPreview is the first meaningful line of a card body, for the dim line
// under the title. It skips the opening H1 (which usually repeats the title)
// and markdown markers.
func cardPreview(body string) string {
	lines := strings.Split(body, "\n")
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if i == 0 && strings.HasPrefix(t, "#") {
			continue
		}
		t = strings.TrimLeft(t, "#>-* ")
		t = strings.TrimSpace(strings.ReplaceAll(t, "**", ""))
		if t != "" {
			return t
		}
	}
	return ""
}

// renderSidebar is the collapsible board column on the left. Focused board
// gets a full-width accent highlight (pad-then-style, same as cards).
func (m appModel) renderSidebar(height, innerWidth int) string {
	title := "boards"
	var lines []string
	if len(m.boards) == 0 {
		lines = append(lines, theme.Dim().Render(padLines("nenhum board", innerWidth)))
	}
	for i, b := range m.boards {
		if i == m.boardIdx {
			style := lipgloss.NewStyle().Foreground(colBase).Background(colPrimary).Bold(true)
			lines = append(lines, style.Render(padLines(" "+b.Name, innerWidth)))
		} else {
			style := lipgloss.NewStyle().Foreground(colText).Background(colBase)
			lines = append(lines, style.Render(padLines(" "+b.Name, innerWidth)))
		}
	}
	content := strings.Join(lines, "\n")
	content = padToHeight(content, height-colBoxOverhead)
	return theme.Panel(m.sidebarFocused).Render(padLines(theme.Title().Render(title)+"\n\n"+content, innerWidth))
}

// renderPreview is the side panel showing the selected card's markdown
// rendered as plain text.
func (m appModel) renderPreview(card *Card, width, height int) string {
	var content string
	if card == nil {
		content = theme.Dim().Render("sem card selecionado")
	} else {
		content = theme.Title().Render(card.Title) + "\n\n" + wrapText(stripMarkdown(card.Body), width-4)
	}
	// The preview's title lives inside the content, so only the border (2
	// lines) is overhead — unlike columns/sidebar which add title+blank
	// externally. Pad so the panel lands exactly on the body's last line.
	content = padToHeight(content, height-2)
	return theme.Panel(true).Render(padLines(content, width-4))
}

// stripMarkdown drops the opening H1 (it duplicates the title) and leading
// markdown markers, leaving plain text to wrap in the preview panel.
func stripMarkdown(body string) string {
	lines := strings.Split(body, "\n")
	var out []string
	for i, line := range lines {
		t := strings.TrimSpace(line)
		if i == 0 && strings.HasPrefix(t, "#") {
			continue
		}
		t = strings.TrimLeft(t, "#>-* ")
		t = strings.TrimSpace(strings.ReplaceAll(t, "**", ""))
		out = append(out, t)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return strings.Join(out, "\n")
}

// renderNotice is the reserved line between the board and the footer where
// the latest notification shows. It's always present so the layout never
// shifts when a notification appears or clears.
func (m appModel) renderNotice(width int) string {
	if m.notice == "" {
		return strings.Repeat(" ", width)
	}
	style := lipgloss.NewStyle().Foreground(colSubtext0)
	return style.Render(padLines("  "+m.notice, width))
}

// renderInputModal shows the huh form (djobs-style) as a centered modal.
func (m appModel) renderInputModal() string {
	width, height := m.width, m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	var label string
	switch m.inputKind {
	case inputNewCard:
		label = "novo card"
	case inputNewColumn:
		label = "nova coluna"
	case inputRenameCard:
		label = "renomear card"
	case inputRenameColumn:
		label = "renomear coluna"
	case inputNewBoard:
		label = "novo board"
	case inputDue:
		label = "due date"
	}
	var body string
	if m.form != nil {
		body = m.form.View()
	}
	box := theme.Modal().Render(theme.Title().Render(label) + "\n\n" + body)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m appModel) renderConfirm() string {
	width, height := m.width, m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	var text string
	if m.confirmCol != nil {
		text = theme.Title().Render("apagar coluna") + "\n\n" +
			theme.Dim().Render(fmt.Sprintf("Apagar a coluna '%s'? Isso remove todos os cards dela.", m.confirmCol.Name)) + "\n\n" +
			theme.Success().Render("y") + theme.Dim().Render(" apagar · ") + theme.Error().Render("n") + theme.Dim().Render(" cancelar")
	} else {
		text = theme.Title().Render("apagar card") + "\n\n" +
			theme.Dim().Render(fmt.Sprintf("Apagar '%s'? Isso remove o arquivo.", m.confirmCard.Title)) + "\n\n" +
			theme.Success().Render("y") + theme.Dim().Render(" apagar · ") + theme.Error().Render("n") + theme.Dim().Render(" cancelar")
	}
	box := theme.Modal().Render(text)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

// renderLogs is a full-page activity log: same chrome as the board, with the
// panel showing rich multi-line entries (newest first).
func (m appModel) renderLogs() string {
	innerW := m.boardAreaWidth()
	innerH := m.height - 2*appMargin
	if innerH < 1 {
		innerH = 1
	}

	header := theme.Header(innerW).Render("TabelaKanban — log de atividades")

	bodyHeight := innerH - headerLines - footerLines
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var lines []string
	for i := len(m.log) - 1; i >= 0; i-- {
		lines = append(lines, m.log[i].rich()...)
	}
	if len(lines) == 0 {
		lines = append(lines, theme.Dim().Render("nada registrado ainda"))
	}
	content := strings.Join(lines, "\n")
	content = padToHeight(content, bodyHeight-colBoxOverhead)
	box := theme.Panel(true).Render(padLines(theme.Title().Render("log de atividades")+"\n\n"+content, innerW-4))

	footer := theme.Footer(innerW).Render("g/h/q fecha")
	return lipgloss.JoinVertical(lipgloss.Left, header, box, footer)
}

type editorFinishedMsg struct{}

// openEditor suspends the TUI to run $EDITOR (falling back to nvim) against
// a card's .md file, the same "jump straight into it" shortcut the other
// TUIs use.
func openEditor(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nvim"
	}
	cmd := exec.Command(editor, path)
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return editorFinishedMsg{} })
}
