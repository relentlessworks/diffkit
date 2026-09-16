# diffkit

> Agentic-first text diff and patch service. Generate unified diffs, apply patches, compare texts with word-level diffs, and summarize changes. Plain text API, agent-driven, single Go binary.

## Quick Start

```bash
# Build
make build

# Run
./diffkit

# Generate a diff
curl "http://localhost:8472/diff?old=hello+world&new=hello+earth"

# Generate a diff via POST (JSON)
curl -X POST http://localhost:8472/diff \
  -H "Content-Type: application/json" \
  -d '{"old":"hello\nworld","new":"hello\nearth"}'

# Apply a patch
curl -X POST http://localhost:8472/patch \
  -H "Content-Type: application/json" \
  -d '{"text":"hello\nworld","patch":"--- old\n+++ new\n@@ -1,2 +1,2 @@\n hello\n-world\n+earth\n"}'

# Word-level diff
curl "http://localhost:8472/worddiff?old=hello+world&new=hello+earth"

# Diff summary
curl "http://localhost:8472/summary?old=a%0Ab%0Ac&new=a%0AB%0Ac%0Ad"

# Get JSON output
curl -H "Accept: application/json" "http://localhost:8472/diff?old=hello&new=world"
```

## API Reference

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/help` | Operating manual (also at `/.well-known/agent.md`) |
| GET | `/health` | Health check |
| GET | `/diff` | Generate unified diff (params: `old`, `new`, `context`) |
| POST | `/diff` | Generate unified diff (JSON body: `old`, `new`, `context`) |
| GET | `/patch` | Apply a unified diff patch (params: `text`, `patch`) |
| POST | `/patch` | Apply a unified diff patch (JSON body: `text`, `patch`) |
| GET | `/worddiff` | Word-level diff (params: `old`, `new`) |
| POST | `/worddiff` | Word-level diff (JSON body: `old`, `new`) |
| GET | `/summary` | Diff summary (params: `old`, `new`) |
| POST | `/summary` | Diff summary (JSON body: `old`, `new`) |
| POST | `/mcp` | MCP (Model Context Protocol) endpoint |

### Response Format

**Plain text (default):**

`/diff` returns a unified diff:
```
--- old
+++ new
@@ -1,3 +1,3 @@
 line1
-line2
+modified
 line3
```

`/patch` returns the patched text directly.

`/worddiff` returns word-level diff with `[-deleted-]` and `{+inserted+}` markers:
```
hello [-world-]{+earth+}
```

`/summary` returns a one-line summary:
```
added=1 removed=1 changed=2 unchanged=3
```

**JSON:** Send `Accept: application/json` header or `?format=json` query param.

```json
{"diff":"--- old\n+++ new\n@@ ...","summary":{"added":1,"removed":1,...}}
```

### Errors

Errors include a hint for self-correction:

```
error: missing old and new text | hint: provide both old and new text, e.g. /diff?old=hello&new=world
```

## Configuration

| Source | Variable | Default | Description |
|--------|----------|---------|-------------|
| Env | `DIFFKIT_ADDR` | `:8472` | Listen address |
| Env | `DIFFKIT_API_KEY` | (empty) | API key for auth (no auth if empty) |
| Flag | `-addr` | `:8472` | Override listen address |
| Flag | `-api-key` | (empty) | Override API key |

Priority: defaults < env vars < flags.

## MCP Integration

diffkit speaks Model Context Protocol at `POST /mcp` for chat client integrations (Claude, Cursor, etc.).

**Tools:**
- `diff` — Generate a unified diff (params: `old`, `new`, `context`)
- `patch` — Apply a unified diff patch (params: `text`, `patch`)
- `worddiff` — Word-level diff (params: `old`, `new`)
- `summary` — Diff summary (params: `old`, `new`)

## Build

```bash
make build    # CGO_ENABLED=0, single static binary
make test     # go test -race
make vet      # go vet
make run      # build + run
```

## Design Principles

- **Agent IS the interface** — No UI, no SDK. The API is the product.
- **Plain text by default** — Token-cheap, grepable, one record per line.
- **Instructive errors** — Every 4xx includes a hint for self-correction.
- **Self-documenting** — `GET /help` returns the full operating manual.
- **Single static binary** — Go, zero external dependencies, CGO_ENABLED=0.
- **Zero config** — Runs out of the box with sensible defaults.

## License

MIT
