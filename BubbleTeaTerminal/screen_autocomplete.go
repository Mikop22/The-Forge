package main

import "strings"

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
