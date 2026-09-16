package api

import (
	"net/http"
)

// helpText is the one-page operating manual for agents.
const helpText = `diffkit — Agentic-First Text Diff and Patch Service
====================================================

diffkit generates unified diffs, applies patches, compares texts with
word-level diffs, and summarizes changes. No database, no state, no
external dependencies. Single Go binary.

AUTHENTICATION
---------------
No auth required by default. If DIFFKIT_API_KEY is set, send:
  Authorization: Bearer <api-key>

ENDPOINTS
---------
GET /help     This operating manual (also at /.well-known/agent.md)
GET /health   Health check (returns "ok")
GET /diff     Generate a unified diff (query params: old, new, context)
POST /diff    Generate a unified diff (JSON body: old, new, context)
GET /patch    Apply a unified diff patch (query params: text, patch)
POST /patch   Apply a unified diff patch (JSON body: text, patch)
GET /worddiff Word-level diff (query params: old, new)
POST /worddiff Word-level diff (JSON body: old, new)
GET /summary  Diff summary (query params: old, new)
POST /summary Diff summary (JSON body: old, new)
POST /mcp     MCP (Model Context Protocol) endpoint

DIFF PARAMETERS
---------------
  old      Original text (required)
  new      Modified text (required)
  context  Number of context lines around changes (default: 3, use 0 for no context)

PATCH PARAMETERS
----------------
  text     Original text to apply the patch to (required)
  patch    Unified diff patch to apply (required)

RESPONSE FORMAT
---------------
Default: plain text.

  /diff returns a unified diff:
    --- old
    +++ new
    @@ -1,3 +1,3 @@
     line1
    -line2
    +modified
     line3

  /patch returns the patched text directly.

  /worddiff returns word-level diff with [-deleted-] and {+inserted+} markers:
    hello [-world-]{+earth+}

  /summary returns a one-line summary:
    added=1 removed=1 changed=2 unchanged=3

JSON: send Accept: application/json header or ?format=json query param.
  {"diff":"--- old\n+++ new\n@@ ...","summary":{"added":1,"removed":1,...}}

ERRORS
------
Errors are plain text with a hint:
  error: missing old and new text | hint: provide both old and new text, e.g. /diff?old=hello&new=world

EXAMPLES
--------
  # Generate a diff
  curl "http://localhost:8472/diff?old=hello+world&new=hello+earth"

  # Generate a diff with 1 line of context
  curl "http://localhost:8472/diff?old=line1%0Aline2%0Aline3&new=line1%0Achanged%0Aline3&context=1"

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

CONFIGURATION
-------------
  DIFFKIT_ADDR     Listen address (default :8472)
  DIFFKIT_API_KEY  API key for auth (default: none, no auth)
  -addr            Override listen address
  -api-key         Override API key

VERSION
-------
0.1.0
`

func (h *Handler) help(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(helpText))
}
