package main

import (
	"strings"
)

type sessionShellState struct {
	events []sessionEvent
}

func newSessionShellState() sessionShellState {
	return sessionShellState{
		events: make([]sessionEvent, 0, 16),
	}
}

func (s sessionShellState) render(m model, content string) string {
	top := s.renderTopStrip(m)
	feed := s.renderFeedContainer(content)
	command := s.renderCommandBar(m)
	return strings.Join([]string{top, feed, command}, "\n")
}

func (s sessionShellState) renderTopStrip(m model) string {
	statusBits := []string{"Forge Director"}
	if m.bridgeAlive {
		statusBits = append(statusBits, "runtime online")
	} else {
		statusBits = append(statusBits, "runtime offline")
	}
	if m.forgeItemName != "" {
		statusBits = append(statusBits, "bench: "+m.forgeItemName)
	}
	return strings.Join([]string{
		styles.Meta.Render("Top Strip"),
		styles.Body.Render(strings.Join(statusBits, " | ")),
	}, "\n")
}

func (s sessionShellState) renderFeedContainer(content string) string {
	return strings.Join([]string{
		styles.Meta.Render("Feed Container"),
		styles.FrameCalm.Render(content),
	}, "\n")
}

func (s sessionShellState) renderCommandBar(m model) string {
	command := strings.TrimSpace(m.commandInput.Value())
	if command == "" {
		command = m.commandInput.Placeholder
	}
	return strings.Join([]string{
		styles.Meta.Render("Persistent Command Bar"),
		styles.PromptInput.Render(command),
	}, "\n")
}
