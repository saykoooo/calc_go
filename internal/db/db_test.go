package db

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/saykoooo/calc_go/internal/calc"
)

var testDB *sql.DB

func setupTestDB() error {
	var err error
	testDB, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		return err
	}

	ctx = context.TODO()
	db = testDB

	err = createTables(ctx, testDB)
	if err != nil {
		return err
	}
	return nil
}

func TestMain(m *testing.M) {
	if err := setupTestDB(); err != nil {
		os.Exit(1)
	}

	code := m.Run()

	db.Close()
	os.Exit(code)
}

func TestUserOperations(t *testing.T) {
	user := &User{
		Name:           "testuser",
		OriginPassword: "password123",
	}

	hash, err := GenerateHash(user.OriginPassword)
	if err != nil {
		t.Fatal("Failed to generate hash")
	}
	user.Password = hash

	id, err := InsertUser(user)
	if err != nil {
		t.Fatal("Failed to insert user:", err)
	}

	if id <= 0 {
		t.Error("Expected positive user ID")
	}

	storedUser, err := SelectUser(user.Name)
	fmt.Println("storedUser:", storedUser)
	if err != nil {
		t.Fatal("Failed to select user:", err)
	}

	if storedUser.ID != id {
		t.Errorf("Expected ID %d, got %d", id, storedUser.ID)
	}

	err = user.ComparePassword(storedUser)
	if err != nil {
		t.Error("Password comparison failed:", err)
	}
}

func TestExpressionOperations(t *testing.T) {
	expr := Expression{
		ExprID:     "expr1",
		Username:   "testuser",
		Status:     "processing",
		RootNodeID: "node1",
		Result:     0,
	}

	id, err := InsertExpression(expr)
	if err != nil {
		t.Fatal("Failed to insert expression:", err)
	}

	if id <= 0 {
		t.Error("Expected positive expression ID")
	}

	storedExpr, err := SelectExpression(expr.ExprID)
	if err != nil {
		t.Fatal("Failed to select expression:", err)
	}

	if storedExpr.Username != expr.Username {
		t.Errorf("Expected username %s, got %s", expr.Username, storedExpr.Username)
	}

	err = SetExpressionStatus(expr.ExprID, "done")
	if err != nil {
		t.Error("Failed to set expression status:", err)
	}

	err = SetExpressionResult(expr.ExprID, 42.5)
	if err != nil {
		t.Error("Failed to set expression result:", err)
	}

	updatedExpr, err := SelectExpression(expr.ExprID)
	if err != nil {
		t.Fatal("Failed to select updated expression:", err)
	}

	if updatedExpr.Status != "done" {
		t.Errorf("Expected status 'done', got %s", updatedExpr.Status)
	}

	if updatedExpr.Result != 42.5 {
		t.Errorf("Expected result 42.5, got %f", updatedExpr.Result)
	}
}

func TestNodeOperations(t *testing.T) {
	nodes := []*calc.Node{
		{
			ID:     "node1",
			ExprID: "expr1",
			Type:   "number",
			Status: "done",
			Result: 2,
		},
		{
			ID:        "node2",
			ExprID:    "expr1",
			Type:      "operation",
			Operation: "+",
			Left:      "node1",
			Right:     "node1",
			Status:    "pending",
		},
	}

	count, err := InsertNodes(nodes)
	if err != nil {
		t.Fatal("Failed to insert nodes:", err)
	}

	if count != int64(len(nodes)) {
		t.Errorf("Expected %d nodes inserted, got %d", len(nodes), count)
	}

	storedNode, err := SelectNode("node1")
	if err != nil {
		t.Fatal("Failed to select node:", err)
	}

	if storedNode.Result != 2 {
		t.Errorf("Expected result 2, got %f", storedNode.Result)
	}

	_, err = SetNodeStatus("node2", "in_progress")
	if err != nil {
		t.Error("Failed to set node status:", err)
	}

	err = SetNodeResult("node2", 4)
	if err != nil {
		t.Error("Failed to set node result:", err)
	}

	task, err := SelectNodeAsTask()
	if err == nil {
		t.Error("Expected no available tasks")
		fmt.Println("Got task:", task)
	}

	err = DeleteNodes("expr1")
	if err != nil {
		t.Error("Failed to delete nodes:", err)
	}
}

func TestErrorHandling(t *testing.T) {
	// 404 expr
	_, err := SelectExpression("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent expression")
	}

	// 404 node
	num, err := SetNodeStatus("nonexistent", "done")
	if err != nil && num != 0 {
		t.Error("Expected error for nonexistent node")
	}

	// try dupe user
	user := &User{
		Name:           "testuser2",
		OriginPassword: "pass",
	}
	hash, _ := GenerateHash(user.OriginPassword)
	user.Password = hash

	_, err = InsertUser(user)   // Первый раз
	num, err = InsertUser(user) // Второй раз
	if err == nil && num != 0 {
		t.Error("Expected error for duplicate user")
	}
}
