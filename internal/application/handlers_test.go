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
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	reqBody := `{"expression": "2 + 3 * 4"}`
	req := httptest.NewRequest("POST", "/api/v1/calculate", strings.NewReader(reqBody))
	req.Header.Set("username", "testuser")
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

	clearState(response["id"])
}

func clearState(expr_id string) {
	db.DeleteNodes(expr_id)
	db.DeleteExpression(expr_id)
}

func TestRegisterHandler_Success(t *testing.T) {
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	reqBody := `{"login": "newuser", "password": "newpass"}`
	req := httptest.NewRequest("POST", "/api/v1/register", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	RegisterHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	db.DeleteUser("newuser")
}

func TestRegisterHandler_UserExists(t *testing.T) {
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	password, err := db.GenerateHash("existingpass")
	if err != nil {
		t.Fatalf("Error while generating hash: %v", err)
	}
	user := &db.User{
		Name:     "existinguser",
		Password: password,
	}
	_, err = db.InsertUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	reqBody := `{"login": "existinguser", "password": "newpass"}`
	req := httptest.NewRequest("POST", "/api/v1/register", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	RegisterHandler(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %d, got %d", http.StatusBadRequest, w.Code)
	}

	db.DeleteUser("existinguser")
}

func TestLoginHandler_Success(t *testing.T) {
	app := New()
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	password, err := db.GenerateHash("loginpass")
	if err != nil {
		t.Fatalf("Error while generating hash: %v", err)
	}
	user := &db.User{
		Name:     "loginuser",
		Password: password,
	}
	_, err = db.InsertUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	reqBody := `{"login": "loginuser", "password": "loginpass"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	app.LoginHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if response["token"] == "" {
		t.Error("Expected non-empty token in response")
	}

	db.DeleteUser("loginuser")
}

func TestLoginHandler_InvalidCredentials(t *testing.T) {
	app := New()

	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	password, err := db.GenerateHash("loginpass2")
	if err != nil {
		t.Fatalf("Error while generating hash: %v", err)
	}
	user := &db.User{
		Name:     "loginuser2",
		Password: password,
	}
	_, err = db.InsertUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	reqBody := `{"login": "loginuser2", "password": "wrongpass"}`
	req := httptest.NewRequest("POST", "/api/v1/login", strings.NewReader(reqBody))
	w := httptest.NewRecorder()

	app.LoginHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}

	db.DeleteUser("loginuser2")
}

func TestGetExpressionsHandler_Success(t *testing.T) {
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	user := &db.User{
		Name:     "expruser",
		Password: "exprpass",
	}
	_, err = db.InsertUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	expr := db.Expression{
		ExprID:     "testexpr1",
		Username:   "expruser",
		Status:     "completed",
		RootNodeID: "root1",
		Expr:       "2 + 2",
		Result:     4,
	}
	_, err = db.InsertExpression(expr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/expressions", nil)
	req.Header.Set("username", "expruser")
	w := httptest.NewRecorder()

	GetExpressionsHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response struct {
		Expressions []ExpressionStatus `json:"expressions"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if len(response.Expressions) == 0 {
		t.Error("Expected at least one expression in response")
	}

	db.DeleteExpression("testexpr1")
	db.DeleteUser("expruser")
}

func TestGetExpressionByIdHandler_Success(t *testing.T) {
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	password, err := db.GenerateHash("loginpass")
	if err != nil {
		t.Fatalf("Error while generating hash: %v", err)
	}
	user := &db.User{
		Name:     "exprbyiduser",
		Password: password,
	}

	_, err = db.InsertUser(user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	nodes := []*calc.Node{
		{ID: "a1", ExprID: "testexpr2", Type: "number", Status: "done", Result: 2},
		{ID: "a2", ExprID: "testexpr2", Type: "number", Status: "done", Result: 3},
		{
			ID:        "a3",
			ExprID:    "testexpr2",
			Type:      "operation",
			Operation: "+",
			Left:      "a1",
			Right:     "a2",
			Status:    "done",
		},
	}

	_, err = db.InsertNodes(nodes)
	if err != nil {
		t.Fatalf("Failed to insert test nodes: %v", err)
	}

	expr := db.Expression{
		ExprID:     "testexpr2",
		Username:   "exprbyiduser",
		Status:     "done",
		RootNodeID: "a3",
		Expr:       "2 + 3",
		Result:     5,
	}
	_, err = db.InsertExpression(expr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/expressions/testexpr2", nil)
	req.Header.Set("username", "exprbyiduser")
	req.SetPathValue("id", "testexpr2")
	w := httptest.NewRecorder()

	GetExpressionByIdHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	var response struct {
		Expression ExpressionStatus `json:"expression"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Error decoding response: %v", err)
	}

	if response.Expression.ID != "testexpr2" {
		t.Errorf("Expected expression ID %s, got %s", "testexpr2", response.Expression.ID)
	}

	clearState("testexpr2")
	db.DeleteUser("exprbyiduser")
}

func TestGetExpressionByIdHandler_Unauthorized(t *testing.T) {
	err := db.Init("../../data/store.db")
	if err != nil {
		t.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Stop()

	nodes := []*calc.Node{
		{ID: "a1", ExprID: "testexpr2", Type: "number", Status: "done", Result: 2},
		{ID: "a2", ExprID: "testexpr2", Type: "number", Status: "done", Result: 3},
		{
			ID:        "a3",
			ExprID:    "testexpr2",
			Type:      "operation",
			Operation: "+",
			Left:      "a1",
			Right:     "a2",
			Status:    "done",
		},
	}

	_, err = db.InsertNodes(nodes)
	if err != nil {
		t.Fatalf("Failed to insert test nodes: %v", err)
	}

	expr := db.Expression{
		ExprID:     "testexpr2",
		Username:   "exprbyiduser",
		Status:     "done",
		RootNodeID: "a3",
		Expr:       "2 + 3",
		Result:     5,
	}
	_, err = db.InsertExpression(expr)
	if err != nil {
		t.Fatalf("Failed to create test expression: %v", err)
	}

	req := httptest.NewRequest("GET", "/api/v1/expressions/testexpr2", nil)
	req.Header.Set("username", "fictionuser")
	req.SetPathValue("id", "testexpr2")
	w := httptest.NewRecorder()

	GetExpressionByIdHandler(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status code %d, got %d", http.StatusUnauthorized, w.Code)
	}

	clearState("testexpr2")
}
