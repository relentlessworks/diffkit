package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/relentlessworks/diffkit/internal/store"
)

func setupTestHandler() *Handler {
	return New(store.New())
}

func TestRoot(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.root(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "diffkit") {
		t.Errorf("body does not contain 'diffkit': %s", w.Body.String())
	}
}

func TestHealth(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	h.health(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if w.Body.String() != "ok\n" {
		t.Errorf("body = %q, want %q", w.Body.String(), "ok\n")
	}
}

func TestHelp(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/help", nil)
	w := httptest.NewRecorder()
	h.help(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "diffkit") {
		t.Errorf("body does not contain 'diffkit'")
	}
	if !strings.Contains(body, "GET /diff") {
		t.Errorf("body does not contain endpoint documentation")
	}
}

func TestDiffGET(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/diff?old=hello+world&new=hello+earth", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "---") {
		t.Errorf("body should contain --- header: %s", body)
	}
	if !strings.Contains(body, "+++") {
		t.Errorf("body should contain +++ header: %s", body)
	}
}

func TestDiffGETIdentical(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/diff?old=hello&new=hello", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "identical=true") {
		t.Errorf("body should contain identical=true: %s", w.Body.String())
	}
}

func TestDiffPOSTJSON(t *testing.T) {
	h := setupTestHandler()
	body := `{"old":"line1\nline2","new":"line1\nmodified"}`
	req := httptest.NewRequest("POST", "/diff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "-line2") {
		t.Errorf("body should contain -line2: %s", w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "+modified") {
		t.Errorf("body should contain +modified: %s", w.Body.String())
	}
}

func TestDiffJSON(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/diff?old=hello&new=world&format=json", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result["identical"] != false {
		t.Errorf("expected identical=false, got %v", result["identical"])
	}
}

func TestDiffMissingParams(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/diff", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if !strings.Contains(w.Body.String(), "hint:") {
		t.Errorf("error should contain hint: %s", w.Body.String())
	}
}

func TestDiffWrongMethod(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("DELETE", "/diff", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestDiffContextParam(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/diff?old=a%0Ab%0Ac%0Ad%0Ae&new=a%0AX%0Ac%0Ad%0Ae&context=0", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	// With context=0, there should be no context lines
	if strings.Contains(body, " a\n") {
		t.Errorf("with context=0, should not have context line 'a': %s", body)
	}
}

func TestDiffInvalidContext(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/diff?old=hello&new=world&context=abc", nil)
	w := httptest.NewRecorder()
	h.diffHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPatchPOSTJSON(t *testing.T) {
	h := setupTestHandler()
	body := `{"text":"hello\nworld","patch":"--- old\n+++ new\n@@ -1,2 +1,2 @@\n hello\n-world\n+earth\n"}`
	req := httptest.NewRequest("POST", "/patch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.patchHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "earth") {
		t.Errorf("body should contain 'earth': %s", w.Body.String())
	}
	if strings.Contains(w.Body.String(), "world") {
		t.Errorf("body should not contain 'world': %s", w.Body.String())
	}
}

func TestPatchGET(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/patch?text=hello&patch=---%20old%0A%2B%2B%2B%20new%0A@@%20-1%20%2B1%20@@%0A-hello%0A%2Bworld", nil)
	w := httptest.NewRecorder()
	h.patchHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "world") {
		t.Errorf("body should contain 'world': %s", w.Body.String())
	}
}

func TestPatchMissingText(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/patch?patch=---%20old", nil)
	w := httptest.NewRecorder()
	h.patchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPatchMissingPatch(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/patch?text=hello", nil)
	w := httptest.NewRecorder()
	h.patchHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestPatchWrongMethod(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("DELETE", "/patch", nil)
	w := httptest.NewRecorder()
	h.patchHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestPatchJSON(t *testing.T) {
	h := setupTestHandler()
	body := `{"text":"hello\nworld","patch":"--- old\n+++ new\n@@ -1,2 +1,2 @@\n hello\n-world\n+earth\n"}`
	req := httptest.NewRequest("POST", "/patch?format=json", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.patchHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result["success"] != true {
		t.Errorf("expected success=true, got %v", result["success"])
	}
}

func TestWordDiffGET(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/worddiff?old=hello+world&new=hello+earth", nil)
	w := httptest.NewRecorder()
	h.wordDiffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "[-") {
		t.Errorf("body should contain deletion marker: %s", body)
	}
	if !strings.Contains(body, "{+") {
		t.Errorf("body should contain insertion marker: %s", body)
	}
}

func TestWordDiffPOSTJSON(t *testing.T) {
	h := setupTestHandler()
	body := `{"old":"hello world","new":"hello earth"}`
	req := httptest.NewRequest("POST", "/worddiff", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.wordDiffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if !strings.Contains(w.Body.String(), "earth") {
		t.Errorf("body should contain 'earth': %s", w.Body.String())
	}
}

func TestWordDiffJSON(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/worddiff?old=hello+world&new=hello+earth&format=json", nil)
	w := httptest.NewRecorder()
	h.wordDiffHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if _, ok := result["words"]; !ok {
		t.Errorf("expected 'words' in JSON response")
	}
}

func TestWordDiffMissingParams(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/worddiff", nil)
	w := httptest.NewRecorder()
	h.wordDiffHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWordDiffWrongMethod(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("DELETE", "/worddiff", nil)
	w := httptest.NewRecorder()
	h.wordDiffHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestSummaryGET(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/summary?old=a%0Ab%0Ac&new=a%0AB%0Ac%0Ad", nil)
	w := httptest.NewRecorder()
	h.summaryHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "added=") {
		t.Errorf("body should contain 'added=': %s", body)
	}
	if !strings.Contains(body, "removed=") {
		t.Errorf("body should contain 'removed=': %s", body)
	}
}

func TestSummaryJSON(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/summary?old=hello&new=world&format=json", nil)
	w := httptest.NewRecorder()
	h.summaryHandler(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if _, ok := result["added"]; !ok {
		t.Errorf("expected 'added' in JSON response")
	}
}

func TestSummaryMissingParams(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/summary", nil)
	w := httptest.NewRecorder()
	h.summaryHandler(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSummaryWrongMethod(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("DELETE", "/summary", nil)
	w := httptest.NewRecorder()
	h.summaryHandler(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

// --- MCP Tests ---

func TestMCPInitialize(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize"}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %s", result.Error.Message)
	}
}

func TestMCPToolsList(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result struct {
		JSONRPC string `json:"jsonrpc"`
		ID      int    `json:"id"`
		Result  struct {
			Tools []mcpTool `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if len(result.Result.Tools) != 4 {
		t.Errorf("expected 4 tools, got %d", len(result.Result.Tools))
	}
}

func TestMCPDiff(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"diff","arguments":{"old":"hello\nworld","new":"hello\nearth"}}}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %s", result.Error.Message)
	}
}

func TestMCPPatch(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"patch","arguments":{"text":"hello\nworld","patch":"--- old\n+++ new\n@@ -1,2 +1,2 @@\n hello\n-world\n+earth\n"}}}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %s", result.Error.Message)
	}
}

func TestMCPWordDiff(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"worddiff","arguments":{"old":"hello world","new":"hello earth"}}}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %s", result.Error.Message)
	}
}

func TestMCPSummary(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"summary","arguments":{"old":"a\nb","new":"a\nB"}}}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var result jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Error != nil {
		t.Errorf("unexpected error: %s", result.Error.Message)
	}
}

func TestMCPUnknownMethod(t *testing.T) {
	h := setupTestHandler()
	body := `{"jsonrpc":"2.0","id":7,"method":"unknown_method"}`
	req := httptest.NewRequest("POST", "/mcp", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.mcp(w, req)
	var result jsonRPCResponse
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}
	if result.Error == nil {
		t.Errorf("expected error for unknown method")
	}
}

func TestMCPWrongMethod(t *testing.T) {
	h := setupTestHandler()
	req := httptest.NewRequest("GET", "/mcp", nil)
	w := httptest.NewRecorder()
	h.mcp(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}
