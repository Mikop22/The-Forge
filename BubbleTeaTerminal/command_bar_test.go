package main

import "testing"

func TestParseWorkshopCommandVariants(t *testing.T) {
	cmd := parseWorkshopCommand("/variants make the projectile feel heavier")
	if cmd.Name != "variants" {
		t.Fatalf("name = %q, want variants", cmd.Name)
	}
	if cmd.Arg != "make the projectile feel heavier" {
		t.Fatalf("arg = %q", cmd.Arg)
	}
}

func TestBuildWorkshopRequestPayloadUsesBenchAndSession(t *testing.T) {
	payload := buildWorkshopRequestPayload(
		workshopCommand{Name: "variants", Arg: "make it heavier"},
		"bench-storm-brand",
		"storm-brand",
	)

	if got := payload["action"]; got != "variants" {
		t.Fatalf("action = %#v, want variants", got)
	}
	if got := payload["session_id"]; got != "bench-storm-brand" {
		t.Fatalf("session_id = %#v", got)
	}
	if got := payload["bench_item_id"]; got != "storm-brand" {
		t.Fatalf("bench_item_id = %#v", got)
	}
}
