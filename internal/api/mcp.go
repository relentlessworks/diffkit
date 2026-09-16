package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/relentlessworks/diffkit/internal/diff"
)

// JSON-RPC 2.0 types

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type jsonRPCResponse struct {
	JSONRPC string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *jsonRPCError `json:"error,omitempty"`
}

// MCP tool definitions

type mcpTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

type mcpToolsResult struct {
	Tools []mcpTool `json:"tools"`
}

type mcpToolCallParams struct {
	Old     string `json:"old"`
	New     string `json:"new"`
	Text    string `json:"text"`
	Patch   string `json:"patch"`
	Context int    `json:"context"`
}

func mcpTools() []mcpTool {
	return []mcpTool{
		{
			Name:        "diff",
			Description: "Generate a unified diff between two texts. Returns the diff in unified diff format.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"old": map[string]interface{}{
						"type":        "string",
						"description": "The original text",
					},
					"new": map[string]interface{}{
						"type":        "string",
						"description": "The modified text",
					},
					"context": map[string]interface{}{
						"type":        "integer",
						"description": "Number of context lines around changes (default: 3)",
						"default":     3,
					},
				},
				"required": []string{"old", "new"},
			},
		},
		{
			Name:        "patch",
			Description: "Apply a unified diff patch to text. Returns the patched text.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "The original text to apply the patch to",
					},
					"patch": map[string]interface{}{
						"type":        "string",
						"description": "The unified diff patch to apply",
					},
				},
				"required": []string{"text", "patch"},
			},
		},
		{
			Name:        "worddiff",
			Description: "Generate a word-level diff between two texts. Uses [-deleted-] and {+inserted+} markers.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"old": map[string]interface{}{
						"type":        "string",
						"description": "The original text",
					},
					"new": map[string]interface{}{
						"type":        "string",
						"description": "The modified text",
					},
				},
				"required": []string{"old", "new"},
			},
		},
		{
			Name:        "summary",
			Description: "Get a summary of changes between two texts (added, removed, changed, unchanged line counts).",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"old": map[string]interface{}{
						"type":        "string",
						"description": "The original text",
					},
					"new": map[string]interface{}{
						"type":        "string",
						"description": "The modified text",
					},
				},
				"required": []string{"old", "new"},
			},
		},
	}
}

func (h *Handler) mcp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, r, "method not allowed", "POST JSON-RPC 2.0 requests to /mcp", http.StatusMethodNotAllowed)
		return
	}

	var req jsonRPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONRPCError(w, nil, -32700, "parse error: invalid JSON")
		return
	}

	if req.JSONRPC != "2.0" {
		writeJSONRPCError(w, req.ID, -32600, "invalid request: jsonrpc must be '2.0'")
		return
	}

	switch req.Method {
	case "initialize":
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    "diffkit",
				"version": "0.1.0",
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		})

	case "tools/list":
		writeJSONRPCResult(w, req.ID, mcpToolsResult{Tools: mcpTools()})

	case "tools/call":
		h.handleMCPToolCall(w, r, req)

	default:
		writeJSONRPCError(w, req.ID, -32601, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (h *Handler) handleMCPToolCall(w http.ResponseWriter, r *http.Request, req jsonRPCRequest) {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments,omitempty"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		writeJSONRPCError(w, req.ID, -32602, "invalid params: expected name and arguments")
		return
	}

	var args mcpToolCallParams
	if len(params.Arguments) > 0 {
		_ = json.Unmarshal(params.Arguments, &args)
	}

	switch params.Name {
	case "diff":
		if args.Old == "" && args.New == "" {
			writeJSONRPCError(w, req.ID, -32602, "missing required arguments: old and new")
			return
		}
		h.Store.IncrDiff()
		context := args.Context
		if context <= 0 {
			context = 3
		}
		result := diff.UnifiedDiff(args.Old, args.New, context)
		if result == "" {
			result = "identical=true\n"
		}
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": result},
			},
		})

	case "patch":
		if args.Text == "" {
			writeJSONRPCError(w, req.ID, -32602, "missing required argument: text")
			return
		}
		if args.Patch == "" {
			writeJSONRPCError(w, req.ID, -32602, "missing required argument: patch")
			return
		}
		h.Store.IncrPatch()
		result, err := diff.ApplyPatch(args.Text, args.Patch)
		if err != nil {
			writeJSONRPCError(w, req.ID, -32603, fmt.Sprintf("patch application failed: %v", err))
			return
		}
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": result},
			},
		})

	case "worddiff":
		if args.Old == "" && args.New == "" {
			writeJSONRPCError(w, req.ID, -32602, "missing required arguments: old and new")
			return
		}
		h.Store.IncrWord()
		wd := diff.ComputeWordDiff(args.Old, args.New)
		result := diff.FormatWordDiff(wd)
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": result},
			},
		})

	case "summary":
		if args.Old == "" && args.New == "" {
			writeJSONRPCError(w, req.ID, -32602, "missing required arguments: old and new")
			return
		}
		s := diff.Summarize(args.Old, args.New)
		result := fmt.Sprintf("added=%d removed=%d changed=%d unchanged=%d", s.Added, s.Removed, s.Changed, s.Unchanged)
		writeJSONRPCResult(w, req.ID, map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": result},
			},
		})

	default:
		writeJSONRPCError(w, req.ID, -32601, fmt.Sprintf("unknown tool: %s", params.Name))
	}
}

func writeJSONRPCResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	})
}

func writeJSONRPCError(w http.ResponseWriter, id interface{}, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &jsonRPCError{Code: code, Message: message},
	})
}
