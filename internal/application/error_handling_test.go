package application

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDivisionByZero(t *testing.T) {
	// Убрать блокировку здесь
	// mu.Lock()
	// defer mu.Unlock()
	clearState()

	// Setup test data
	num1 := &Node{ID: "1", Type: "number", Status: "done", Result: 5}
	num2 := &Node{ID: "2", Type: "number", Status: "done", Result: 0}
	opNode := &Node{
		ID:        "3",
		Type:      "operation",
		Operation: "/",
		Left:      "1",
		Right:     "2",
		Status:    "pending",
		ExprID:    "e1",
	}
	expr := &Expression{ID: "e1", RootNodeID: "3", Status: "processing"}

	// Добавить локальную блокировку для настройки данных
	mu.Lock()
	nodes[num1.ID] = num1
	nodes[num2.ID] = num2
	nodes[opNode.ID] = opNode
	expressions[expr.ID] = expr
	mu.Unlock() // Важно разблокировать перед вызовом обработчика

	app := New()
	req := httptest.NewRequest("GET", "/internal/task", nil)
	w := httptest.NewRecorder()

	app.GetTaskHandler(w, req)

	// Проверки
	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	mu.Lock()
	defer mu.Unlock()
	if expr.Status != "error" {
		t.Errorf("Expected expression status 'error', got '%s'", expr.Status)
	}
}
