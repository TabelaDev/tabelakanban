package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
)

// newInputForm builds the djobs-style modal form for the given input kind.
// The value is the pre-fill (e.g. the current card title when renaming).
func newInputForm(kind inputKind, value string) *huh.Form {
	switch kind {
	case inputNewCard:
		title, due := value, ""
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Key("title").Title("Título").Placeholder("ex.: configurar CI").Value(&title),
				huh.NewInput().Key("due").Title("Due date (DD/MM)").Placeholder("vazio = sem data").Value(&due).Validate(validateDue),
			),
		).WithTheme(huh.ThemeCatppuccin()).WithShowHelp(true).WithWidth(40)
	case inputRenameCard:
		title := value
		return singleFieldForm("Título do card", &title)
	case inputNewColumn:
		name := value
		return singleFieldForm("Nome da coluna", &name)
	case inputRenameColumn:
		name := value
		return singleFieldForm("Nome da coluna", &name)
	case inputNewBoard:
		name := value
		return singleFieldForm("Nome do board", &name)
	case inputDue:
		due := value
		return huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Key("due").Title("Due date (DD/MM)").Placeholder("vazio = limpar").Value(&due).Validate(validateDue),
			),
		).WithTheme(huh.ThemeCatppuccin()).WithShowHelp(true).WithWidth(40)
	}
	return nil
}

func singleFieldForm(title string, value *string) *huh.Form {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Key("title").Title(title).Value(value),
		),
	).WithTheme(huh.ThemeCatppuccin()).WithShowHelp(true).WithWidth(40)
}

// validateDue accepts "DD/MM" or an empty string (no due date).
func validateDue(s string) error {
	_, err := parseDueDDMM(s)
	return err
}

// parseDueDDMM parses a "DD/MM" due date. Empty string means no due date. A
// date that already passed this year rolls over to next year.
func parseDueDDMM(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, nil
	}
	var day, month int
	if _, err := fmt.Sscanf(s, "%d/%d", &day, &month); err != nil {
		return time.Time{}, fmt.Errorf("use o formato DD/MM")
	}
	if day < 1 || day > 31 || month < 1 || month > 12 {
		return time.Time{}, fmt.Errorf("data inválida")
	}
	now := time.Now()
	t := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, now.Location())
	if t.Before(now) {
		t = t.AddDate(1, 0, 0)
	}
	return t, nil
}

// dueDDMM formats a due date as "DD/MM" (empty when unset), for pre-filling
// the due form.
func dueDDMM(due time.Time) string {
	if due.IsZero() {
		return ""
	}
	return due.Format("02/01")
}
