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
	{"/variants", "<describe changes>", "Generate shelf variants from the bench"},
	{"/bench", "<id or number>", "Set a shelf variant as the active bench"},
	{"/try", "", "Reinject the current bench item into Terraria"},
	{"/restore", "baseline | live", "Restore bench to a previous state"},
	{"/status", "", "Show bench label and runtime state"},
	{"/memory", "", "Show pinned memory notes"},
	{"/what-changed", "", "Summarise changes since last bench"},
	{"/clear", "", "Clear the active bench and shelf"},
	{"/help", "", "List all available commands"},
}

// renderAutocompleteDrawer renders a two-column command list below the prompt.
// Returns "" when there are no matches. The bottom-anchored panel layout
// naturally shifts content upward as lines are added here.
func renderAutocompleteDrawer(m model) string {
	entries := filterAutocomplete(m.commandInput.Value())

	// Show "no matches" hint when user has typed a /command that matches nothing.
	if len(entries) == 0 {
		if v := m.commandInput.Value(); strings.HasPrefix(v, "/") && len(v) > 1 {
			return lipgloss.NewStyle().Foreground(colorDim).Render("No matching commands")
		}
		return ""
	}

	idx := m.autocompleteIndex
	if idx < 0 {
		idx = 0
	}
	if idx >= len(entries) {
		idx = len(entries) - 1
	}

	// slashWidth must be >= the widest Slash+ArgHint pair in autocompleteRegistry.
	// Current widest: "/restore baseline | live" = 24 chars. 2 chars of margin.
	const slashWidth = 26
	contentW := shellContentWidth(m)
	descWidth := max(1, contentW-slashWidth)
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
			lines = append(lines, selSlash.Width(slashWidth).Render(name)+selDesc.Width(descWidth).Render(e.Desc))
		} else {
			lines = append(lines, dimStyle.Width(slashWidth).Render(name)+dimStyle.Width(descWidth).Render(e.Desc))
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
