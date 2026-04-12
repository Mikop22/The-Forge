package main

import (
	"strings"

	"theforge/internal/ipc"
)

type workshopBench struct {
	ItemID          string
	Label           string
	Manifest        map[string]interface{}
	SpritePath      string
	ProjectilePath  string
	Stats           itemStats
	ContentType     string
	SubType         string
	CraftingStation string
}

type workshopVariant struct {
	VariantID      string
	Label          string
	Rationale      string
	ChangeSummary  string
	Manifest       map[string]interface{}
	SpritePath     string
	ProjectilePath string
}

type workshopRuntimeBanner struct {
	BridgeAlive      bool
	WorldLoaded      bool
	LiveItemName     string
	LastInjectStatus string
	LastRuntimeNote  string
}

type workshopState struct {
	SessionID string
	Bench     workshopBench
	Shelf     []workshopVariant
	Runtime   workshopRuntimeBanner
}

func newWorkshopState() workshopState {
	return workshopState{
		Shelf: []workshopVariant{},
	}
}

func workshopIDFromLabel(label string) string {
	cleaned := strings.TrimSpace(strings.ToLower(label))
	if cleaned == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range cleaned {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

func workshopBenchFromCraftedItem(item craftedItem) workshopBench {
	return workshopBench{
		ItemID:          workshopIDFromLabel(item.label),
		Label:           item.label,
		SpritePath:      item.spritePath,
		ProjectilePath:  item.projSpritePath,
		Stats:           item.stats,
		ContentType:     item.contentType,
		SubType:         item.subType,
		CraftingStation: item.craftingStation,
	}
}

func (ws *workshopState) SetBenchFromCraftedItem(item craftedItem, manifest map[string]interface{}) {
	ws.Bench = workshopBenchFromCraftedItem(item)
	ws.Bench.Manifest = manifest
	if ws.Bench.ItemID != "" {
		ws.SessionID = "bench-" + ws.Bench.ItemID
	}
}

func workshopBenchFromStatus(bench ipc.WorkshopBench) workshopBench {
	result := workshopBench{
		ItemID:         bench.ItemID,
		Label:          bench.Label,
		Manifest:       bench.Manifest,
		SpritePath:     bench.SpritePath,
		ProjectilePath: bench.ProjectileSpritePath,
		Stats:          extractItemStats(bench.Manifest),
		ContentType:    manifestString(bench.Manifest, "type", "content_type"),
		SubType:        manifestString(bench.Manifest, "sub_type"),
		CraftingStation: manifestString(
			bench.Manifest,
			"crafting_station",
		),
	}
	return result
}

func (ws *workshopState) ApplyStatus(status ipc.WorkshopStatus) {
	ws.SessionID = status.SessionID
	ws.Bench = workshopBenchFromStatus(status.Bench)
	ws.Shelf = make([]workshopVariant, 0, len(status.Shelf))
	for _, variant := range status.Shelf {
		ws.Shelf = append(ws.Shelf, workshopVariant{
			VariantID:      variant.VariantID,
			Label:          variant.Label,
			Rationale:      variant.Rationale,
			ChangeSummary:  variant.ChangeSummary,
			Manifest:       variant.Manifest,
			SpritePath:     variant.SpritePath,
			ProjectilePath: variant.ProjectileSpritePath,
		})
	}
}

func craftedItemFromWorkshopBench(bench workshopBench) craftedItem {
	return craftedItem{
		label:           bench.Label,
		contentType:     bench.ContentType,
		subType:         bench.SubType,
		craftingStation: bench.CraftingStation,
		stats:           extractItemStats(bench.Manifest),
		spritePath:      bench.SpritePath,
		projSpritePath:  bench.ProjectilePath,
	}
}

func manifestString(manifest map[string]interface{}, keys ...string) string {
	if manifest == nil {
		return ""
	}
	for _, key := range keys {
		if value, ok := manifest[key].(string); ok && value != "" {
			return value
		}
	}
	return ""
}
