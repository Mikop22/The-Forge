# Spike: localhost IPC between TUI and orchestrator (optional / future)

**Status:** design note only — file-based requests and status remain the source of truth for large artifacts.

## Goal

Replace or supplement `user_request.json` / polling with a lightweight control channel for *acknowledgements*, cancellation, and back-pressure while keeping sprite payloads and `generation_status.json` on disk where they belong.

## Options

| Approach | Pros | Cons |
|----------|------|------|
| **Unix domain socket** (macOS/Linux) | Low latency, clear framing | Windows needs named pipes or a separate code path |
| **TCP loopback** (`127.0.0.1:port`) | Cross-platform, easy to prototype | Firewall/port collisions; must authenticate or bind to localhost only |
| **stdio / subprocess** | Simple when TUI spawns orchestrator | Poor fit when orchestrator is long-lived and shared |

## Recommendation

If implemented, prefer **TCP on loopback** with a random free port written to `orchestrator_ipc.json` in ModSources (path + shared secret token). The TUI would send small JSON control messages; binary blobs and large JSON would still use files.

## Tradeoffs vs files-only

- **IPC:** faster handshake, explicit “request accepted”, optional cancel — at the cost of connection lifecycle, reconnection, and cross-platform testing.
- **Files:** robust under sync folders and crashy editors; already integrated with Watchdog — at the cost of polling latency and no explicit cancel path.

Phases A–C of the reliability roadmap do not require IPC; this spike is intentionally deferred until file-based flow is stable.
