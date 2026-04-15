package main

import "testing"

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
