package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestParseDotEnvStripsCommentsOutsideQuotes(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), ".env")
	content := "" +
		"OPENAI_API_KEY=\"sk-test\" # local key\n" +
		"PLAIN=value # trailing comment\n" +
		"HASHED=\"value # keep this\"\n" +
		"SINGLE='two words' # note\n"
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("write env: %v", err)
	}

	got := parseDotEnv(envPath)
	want := []string{
		"OPENAI_API_KEY=sk-test",
		"PLAIN=value",
		"HASHED=value # keep this",
		"SINGLE=two words",
	}

	if len(got) != len(want) {
		t.Fatalf("parseDotEnv() returned %d pairs, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseDotEnv()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestReadOrchestratorHeartbeatUsesDistinctFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir := modSourcesDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("mkdir mod sources: %v", err)
	}

	bridgeHeartbeat := []byte(`{"status":"listening","pid":` + strconv.Itoa(os.Getpid()) + `}`)
	if err := os.WriteFile(filepath.Join(dir, "forge_connector_alive.json"), bridgeHeartbeat, 0644); err != nil {
		t.Fatalf("write bridge heartbeat: %v", err)
	}

	if !readBridgeHeartbeat() {
		t.Fatal("readBridgeHeartbeat() = false, want true for live bridge heartbeat")
	}
	if readOrchestratorHeartbeat() {
		t.Fatal("readOrchestratorHeartbeat() = true with only bridge heartbeat present")
	}

	orchestratorHeartbeat := []byte(`{"status":"listening","pid":` + strconv.Itoa(os.Getpid()) + `}`)
	if err := os.WriteFile(filepath.Join(dir, "orchestrator_alive.json"), orchestratorHeartbeat, 0644); err != nil {
		t.Fatalf("write orchestrator heartbeat: %v", err)
	}

	if !readOrchestratorHeartbeat() {
		t.Fatal("readOrchestratorHeartbeat() = false, want true for live orchestrator heartbeat")
	}
}
