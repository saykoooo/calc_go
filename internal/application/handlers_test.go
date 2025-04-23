package application

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saykoooo/calc_go/internal/calc"
)

func TestCalcHandler_Success(t *testing.T) {
	clearState()

	reqBody := `{"expression": "2 + 3 * 4"}`
	req := httptest.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	CalcHandler(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if response["id"] == "" {
		t.Error("Expected non-empty ID in response")
	}
}

func TestGetTaskHandler_WithTask(t *testing.T) {
	clearState()

	num1 := &calc.Node{ID: "1", Type: "number", Status: "done", Result: 2}
	num2 := &calc.Node{ID: "2", Type: "number", Status: "done", Result: 3}
	opNode := &calc.Node{
		ID:        "3",
		Type:      "operation",
		Operation: "+",
		Left:      "1",
		Right:     "2",
		Status:    "pending",
	}
	mu.Lock()
	nodes[num1.ID] = num1
	nodes[num2.ID] = num2
	nodes[opNode.ID] = opNode
	mu.Unlock()

	app := New()
	req := httptest.NewRequest("GET", "/internal/task", nil)
	w := httptest.NewRecorder()

	app.GetTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	task := response["task"].(map[string]interface{})
	if task["id"] != "3" {
		t.Errorf("Expected task ID '3', got '%v'", task["id"])
	}
}

func TestPostTaskHandler_Success(t *testing.T) {
	clearState()

	mu.Lock()
	expr := &Expression{ID: "e1", RootNodeID: "1", Status: "processing"}
	node := &calc.Node{ID: "1", ExprID: "e1", Type: "operation", Status: "in_progress"}
	expressions[expr.ID] = expr
	nodes[node.ID] = node
	mu.Unlock()

	reqBody := `{"id": "1", "result": 5}`
	req := httptest.NewRequest("POST", "/internal/task", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	PostTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	if node.Status != "done" {
		t.Errorf("Expected node status 'done', got '%s'", node.Status)
	}

	if expr.Result != 5.0 {
		t.Errorf("Expected expression result 5.0, got %f", expr.Result)
	}
}

func clearState() {
	mu.Lock()
	defer mu.Unlock()
	expressions = make(map[string]*Expression)
	nodes = make(map[string]*calc.Node)
}
