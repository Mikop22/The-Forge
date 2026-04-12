package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"theforge/internal/ipc"
)

func TestStagingViewShowsRuntimeBannerFields(t *testing.T) {
	m := initialModel()
	item := craftedItem{
		label:          "Storm Brand",
		contentType:    "Weapon",
		subType:        "Staff",
		spritePath:     "",
		projSpritePath: "",
	}
	m.previewItem = &item
	m.workshop.SetBenchFromCraftedItem(item, map[string]interface{}{})
	m.revealPhase = 3
	m.workshop.Runtime = workshopRuntimeBanner{
		BridgeAlive:      true,
		WorldLoaded:      true,
		LiveItemName:     "Storm Brand",
		LastInjectStatus: "item_injected",
		LastRuntimeNote:  "Ready on bench",
	}

	view := m.stagingView()
	for _, want := range []string{
		"Runtime Online",
		"World Loaded",
		"Live item: Storm Brand",
		"Inject status: item_injected",
		"Ready on bench",
	} {
		if !strings.Contains(view, want) {
			t.Fatalf("stagingView() missing %q\n%s", want, view)
		}
	}
}

func TestApplyWorkshopStatusRefreshesForgeItemName(t *testing.T) {
	m := initialModel()
	m.forgeItemName = "Old Name"
	m.applyWorkshopStatus(ipc.WorkshopStatus{
		SessionID: "bench-storm-brand",
		Bench: ipc.WorkshopBench{
			ItemID: "storm-brand",
			Label:  "Storm Brand",
			Manifest: map[string]interface{}{
				"type":     "Weapon",
				"sub_type": "Staff",
				"stats": map[string]interface{}{
					"damage": 24.0,
				},
			},
		},
	})

	if m.forgeItemName != "Storm Brand" {
		t.Fatalf("forgeItemName = %q, want bench label", m.forgeItemName)
	}
}

func TestAcceptInjectUsesBenchLabelAfterWorkshopStatus(t *testing.T) {
	home := t.TempDir()
	ms := filepath.Join(home, "ModSources")
	t.Setenv("HOME", home)
	t.Setenv("FORGE_MOD_SOURCES_DIR", ms)

	m := initialModel()
	m.previewMode = previewModeActions
	m.applyWorkshopStatus(ipc.WorkshopStatus{
		SessionID: "bench-storm-brand",
		Bench: ipc.WorkshopBench{
			ItemID: "storm-brand",
			Label:  "Storm Brand",
			Manifest: map[string]interface{}{
				"type":     "Weapon",
				"sub_type": "Staff",
				"stats": map[string]interface{}{
					"damage": 24.0,
				},
			},
		},
	})
	m.forgeItemName = "Old Name"

	_, _ = m.updateStaging(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})

	data, err := os.ReadFile(filepath.Join(ms, "forge_inject.json"))
	if err != nil {
		t.Fatalf("read forge_inject.json: %v", err)
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("unmarshal forge_inject.json: %v", err)
	}
	if got := payload["item_name"]; got != "Storm Brand" {
		t.Fatalf("item_name = %#v, want Storm Brand", got)
	}
}

func TestResolveRuntimeBannerPrefersHeartbeatForBridgeAlive(t *testing.T) {
	banner := resolveRuntimeBanner(
		ipc.RuntimeSummary{
			BridgeAlive:      true,
			WorldLoaded:      true,
			LiveItemName:     "Storm Brand",
			LastInjectStatus: "item_injected",
			LastRuntimeNote:  "Ready on bench",
		},
		false,
		"",
		"",
	)

	if banner.BridgeAlive {
		t.Fatal("BridgeAlive = true, want false when heartbeat is absent")
	}
}
