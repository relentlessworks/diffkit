package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/relentlessworks/diffkit/internal/diff"
	"github.com/relentlessworks/diffkit/internal/store"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	Store *store.Store
}

// New creates a new API handler.
func New(s *store.Store) *Handler {
	return &Handler{Store: s}
}

// RegisterRoutes wires all endpoints onto the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/", h.root)
	mux.HandleFunc("/help", h.help)
	mux.HandleFunc("/.well-known/agent.md", h.help)
	mux.HandleFunc("/health", h.health)
	mux.HandleFunc("/diff", h.diffHandler)
	mux.HandleFunc("/patch", h.patchHandler)
	mux.HandleFunc("/worddiff", h.wordDiffHandler)
	mux.HandleFunc("/summary", h.summaryHandler)
	mux.HandleFunc("/mcp", h.mcp)
}

// --- Helpers ---

func wantsJSON(r *http.Request) bool {
	if r.URL.Query().Get("format") == "json" {
		return true
	}
	accept := r.Header.Get("Accept")
	return strings.Contains(accept, "application/json")
}

func writeError(w http.ResponseWriter, r *http.Request, msg, hint string, code int) {
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]string{
			"error": msg,
			"hint":  hint,
		})
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "error: %s | hint: %s\n", msg, hint)
}

func writeText(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Write([]byte(text))
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// --- Handlers ---

func (h *Handler) root(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	writeText(w, "diffkit — agentic-first text diff and patch service | hint: GET /help for the operating manual\n")
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeText(w, "ok\n")
}

// diffRequest holds the parameters for a diff request.
type diffRequest struct {
	Old     string `json:"old"`
	New     string `json:"new"`
	Context int    `json:"context"`
}

func (h *Handler) diffHandler(w http.ResponseWriter, r *http.Request) {
	var oldText, newText string
	context := 3

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, "failed to read request body", "ensure the request body is valid", http.StatusBadRequest)
			return
		}

		// Try JSON first
		var req diffRequest
		if err := json.Unmarshal(body, &req); err == nil && (req.Old != "" || req.New != "") {
			oldText = req.Old
			newText = req.New
			if req.Context > 0 {
				context = req.Context
			}
		} else {
			// Try form-encoded: old=...&new=...
			values, err := url.ParseQuery(string(body))
			if err == nil {
				oldText = values.Get("old")
				newText = values.Get("new")
				if c := values.Get("context"); c != "" {
					if n, err := strconv.Atoi(c); err == nil && n >= 0 {
						context = n
					}
				}
			} else {
				writeError(w, r, "invalid request body", "send JSON {\"old\":\"...\",\"new\":\"...\"} or form-encoded old=...&new=...", http.StatusBadRequest)
				return
			}
		}
	} else if r.Method == http.MethodGet {
		oldText = r.URL.Query().Get("old")
		newText = r.URL.Query().Get("new")
		if c := r.URL.Query().Get("context"); c != "" {
			if n, err := strconv.Atoi(c); err == nil && n >= 0 {
				context = n
			} else {
				writeError(w, r, "invalid context parameter", "context must be a non-negative integer, e.g. /diff?context=3", http.StatusBadRequest)
				return
			}
		}
	} else {
		writeError(w, r, "method not allowed", "use GET /diff?old=...&new=... or POST /diff with JSON body", http.StatusMethodNotAllowed)
		return
	}

	if oldText == "" && newText == "" {
		writeError(w, r, "missing old and new text", "provide both old and new text, e.g. /diff?old=hello&new=world or POST JSON {\"old\":\"...\",\"new\":\"...\"}", http.StatusBadRequest)
		return
	}

	h.Store.IncrDiff()

	result := diff.UnifiedDiff(oldText, newText, context)

	if result == "" {
		if wantsJSON(r) {
			writeJSON(w, map[string]interface{}{
				"identical": true,
				"diff":      "",
				"summary": map[string]int{
					"added":     0,
					"removed":   0,
					"unchanged": diff.Summarize(oldText, newText).Unchanged,
				},
			})
			return
		}
		writeText(w, "identical=true\n")
		return
	}

	if wantsJSON(r) {
		s := diff.Summarize(oldText, newText)
		writeJSON(w, map[string]interface{}{
			"identical": false,
			"diff":      result,
			"summary": map[string]int{
				"added":     s.Added,
				"removed":   s.Removed,
				"unchanged": s.Unchanged,
			},
		})
		return
	}

	writeText(w, result)
}

// patchRequest holds the parameters for a patch request.
type patchRequest struct {
	Text  string `json:"text"`
	Patch string `json:"patch"`
}

func (h *Handler) patchHandler(w http.ResponseWriter, r *http.Request) {
	var text, patch string

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, "failed to read request body", "ensure the request body is valid", http.StatusBadRequest)
			return
		}

		var req patchRequest
		if err := json.Unmarshal(body, &req); err == nil && (req.Text != "" || req.Patch != "") {
			text = req.Text
			patch = req.Patch
		} else {
			values, err := url.ParseQuery(string(body))
			if err == nil {
				text = values.Get("text")
				patch = values.Get("patch")
			} else {
				writeError(w, r, "invalid request body", "send JSON {\"text\":\"...\",\"patch\":\"...\"} or form-encoded text=...&patch=...", http.StatusBadRequest)
				return
			}
		}
	} else if r.Method == http.MethodGet {
		text = r.URL.Query().Get("text")
		patch = r.URL.Query().Get("patch")
	} else {
		writeError(w, r, "method not allowed", "use GET /patch?text=...&patch=... or POST /patch with JSON body", http.StatusMethodNotAllowed)
		return
	}

	if text == "" {
		writeError(w, r, "missing text parameter", "provide the original text to patch, e.g. /patch?text=hello&patch=...", http.StatusBadRequest)
		return
	}
	if patch == "" {
		writeError(w, r, "missing patch parameter", "provide a unified diff patch, e.g. /patch?text=...&patch=---%20old%0A+++%20new%0A@@%20-1%20+1%20@@%0A-hello%0A+world", http.StatusBadRequest)
		return
	}

	h.Store.IncrPatch()

	result, err := diff.ApplyPatch(text, patch)
	if err != nil {
		writeError(w, r, fmt.Sprintf("patch application failed: %v", err), "ensure the patch is a valid unified diff that matches the provided text", http.StatusUnprocessableEntity)
		return
	}

	if wantsJSON(r) {
		writeJSON(w, map[string]interface{}{
			"success": true,
			"result":  result,
		})
		return
	}

	writeText(w, result)
}

// wordDiffRequest holds the parameters for a word diff request.
type wordDiffRequest struct {
	Old string `json:"old"`
	New string `json:"new"`
}

func (h *Handler) wordDiffHandler(w http.ResponseWriter, r *http.Request) {
	var oldText, newText string

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, "failed to read request body", "ensure the request body is valid", http.StatusBadRequest)
			return
		}

		var req wordDiffRequest
		if err := json.Unmarshal(body, &req); err == nil && (req.Old != "" || req.New != "") {
			oldText = req.Old
			newText = req.New
		} else {
			values, err := url.ParseQuery(string(body))
			if err == nil {
				oldText = values.Get("old")
				newText = values.Get("new")
			} else {
				writeError(w, r, "invalid request body", "send JSON {\"old\":\"...\",\"new\":\"...\"} or form-encoded old=...&new=...", http.StatusBadRequest)
				return
			}
		}
	} else if r.Method == http.MethodGet {
		oldText = r.URL.Query().Get("old")
		newText = r.URL.Query().Get("new")
	} else {
		writeError(w, r, "method not allowed", "use GET /worddiff?old=...&new=... or POST /worddiff with JSON body", http.StatusMethodNotAllowed)
		return
	}

	if oldText == "" && newText == "" {
		writeError(w, r, "missing old and new text", "provide both old and new text, e.g. /worddiff?old=hello+world&new=hello+earth", http.StatusBadRequest)
		return
	}

	h.Store.IncrWord()

	wd := diff.ComputeWordDiff(oldText, newText)
	formatted := diff.FormatWordDiff(wd)

	if wantsJSON(r) {
		words := make([]map[string]interface{}, 0, len(wd.Words))
		for _, w := range wd.Words {
			opStr := "equal"
			switch w.Op {
			case diff.OpInsert:
				opStr = "insert"
			case diff.OpDelete:
				opStr = "delete"
			}
			words = append(words, map[string]interface{}{
				"op":   opStr,
				"text": w.Text,
			})
		}
		writeJSON(w, map[string]interface{}{
			"formatted": formatted,
			"words":     words,
		})
		return
	}

	writeText(w, formatted+"\n")
}

func (h *Handler) summaryHandler(w http.ResponseWriter, r *http.Request) {
	var oldText, newText string

	if r.Method == http.MethodPost {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeError(w, r, "failed to read request body", "ensure the request body is valid", http.StatusBadRequest)
			return
		}

		var req diffRequest
		if err := json.Unmarshal(body, &req); err == nil && (req.Old != "" || req.New != "") {
			oldText = req.Old
			newText = req.New
		} else {
			values, err := url.ParseQuery(string(body))
			if err == nil {
				oldText = values.Get("old")
				newText = values.Get("new")
			} else {
				writeError(w, r, "invalid request body", "send JSON {\"old\":\"...\",\"new\":\"...\"} or form-encoded old=...&new=...", http.StatusBadRequest)
				return
			}
		}
	} else if r.Method == http.MethodGet {
		oldText = r.URL.Query().Get("old")
		newText = r.URL.Query().Get("new")
	} else {
		writeError(w, r, "method not allowed", "use GET /summary?old=...&new=... or POST /summary with JSON body", http.StatusMethodNotAllowed)
		return
	}

	if oldText == "" && newText == "" {
		writeError(w, r, "missing old and new text", "provide both old and new text, e.g. /summary?old=hello&new=world", http.StatusBadRequest)
		return
	}

	s := diff.Summarize(oldText, newText)

	if wantsJSON(r) {
		writeJSON(w, map[string]interface{}{
			"added":     s.Added,
			"removed":   s.Removed,
			"changed":   s.Changed,
			"unchanged": s.Unchanged,
		})
		return
	}

	writeText(w, fmt.Sprintf("added=%d removed=%d changed=%d unchanged=%d\n", s.Added, s.Removed, s.Changed, s.Unchanged))
}
