package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type autocompleteEntry struct {
	Slash   string
	ArgHint string
	Desc    string
}

var autocompleteRegistry = []autocompleteEntry{
	{"/forge", "<prompt>", "Generate a new item from scratch"},
	{"/variants", "<direction>", "Generate shelf variants from the bench"},
	{"/bench", "<id or number>", "Set a shelf variant as the active bench"},
	{"/try", "", "Reinject the current bench item into Terraria"},
	{"/restore", "baseline | live", "Restore bench to a previous state"},
	{"/status", "", "Show bench label and runtime state"},
	{"/memory", "", "Show pinned memory notes"},
	{"/what-changed", "", "Summarise changes since last bench"},
	{"/help", "", "List all available commands"},
}

// renderAutocompleteDrawer renders a two-column command list below the prompt.
// Returns "" when there are no matches. The bottom-anchored panel layout
// naturally shifts content upward as lines are added here.
func renderAutocompleteDrawer(m model) string {
	entries := filterAutocomplete(m.commandInput.Value())
	if len(entries) == 0 {
		return ""
	}

	idx := m.autocompleteIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= len(entries) {
		idx = len(entries) - 1
	}

	const slashWidth = 20
	dimStyle := lipgloss.NewStyle().Foreground(colorDim)
	selSlash := lipgloss.NewStyle().Foreground(colorRune).Bold(true)
	selDesc := lipgloss.NewStyle().Foreground(colorText)

	lines := make([]string, 0, len(entries))
	for i, e := range entries {
		name := e.Slash
		if e.ArgHint != "" {
			name += " " + e.ArgHint
		}
		if i == idx {
			lines = append(lines, selSlash.Width(slashWidth).Render(name)+selDesc.Render(e.Desc))
		} else {
			lines = append(lines, dimStyle.Width(slashWidth).Render(name)+dimStyle.Render(e.Desc))
		}
	}
	return strings.Join(lines, "\n")
}

// filterAutocomplete returns matching entries for the given raw input value.
// Returns nil when input is empty or does not start with "/".
func filterAutocomplete(input string) []autocompleteEntry {
	if input == "" || !strings.HasPrefix(input, "/") {
		return nil
	}
	lower := strings.ToLower(input)
	var out []autocompleteEntry
	for _, e := range autocompleteRegistry {
		if strings.HasPrefix(e.Slash, lower) {
			out = append(out, e)
		}
	}
	return out
}
