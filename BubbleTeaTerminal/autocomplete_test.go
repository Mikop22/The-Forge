package main

import (
	"strings"
	"testing"
)

func TestAutocompleteFiltersOnPrefix(t *testing.T) {
	items := filterAutocomplete("/fo")
	if len(items) == 0 {
		t.Fatal("filterAutocomplete(\"/fo\") = empty, want at least /forge")
	}
	if items[0].Slash != "/forge" {
		t.Fatalf("first match = %q, want /forge", items[0].Slash)
	}
}

func TestAutocompleteReturnsAllOnSlashOnly(t *testing.T) {
	items := filterAutocomplete("/")
	if len(items) < 8 {
		t.Fatalf("filterAutocomplete(\"/\") = %d items, want all commands (>=8)", len(items))
	}
}

func TestAutocompleteReturnsNilOnNonSlash(t *testing.T) {
	items := filterAutocomplete("radiant spear")
	if items != nil {
		t.Fatalf("filterAutocomplete(non-slash) = %v, want nil", items)
	}
}

func TestAutocompleteReturnsNilOnEmptyInput(t *testing.T) {
	items := filterAutocomplete("")
	if items != nil {
		t.Fatalf("filterAutocomplete(\"\") = %v, want nil", items)
	}
}

func TestAutocompleteDrawerRendersWhenSlashTyped(t *testing.T) {
	t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
	m := initialModel()
	m.commandInput.SetValue("/")
	m.width = 120
	m.height = 40

	got := m.View()
	if !strings.Contains(got, "/forge") {
		t.Fatalf("view = %q, want autocomplete drawer showing /forge when '/' typed", got)
	}
	if !strings.Contains(got, "Generate a new item") {
		t.Fatalf("view = %q, want command description in autocomplete drawer", got)
	}
}

func TestAutocompleteDrawerHiddenWhenNoMatch(t *testing.T) {
	t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
	m := initialModel()
	m.commandInput.SetValue("/zzz")

	got := m.View()
	if strings.Contains(got, "Generate a new item") {
		t.Fatalf("view = %q, want autocomplete hidden when no matches", got)
	}
}

func TestAutocompleteDrawerHighlightsSelectedRow(t *testing.T) {
	t.Setenv("FORGE_MOD_SOURCES_DIR", t.TempDir())
	m := initialModel()
	m.commandInput.SetValue("/")
	m.autocompleteIndex = 1
	m.width = 120

	got := m.View()
	if !strings.Contains(got, "/variants") {
		t.Fatalf("view = %q, want /variants in drawer", got)
	}
}
