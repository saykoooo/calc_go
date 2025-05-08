package application

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saykoooo/calc_go/internal/calc"
	"github.com/saykoooo/calc_go/internal/db"
)

func TestDivisionByZero(t *testing.T) {
	nodes := []*calc.Node{
		{ID: "1", ExprID: "e1", Type: "number", Status: "done", Result: 5},
		{ID: "2", ExprID: "e1", Type: "number", Status: "done", Result: 0},
		{
			ID:        "3",
			ExprID:    "e1",
			Type:      "operation",
			Operation: "/",
			Left:      "1",
			Right:     "2",
			Status:    "pending",
		},
	}

	expr := db.Expression{
		ExprID:     "e1",
		Status:     "processing",
		RootNodeID: "3",
	}

	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	clearState("e1")

	_, err = db.InsertExpression(expr)
	if err != nil {
		t.Fatalf("Failed to insert expression: %v", err)
	}

	_, err = db.InsertNodes(nodes)
	if err != nil {
		t.Fatalf("Failed to insert nodes: %v", err)
	}

	app := New()
	req := httptest.NewRequest("GET", "/internal/task", nil)
	w := httptest.NewRecorder()

	app.GetTaskHandler(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}

	updatedExpr, err := db.SelectExpression("e1")
	if err != nil {
		t.Fatalf("Failed to get expression: %v", err)
	}

	if updatedExpr.Status != "error" {
		t.Errorf("Expected expression status 'error', got '%s'", updatedExpr.Status)
	}

	updatedNode, err := db.SelectNode("3")
	if err != nil {
		t.Fatalf("Failed to get node: %v", err)
	}

	if updatedNode.Status != "error" {
		t.Errorf("Expected node status 'error', got '%s'", updatedNode.Status)
	}
}
