package main

import "time"

type sessionEventKind string

const (
	sessionEventKindPrompt  sessionEventKind = "prompt"
	sessionEventKindSystem   sessionEventKind = "system"
	sessionEventKindRuntime  sessionEventKind = "runtime"
	sessionEventKindFailure  sessionEventKind = "failure"
	sessionEventKindHistory  sessionEventKind = "history"
	sessionEventKindMemory   sessionEventKind = "memory"
)

type sessionEvent struct {
	Kind      sessionEventKind
	Message   string
	CreatedAt time.Time
}
