package db

import (
	"context"
	"database/sql"
	"log"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"github.com/saykoooo/calc_go/internal/calc"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int64
	Name           string
	Password       string
	OriginPassword string
}

type Task struct {
	ID     string
	ExprID string
	Oper   string
	Arg1   float64
	Arg2   float64
}

type Expression struct {
	ExprID     string
	Username   string
	Status     string
	RootNodeID string
	Result     float64
}

var (
	ctx context.Context
	db  *sql.DB
	mu  sync.Mutex
	nu  sync.Mutex
	eu  sync.Mutex
)

func (u User) ComparePassword(u2 User) error {
	err := compare(u2.Password, u.OriginPassword)
	if err != nil {
		log.Println("DB: Auth fail")
		return err
	}
	log.Println("DB: Auth success")
	return nil
}

func createTables(ctx context.Context, db *sql.DB) error {
	const userTable = `
	CREATE TABLE IF NOT EXISTS users(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE,
		password TEXT
	);
	`

	if _, err := db.ExecContext(ctx, userTable); err != nil {
		return err
	}

	const nodeTable = `
	CREATE TABLE IF NOT EXISTS nodes(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT,
		expr_id TEXT,
		type TEXT,
		l_id TEXT,
		r_id TEXT,
		oper TEXT,
		status TEXT,
		result REAL           
	);
	`
	if _, err := db.ExecContext(ctx, nodeTable); err != nil {
		return err
	}

	const expressionTable = `
	CREATE TABLE IF NOT EXISTS expressions(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		expr_id TEXT,
		username TEXT,
		status TEXT,
		root_node_id INTEGER,
		result REAL           
	);
	`
	if _, err := db.ExecContext(ctx, expressionTable); err != nil {
		return err
	}

	return nil
}

func InsertExpression(expr Expression) (int64, error) {
	var q = `
	INSERT INTO expressions (expr_id, username, status, root_node_id, result) values ($1, $2, $3, $4, $5)
	`
	eu.Lock()
	defer eu.Unlock()
	result, err := db.ExecContext(ctx, q, expr.ExprID, expr.Username, expr.Status, expr.RootNodeID, expr.Result)
	if err != nil {
		return 0, nil
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func SetExpressionStatus(expr_id string, status string) error {
	q := "UPDATE expressions SET status=$1 WHERE expr_id=$2"

	eu.Lock()
	defer eu.Unlock()
	result, err := db.ExecContext(ctx, q, status, expr_id)

	if err != nil {
		log.Println("DB: Error updating expression: ", err)
		return err
	}
	num, _ := result.RowsAffected()
	log.Println("DB: Expression status updated: ", num)
	return nil
}

func SelectExpression(expr_id string) (Expression, error) {
	var (
		expr Expression
		err  error
	)

	eu.Lock()
	defer eu.Unlock()
	var q = `
	SELECT expr_id, username, status, root_node_id, result
	FROM expressions 
	WHERE expr_id = $1
	`
	err = db.QueryRowContext(ctx, q, expr_id).Scan(&expr.ExprID, &expr.Username, &expr.Status, &expr.RootNodeID, &expr.Result)
	if err != nil {
		log.Printf("DB: SelectExpression error: %v", err)
	}
	return expr, err
}

func SelectExpressionsByUser(username string) ([]Expression, error) {
	var (
		expr []Expression
		err  error
	)

	eu.Lock()
	defer eu.Unlock()
	var q = `
	SELECT expr_id, username, status, root_node_id, result
	FROM expressions 
	WHERE username = $1
	`
	rows, err := db.QueryContext(ctx, q, username)
	if err != nil {
		log.Printf("DB: SelectExpressionsByUser error: %v", err)
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var ex Expression
		if err := rows.Scan(&ex.ExprID, &ex.Username, &ex.Status, &ex.RootNodeID, &ex.Result); err != nil {
			log.Printf("DB: SelectExpressionsByUser::Scan error: %v", err)
			return expr, err
		}
		expr = append(expr, ex)
	}
	if err = rows.Err(); err != nil {
		log.Printf("DB: SelectExpressionsByUser::Err error: %v", err)
		return expr, err
	}
	return expr, nil
}

func SetExpressionResult(expr_id string, payload float64) error {
	q := `UPDATE expressions SET status="done", result=$1 WHERE expr_id=$2`
	nu.Lock()
	defer nu.Unlock()
	result, err := db.ExecContext(ctx, q, payload, expr_id)

	if err != nil {
		log.Println("DB: Error updating expression: ", err)
		return err
	}
	num, _ := result.RowsAffected()
	log.Println("DB: Expression updated: ", num)
	return nil
}

func DeleteExpression(expr_id string) error {
	q := "DELETE FROM expressions WHERE	expr_id=$1"

	eu.Lock()
	defer eu.Unlock()
	result, err := db.ExecContext(ctx, q, expr_id)

	if err != nil {
		log.Println("DB: Error deleting Expression: ", err)
		return err
	}
	num, _ := result.RowsAffected()
	log.Println("DB: Expression deleted: ", num)
	return nil
}

func InsertUser(user *User) (int64, error) {
	var q = `
	INSERT INTO users (name, password) values ($1, $2)
	`
	mu.Lock()
	defer mu.Unlock()
	result, err := db.ExecContext(ctx, q, user.Name, user.Password)
	if err != nil {
		return 0, nil
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}
	return id, nil
}

func InsertNodes(nodes []*calc.Node) (int64, error) {
	q := "INSERT INTO nodes(node_id,	expr_id, type, l_id, r_id,	oper, status, result) VALUES "
	vals := []interface{}{}

	for _, row := range nodes {
		q += "(?, ?, ?, ?, ?, ?, ?, ?),"
		vals = append(vals, row.ID, row.ExprID, row.Type, row.Left, row.Right, row.Operation, row.Status, row.Result)
	}
	q = q[0 : len(q)-1]
	stmt, _ := db.Prepare(q)

	nu.Lock()
	defer nu.Unlock()
	result, err := stmt.ExecContext(ctx, vals...)

	if err != nil {
		log.Println("DB: Error inserting nodes: ", err)
		return 0, err
	}

	return result.RowsAffected()
}

func DeleteNodes(expr_id string) error {
	q := "DELETE FROM nodes WHERE	expr_id=$1"

	nu.Lock()
	defer nu.Unlock()
	result, err := db.ExecContext(ctx, q, expr_id)

	if err != nil {
		log.Println("DB: Error deleting nodes: ", err)
		return err
	}
	num, _ := result.RowsAffected()
	log.Println("DB: Nodes deleted: ", num)
	return nil
}

func SelectNode(id string) (calc.Node, error) {
	var (
		node calc.Node
		err  error
	)

	nu.Lock()
	defer nu.Unlock()
	var q = `
	SELECT node_id, expr_id, type, l_id, r_id, oper, status, result 
	FROM nodes 
	WHERE node_id = $1
	`
	err = db.QueryRowContext(ctx, q, id).Scan(&node.ID, &node.ExprID, &node.Type, &node.Left,
		&node.Right, &node.Operation, &node.Status, &node.Result)
	if err != nil {
		log.Printf("DB: SelectNode error: %v", err)
	}
	return node, err
}

func SelectNodeAsTask() (Task, error) {
	var (
		task Task
		err  error
	)

	nu.Lock()
	defer nu.Unlock()

	var q = `
	SELECT N.node_id, N.expr_id, N.oper, L.result AS arg1, R.result AS arg2 
	FROM nodes AS N
	JOIN nodes AS L ON N.l_id = L.node_id
	JOIN nodes as R ON N.r_id = R.node_id
	WHERE N.type = "operation" AND N.status = "pending" AND L.status = "done" AND R.status = "done"
	LIMIT 1
	`
	err = db.QueryRowContext(ctx, q).Scan(&task.ID, &task.ExprID, &task.Oper, &task.Arg1, &task.Arg2)
	return task, err
}

func SetNodeStatus(node_id string, status string) (int64, error) {
	q := "UPDATE nodes SET status=$1 WHERE node_id=$2"

	nu.Lock()
	defer nu.Unlock()
	result, err := db.ExecContext(ctx, q, status, node_id)

	if err != nil {
		log.Println("DB: Error updating node: ", err)
		return 0, err
	}
	// num, _ := result.RowsAffected()
	// log.Println("DB: Node status updated: ", num)
	return result.RowsAffected()
}

func SetNodeResult(node_id string, payload float64) error {
	q := `UPDATE nodes SET status="done", result=$1 WHERE node_id=$2`
	nu.Lock()
	defer nu.Unlock()
	result, err := db.ExecContext(ctx, q, payload, node_id)

	if err != nil {
		log.Println("DB: Error updating node: ", err)
		return err
	}
	num, _ := result.RowsAffected()
	log.Println("DB: Node updated: ", num)
	return nil
}

func SelectUser(name string) (User, error) {
	var (
		user User
		err  error
	)

	mu.Lock()
	defer mu.Unlock()

	var q = "SELECT id, name, password FROM users WHERE name=$1"
	err = db.QueryRowContext(ctx, q, name).Scan(&user.ID, &user.Name, &user.Password)
	return user, err
}

func GenerateHash(s string) (string, error) {
	saltedBytes := []byte(s)
	hashedBytes, err := bcrypt.GenerateFromPassword(saltedBytes, bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	hash := string(hashedBytes[:])
	return hash, nil
}

func compare(hash string, s string) error {
	incoming := []byte(s)
	existing := []byte(hash)
	return bcrypt.CompareHashAndPassword(existing, incoming)
}

func Init(db_file string) error {
	ctx = context.TODO()
	var err error

	db, err = sql.Open("sqlite3", db_file)
	if err != nil {
		return err
	}

	err = db.PingContext(ctx)
	if err != nil {
		return err
	}

	if err = createTables(ctx, db); err != nil {
		return err
	}

	return nil
}

func Stop() {
	db.Close()
}
