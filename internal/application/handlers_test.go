package application

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/saykoooo/calc_go/internal/calc"
	"github.com/saykoooo/calc_go/internal/db"
)

func TestCalcHandler_Success(t *testing.T) {
	// Initialize database
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	// Setup test user
	user := &db.User{
		Name:     "testuser",
		Password: "testpass",
	}
	_, err = db.InsertUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test request
	reqBody := `{"expression": "2 + 3 * 4"}`
	req := httptest.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
	req.Header.Set("username", "testuser")
	w := httptest.NewRecorder()

	// Call handler
	CalcHandler(w, req)

	// Verify response
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

	// Cleanup
	clearState(response["id"])
}

// func aTestCalcHandler_Success(t *testing.T) {
// 	clearState()
// 	reqBody := `{"expression": "2 + 3 * 4"}`
// 	req := httptest.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
// 	w := httptest.NewRecorder()
// 	CalcHandler(w, req)
// 	if w.Code != http.StatusCreated {
// 		t.Errorf("Expected status code %d, got %d", http.StatusCreated, w.Code)
// 	}
// 	var response map[string]string
// 	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
// 		t.Fatalf("Error decoding response: %v", err)
// 	}
// 	if response["id"] == "" {
// 		t.Error("Expected non-empty ID in response")
// 	}
// }

func TestGetTaskHandler_WithTask(t *testing.T) {
	// Initialize database
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	// Setup test data
	expr := db.Expression{
		ExprID:     "testexpr",
		Status:     "processing",
		RootNodeID: "3",
	}
	_, err = db.InsertExpression(expr)
	if err != nil {
		t.Fatalf("Failed to insert test expression: %v", err)
	}

	nodes := []*calc.Node{
		{ID: "a1", ExprID: "e2", Type: "number", Status: "done", Result: 2},
		{ID: "a2", ExprID: "e2", Type: "number", Status: "done", Result: 3},
		{
			ID:        "a3",
			ExprID:    "e2",
			Type:      "operation",
			Operation: "+",
			Left:      "a1",
			Right:     "a2",
			Status:    "pending",
		},
	}

	_, err = db.InsertNodes(nodes)
	if err != nil {
		t.Fatalf("Failed to insert test nodes: %v", err)
	}

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
	if task["id"] != "a3" {
		t.Errorf("Expected task ID 'a3', got '%v'", task["id"])
	}

	// Cleanup
	clearState("e2")
}

// func TestGetTaskHandler_WithTask(t *testing.T) {
// 	clearState()
// 	num1 := &calc.Node{ID: "1", Type: "number", Status: "done", Result: 2}
// 	num2 := &calc.Node{ID: "2", Type: "number", Status: "done", Result: 3}
// 	opNode := &calc.Node{
// 		ID:        "3",
// 		Type:      "operation",
// 		Operation: "+",
// 		Left:      "1",
// 		Right:     "2",
// 		Status:    "pending",
// 	}
// 	mu.Lock()
// 	nodes[num1.ID] = num1
// 	nodes[num2.ID] = num2
// 	nodes[opNode.ID] = opNode
// 	mu.Unlock()
// 	app := New()
// 	req := httptest.NewRequest("GET", "/internal/task", nil)
// 	w := httptest.NewRecorder()
// 	app.GetTaskHandler(w, req)
// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}
// 	var response map[string]interface{}
// 	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
// 		t.Fatalf("Error decoding response: %v", err)
// 	}
// 	task := response["task"].(map[string]interface{})
// 	if task["id"] != "3" {
// 		t.Errorf("Expected task ID '3', got '%v'", task["id"])
// 	}
// }

func TestPostTaskHandler_Success(t *testing.T) {
	// Initialize database
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	// Setup test data
	expr := db.Expression{
		ExprID:     "e3",
		Status:     "processing",
		RootNodeID: "b1",
	}
	_, err = db.InsertExpression(expr)
	if err != nil {
		t.Fatalf("Failed to insert test expression: %v", err)
	}

	nodes := []*calc.Node{
		{
			ID:     "b1",
			ExprID: "e3",
			Type:   "operation",
			Status: "in_progress",
		},
	}
	_, err = db.InsertNodes(nodes)
	if err != nil {
		t.Fatalf("Failed to insert test nodes: %v", err)
	}

	// Create test request
	reqBody := `{"id": "b1", "result": 5}`
	req := httptest.NewRequest("POST", "/internal/task", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	PostTaskHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Verify database state
	updatedExpr, err := db.SelectExpression("e3")
	if err != nil {
		t.Fatalf("Failed to get expression: %v", err)
	}

	if updatedExpr.Status != "done" {
		t.Errorf("Expected expression status 'done', got %s", updatedExpr.Status)
	}

	if updatedExpr.Result != 5.0 {
		t.Errorf("Expected expression result 5.0, got %f", updatedExpr.Result)
	}

	// Cleanup
	clearState("e3")
}

// func TestPostTaskHandler_Success(t *testing.T) {
// 	clearState()

// 	mu.Lock()
// 	expr := &Expression{ID: "e1", RootNodeID: "1", Status: "processing"}
// 	node := &calc.Node{ID: "1", ExprID: "e1", Type: "operation", Status: "in_progress"}
// 	expressions[expr.ID] = expr
// 	nodes[node.ID] = node
// 	mu.Unlock()

// 	reqBody := `{"id": "1", "result": 5}`
// 	req := httptest.NewRequest("POST", "/internal/task", strings.NewReader(reqBody))
// 	w := httptest.NewRecorder()

// 	PostTaskHandler(w, req)

// 	if w.Code != http.StatusOK {
// 		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
// 	}

// 	if node.Status != "done" {
// 		t.Errorf("Expected node status 'done', got '%s'", node.Status)
// 	}

// 	if expr.Result != 5.0 {
// 		t.Errorf("Expected expression result 5.0, got %f", expr.Result)
// 	}
// }

func clearState(expr_id string) {
	db.DeleteNodes(expr_id)
	db.DeleteExpression(expr_id)
}
