# ModSources path resolution (single source of truth)

**Audience:** TUI (Go), orchestrator (Python), Gatekeeper (Python), ForgeConnector (C#).  
**Goal:** Every component resolves the tModLoader `ModSources` directory the same way when the user has not manually set mismatched environment variables.

## Precedence (highest first)

1. **`FORGE_MOD_SOURCES_DIR`** — If set and non-whitespace after trim, use `Path.expanduser()` (Python) / use as-is in Go (user should pass absolute paths; `~` is expanded in Python).
2. **`~/.config/theforge/config.toml`** — Root keys only (lines before the first `[section]` header). Key: `mod_sources_dir = "/path/to/ModSources"` (optional quotes; inline `#` comments stripped like the TUI).
3. **OS default** — Same layout as tModLoader:
   - **Windows:** `%USERPROFILE%\Documents\My Games\Terraria\tModLoader\ModSources`
   - **Linux:** `~/.local/share/Terraria/tModLoader/ModSources`
   - **macOS / other Unix:** `~/Library/Application Support/Terraria/tModLoader/ModSources`

## Implementation

| Component | Module / location |
|-----------|-------------------|
| Python canonical helper | [`agents/paths.py`](../../agents/paths.py) — `mod_sources_root()` |
| Go TUI | [`BubbleTeaTerminal/main.go`](../../BubbleTeaTerminal/main.go) — `modSourcesDir()` (reference; keep in sync) |
| C# bridge | [`mod/ForgeConnector/ForgeConnectorSystem.cs`](../../mod/ForgeConnector/ForgeConnectorSystem.cs) — `GetModSourcesDir()` reads `FORGE_MOD_SOURCES_DIR` then OS defaults (config.toml is not read in-game; set env to match TUI if using a custom path from config only) |

## Heartbeat alignment

The orchestrator writes `mod_sources_root` into [`orchestrator_alive.json`](../../agents/orchestrator.py) so the TUI can compare the resolved path with its own `modSourcesDir()` and warn if they differ (e.g. stale heartbeat from another machine).
