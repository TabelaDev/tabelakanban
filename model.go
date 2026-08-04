package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// mode tracks which interaction is on top of the kanban: normal column/card
// navigation, a text input (new card/column/rename/board), a delete
// confirmation, or the board picker.
type mode int

const (
	modeBoard mode = iota
	modeInput
	modeConfirm
	modePicker
)

// inputKind says what the text input is creating/renaming.
type inputKind int

const (
	inputNewCard inputKind = iota
	inputNewColumn
	inputRenameCard
	inputNewBoard
)

const (
	headerLines = 1
	footerLines = 1
	// Column box: 2 border + 1 title + 1 blank = 4 lines of overhead; each
	// card block is 2 lines (title + preview).
	colBoxOverhead = 4
	cardLines      = 2
	colInnerWidth  = 24
	panelGap       = 1
	minVisibleCols = 1
	minVisibleRows = 3
)

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

	mode      mode
	input     textinput.Model
	inputKind inputKind
	// confirmCard is the card awaiting y/n confirmation.
	confirmCard Card

	picker list.Model

	width  int
	height int
	status string
}

func newModel() appModel {
	m := appModel{}
	m.input = textinput.New()
	m.input.CharLimit = 60
	m.rescan()
	return m
}

// rescan reloads boards from disk. It keeps boardIdx/colIdx pinned to the
// same board/column by name when possible, so a refresh (e.g. after the
// editor closes) doesn't jump the cursor.
func (m *appModel) rescan() {
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

	m.status = fmt.Sprintf("%d board(s)", len(boards))
	if len(warnings) > 0 {
		m.status += " — " + strings.Join(warnings, "; ")
	}
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
		m.reclamp()
		return m, nil
	}
	if _, ok := msg.(editorFinishedMsg); ok {
		m.rescan()
		m.reclamp()
		return m, nil
	}

	switch m.mode {
	case modeInput:
		return m.updateInput(msg)
	case modeConfirm:
		return m.updateConfirm(msg)
	case modePicker:
		return m.updatePicker(msg)
	default:
		return m.updateBoard(msg)
	}
}

func (m *appModel) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "ctrl+r":
		m.rescan()
		m.reclamp()
		return m, nil
	case "b":
		m.openPicker()
		return m, nil
	case "n":
		return m.startInput(inputNewCard, "novo card: ", "")
	case "N":
		return m.startInput(inputNewColumn, "nova coluna: ", "")
	case "B":
		return m.startInput(inputNewBoard, "novo board: ", "")
	case "r":
		if card := m.currentCard(); card != nil {
			return m.startInput(inputRenameCard, "renomear: ", card.Title)
		}
	case "d":
		if card := m.currentCard(); card != nil {
			m.confirmCard = *card
			m.mode = modeConfirm
		}
	case "o", "enter":
		if card := m.currentCard(); card != nil {
			return m, openEditor(card.Path)
		}
	case "h", "left":
		if m.colIdx > 0 {
			m.colIdx--
			m.cardIdx = 0
		}
	case "l", "right":
		if c := m.currentBoard(); c != nil && m.colIdx < len(c.Columns)-1 {
			m.colIdx++
			m.cardIdx = 0
		}
	case "H":
		m.moveCardBetween(-1)
	case "L":
		m.moveCardBetween(1)
	case "j", "down":
		if c := m.currentColumn(); c != nil && m.cardIdx < len(c.Cards)-1 {
			m.cardIdx++
		}
	case "k", "up":
		if m.cardIdx > 0 {
			m.cardIdx--
		}
	}
	m.reclamp()
	return m, nil
}

// moveCardBetween moves the selected card to the adjacent column.
func (m *appModel) moveCardBetween(delta int) {
	card := m.currentCard()
	col := m.currentColumn()
	b := m.currentBoard()
	if card == nil || col == nil || b == nil {
		return
	}
	next := m.colIdx + delta
	if next < 0 || next >= len(b.Columns) {
		m.status = "não há coluna nessa direção"
		return
	}
	moved, err := moveCard(*card, b.Columns[next])
	if err != nil {
		m.status = "erro movendo card: " + err.Error()
		return
	}
	toName := b.Columns[next].Name
	m.rescan()
	m.colIdx = next
	m.cardIdx = indexOfCard(m.boards, b.Columns[next].Path, moved.Title)
	m.reclamp()
	m.status = fmt.Sprintf("card '%s' movido para %s", moved.Title, toName)
}

func (m *appModel) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch keyMsg.String() {
	case "y":
		if err := deleteCard(m.confirmCard); err != nil {
			m.status = "erro apagando: " + err.Error()
		} else {
			m.status = "card apagado"
		}
		m.mode = modeBoard
		m.rescan()
		m.reclamp()
	case "n", "esc", "q":
		m.mode = modeBoard
	}
	return m, nil
}

func (m *appModel) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.picker, cmd = m.picker.Update(msg)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			if item, ok := m.picker.SelectedItem().(boardItem); ok {
				m.boardIdx = indexOfBoard(m.boards, item.title)
				m.mode = modeBoard
				m.colIdx = 0
				m.cardIdx = 0
				m.reclamp()
			}
			return m, nil
		case "esc", "q", "ctrl+c":
			m.mode = modeBoard
			return m, nil
		}
	}
	return m, cmd
}

func (m *appModel) startInput(kind inputKind, prompt, value string) (tea.Model, tea.Cmd) {
	m.inputKind = kind
	m.input.SetValue(value)
	m.input.Prompt = prompt
	m.input.Focus()
	m.mode = modeInput
	return m, textinput.Blink
}

func (m *appModel) updateInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "enter":
			m.commitInput()
			m.mode = modeBoard
			return m, nil
		case "esc":
			m.input.Blur()
			m.mode = modeBoard
			return m, nil
		}
	}
	return m, cmd
}

func (m *appModel) commitInput() {
	title := strings.TrimSpace(m.input.Value())
	m.input.Blur()
	switch m.inputKind {
	case inputNewCard:
		col := m.currentColumn()
		if col == nil {
			m.status = "nenhuma coluna focada"
			return
		}
		card, err := createCard(*col, title)
		if err != nil {
			m.status = "erro criando card: " + err.Error()
			return
		}
		m.status = "card criado"
		m.rescan()
		m.cardIdx = indexOfCard(m.boards, col.Path, card.Title)
	case inputNewColumn:
		b := m.currentBoard()
		if b == nil {
			m.status = "nenhum board"
			return
		}
		if err := createColumn(*b, title); err != nil {
			m.status = "erro criando coluna: " + err.Error()
			return
		}
		m.status = "coluna criada"
		m.rescan()
		m.colIdx = indexOfColumn(m.currentBoard(), title)
	case inputRenameCard:
		card := m.currentCard()
		if card == nil {
			m.status = "nenhum card"
			return
		}
		renamed, err := renameCard(*card, title)
		if err != nil {
			m.status = "erro renomeando: " + err.Error()
			return
		}
		m.status = "renomeado"
		m.rescan()
		m.cardIdx = indexOfCard(m.boards, m.currentColumnPath(), renamed.Title)
	case inputNewBoard:
		dir, err := createBoard(m.roots, title)
		if err != nil {
			m.status = "erro criando board: " + err.Error()
			return
		}
		m.status = fmt.Sprintf("board criado em %s", dir)
		m.rescan()
		m.boardIdx = indexOfBoard(m.boards, title)
	}
	m.reclamp()
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

func (m *appModel) openPicker() {
	if len(m.boards) == 0 {
		m.status = "nenhum board — crie um com B"
		return
	}
	items := make([]list.Item, 0, len(m.boards))
	for _, b := range m.boards {
		items = append(items, boardItem{title: b.Name, cols: len(b.Columns)})
	}
	m.picker = list.New(items, list.NewDefaultDelegate(), 0, 0)
	m.picker.Title = "boards"
	m.picker.SetShowStatusBar(false)
	m.picker.SetFilteringEnabled(false)
	m.mode = modePicker
}

type boardItem struct {
	title string
	cols  int
}

func (i boardItem) Title() string       { return i.title }
func (i boardItem) Description() string { return fmt.Sprintf("%d colunas", i.cols) }
func (i boardItem) FilterValue() string { return i.title }

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
	visible := m.visibleCols()
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

func (m *appModel) visibleCols() int {
	if m.width == 0 {
		return 1
	}
	colTotal := colInnerWidth + 4 + panelGap
	if colTotal <= 0 {
		return 1
	}
	if v := m.width / colTotal; v >= minVisibleCols {
		return v
	}
	return minVisibleCols
}

func (m appModel) View() string {
	if m.mode == modePicker {
		return m.renderPicker()
	}
	if m.mode == modeConfirm {
		return m.renderConfirm()
	}

	header := headerStyle(m.width).Render("TabelaKanban — " + m.currentBoardName())

	var inputLine string
	if m.mode == modeInput {
		inputLine = m.renderInput()
	}

	bodyHeight := m.height - headerLines - footerLines
	if inputLine != "" {
		bodyHeight--
	}
	body := m.renderBoard(bodyHeight)

	footer := footerStyle(m.width).Render(m.footerText())

	return lipgloss.JoinVertical(lipgloss.Left, header, inputLine, body, footer)
}

func (m appModel) renderBoard(bodyHeight int) string {
	b := m.currentBoard()
	if b == nil {
		return panelStyle(false).Render(padLines(
			titleStyle().Render("TabelaKanban")+"\n\n"+dimStyle().Render("nenhum board ainda — pressione B para criar um"), m.width-4,
		))
	}
	if len(b.Columns) == 0 {
		return panelStyle(false).Render(padLines(
			titleStyle().Render(b.Name)+"\n\n"+dimStyle().Render("sem colunas — pressione N para criar uma"), m.width-4,
		))
	}

	visible := m.visibleCols()
	colHeight := bodyHeight
	innerColHeight := colHeight - colBoxOverhead
	if innerColHeight < cardLines*minVisibleRows {
		innerColHeight = cardLines * minVisibleRows
	}
	maxCards := innerColHeight / cardLines

	var boxes []string
	columns := b.Columns
	for i := m.colScroll; i < len(columns) && len(boxes) < visible; i++ {
		boxes = append(boxes, m.renderColumn(columns[i], i == m.colIdx, maxCards))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
}

func (m appModel) renderColumn(col Column, focused bool, maxCards int) string {
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
	var lines []string
	for i := scroll; i < end; i++ {
		card := col.Cards[i]
		selected := focused && i == m.cardIdx
		preview := cardPreview(card.Body)
		block := cardStyle(selected).Render(padLines(
			" "+card.Title+"\n "+preview, colInnerWidth,
		))
		lines = append(lines, block)
	}
	// Pad remaining card slots so all columns render the same height.
	for len(lines) < maxCards {
		lines = append(lines, padLines(" \n ", colInnerWidth))
	}
	content := strings.Join(lines, "\n")
	return panelStyle(focused).Render(padLines(titleStyle().Render(title)+"\n\n"+content, colInnerWidth))
}

// cardPreview is the first meaningful line of a card body, for the dim line
// under the title. Skips markdown heading markers and blanks.
func cardPreview(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(strings.TrimLeft(line, "#*- "))
		if line != "" {
			return line
		}
	}
	return ""
}

func (m appModel) renderInput() string {
	prompt := "  " + m.input.Prompt
	return dimStyle().Render(prompt) + m.input.View()
}

func (m appModel) renderPicker() string {
	width, height := m.width, m.height
	if width <= 0 {
		width = 100
	}
	if height <= 0 {
		height = 30
	}
	m.picker.SetSize(width-4, height-4)
	box := theme.Modal().Render(m.picker.View())
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
	text := titleStyle().Render("apagar card") + "\n\n" +
		dimStyle().Render(fmt.Sprintf("Apagar '%s'? Isso remove o arquivo.", m.confirmCard.Title)) + "\n\n" +
		theme.Success().Render("y") + dimStyle().Render(" apagar · ") + theme.Error().Render("n") + dimStyle().Render(" cancelar")
	box := theme.Modal().Render(text)
	return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, box)
}

func (m appModel) footerText() string {
	help := "h/l coluna · j/k card · H/L mover · n card · N coluna · B board · r renomear · d apagar · o/enter editar · b picker · ctrl+r recarregar · q sair"
	footer := help
	if m.status != "" {
		footer = m.status + "   " + help
	}
	if avail := m.width - 4; avail > 0 {
		footer = strings.TrimRight(padLines(footer, avail), " ")
	}
	return footer
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
