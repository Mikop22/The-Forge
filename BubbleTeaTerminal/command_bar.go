package main

import "strings"

type workshopCommand struct {
	Name string
	Arg  string
}

func parseWorkshopCommand(input string) workshopCommand {
	text := strings.TrimSpace(input)
	if text == "" {
		return workshopCommand{}
	}
	if !strings.HasPrefix(text, "/") {
		return workshopCommand{Name: "variants", Arg: text}
	}

	trimmed := strings.TrimSpace(strings.TrimPrefix(text, "/"))
	if trimmed == "" {
		return workshopCommand{}
	}
	parts := strings.Fields(trimmed)
	name := strings.ToLower(parts[0])
	arg := ""
	if len(parts) > 1 {
		arg = strings.TrimSpace(trimmed[len(parts[0]):])
	}
	return workshopCommand{Name: name, Arg: arg}
}

func buildWorkshopRequestPayload(cmd workshopCommand, sessionID, benchItemID string) map[string]interface{} {
	payload := map[string]interface{}{
		"action":        cmd.Name,
		"session_id":    sessionID,
		"bench_item_id": benchItemID,
	}

	switch cmd.Name {
	case "variants":
		payload["directive"] = cmd.Arg
	case "bench":
		payload["variant_id"] = cmd.Arg
	case "restore":
		target := strings.ToLower(strings.TrimSpace(cmd.Arg))
		if target == "live" {
			target = "last_live"
		}
		if target == "" {
			target = "baseline"
		}
		payload["restore_target"] = target
	case "try":
		// No extra fields for V1.
	default:
		payload["directive"] = cmd.Arg
	}

	return payload
}
