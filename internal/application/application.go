package application

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/saykoooo/calc_go/internal/calc"
)

type Config struct {
	Addr               string
	TimeAddition       time.Duration
	TimeSubtraction    time.Duration
	TimeMultiplication time.Duration
	TimeDivision       time.Duration
}

type Expression struct {
	ID         string
	Status     string
	RootNodeID string
	Result     float64
}

type ExpressionStatus struct {
	ID     string  `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

var (
	expressions = make(map[string]*Expression)
	nodes       = make(map[string]*calc.Node)
	mu          sync.Mutex
)

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		log.Println("Missing PORT environment variable. Using default value: 8080")
		config.Addr = "8080"
	}
	config.TimeAddition = getEnvDuration("TIME_ADDITION_MS", 1000)
	config.TimeSubtraction = getEnvDuration("TIME_SUBTRACTION_MS", 1000)
	config.TimeMultiplication = getEnvDuration("TIME_MULTIPLICATION_MS", 1000)
	config.TimeDivision = getEnvDuration("TIME_DIVISION_MS", 1000)
	return config
}

func getEnvDuration(name string, defVal int) time.Duration {
	val := os.Getenv(name)
	if val == "" {
		return time.Duration(defVal) * time.Millisecond
	}
	num, err := strconv.Atoi(val)
	if err != nil {
		log.Printf("Invalid value for %s: %s. Using default value %d ms", name, val, defVal)
		return time.Duration(defVal) * time.Millisecond
	}
	return time.Duration(num) * time.Millisecond
}

type Application struct {
	config *Config
}

func New() *Application {
	return &Application{
		config: ConfigFromEnv(),
	}
}

type Request struct {
	Expression string `json:"expression"`
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (a *Application) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	log.Println("Searching for tasks...")
	var task *calc.Node
	for _, node := range nodes {
		if node.Type == "operation" && node.Status == "pending" {
			left, existsLeft := nodes[node.Left]
			right, existsRight := nodes[node.Right]

			if existsLeft && existsRight && left.Status == "done" && right.Status == "done" {
				task = node
				break
			}
		}
	}

	if task == nil {
		log.Println("No pending tasks available")
		http.Error(w, "no task", http.StatusNotFound)
		return
	}

	log.Printf("Found task with ID %s: %s %f %s %f", task.ID, task.Operation, nodes[task.Left].Result, task.Operation, nodes[task.Right].Result)

	left, existsLeft := nodes[task.Left]
	right, existsRight := nodes[task.Right]

	if !existsLeft || !existsRight {
		log.Printf("Invalid node reference for task %s", task.ID)
		http.Error(w, "invalid node reference", http.StatusInternalServerError)
		return
	}

	arg1 := left.Result
	arg2 := right.Result

	if task.Operation == "/" && arg2 == 0 {
		log.Printf("Division by zero in task %s", task.ID)
		task.Status = "error"
		expr := expressions[task.ExprID]
		expr.Status = "error"
		http.Error(w, "division by zero", http.StatusInternalServerError)
		return
	}

	var opTime time.Duration
	switch task.Operation {
	case "+":
		opTime = a.config.TimeAddition
	case "-":
		opTime = a.config.TimeSubtraction
	case "*":
		opTime = a.config.TimeMultiplication
	case "/":
		opTime = a.config.TimeDivision
	default:
		log.Printf("Invalid operation %s in task %s", task.Operation, task.ID)
		http.Error(w, "invalid operation", http.StatusInternalServerError)
		return
	}

	task.Status = "in_progress"
	log.Printf("Task %s marked as in_progress", task.ID)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"task": map[string]interface{}{
			"id":             task.ID,
			"arg1":           arg1,
			"arg2":           arg2,
			"operation":      task.Operation,
			"operation_time": opTime.Milliseconds(),
		},
	})
}

func PostTaskHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ID     string  `json:"id"`
		Result float64 `json:"result"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock()

	node, exists := nodes[req.ID]
	if !exists {
		log.Printf("Task with ID %s not found", req.ID)
		http.Error(w, "task not found", http.StatusNotFound)
		return
	}

	node.Result = req.Result
	node.Status = "done"
	log.Printf("Task %s completed with result %f", req.ID, req.Result)

	expr := expressions[node.ExprID]
	if node.ID == expr.RootNodeID {
		expr.Result = req.Result
		expr.Status = "done"
		log.Printf("Expression %s completed with result %f", expr.ID, req.Result)
	}

	w.WriteHeader(http.StatusOK)
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/api/v1/calculate" {
		http.Error(w, "Bad URL", http.StatusNotFound)
		return
	}
}

func CalcHandler(w http.ResponseWriter, r *http.Request) {
	request := new(Request)
	defer r.Body.Close()
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := json.NewDecoder(r.Body).Decode(&request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	root, result, err := calc.ParseExpression(request.Expression)
	if err != nil {
		log.Printf("Error parsing expression: %v", err)
		http.Error(w, "invalid expression", http.StatusUnprocessableEntity)
		return
	}

	exprID := calc.GenerateID()
	mu.Lock()
	defer mu.Unlock()

	expressions[exprID] = &Expression{
		ID:         exprID,
		Status:     "processing",
		RootNodeID: root.ID,
	}

	for _, n := range result {
		n.ExprID = exprID
		nodes[n.ID] = n
	}
	log.Println("Nodes after saving:")
	for _, n := range nodes {
		log.Printf("Node ID: %s, Type: %s, Status: %s, Left: %s, Right: %s", n.ID, n.Type, n.Status, n.Left, n.Right)
	}
	log.Printf("Expression with ID %s created and processing started", exprID)

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": exprID})
}

func GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	response := struct {
		Expressions []ExpressionStatus `json:"expressions"`
	}{
		Expressions: make([]ExpressionStatus, 0, len(expressions)),
	}

	for _, expr := range expressions {
		response.Expressions = append(response.Expressions, ExpressionStatus{
			ID:     expr.ID,
			Status: expr.Status,
			Result: expr.Result,
		})
	}

	log.Println("Returning list of expressions")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func GetExpressionByIdHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	mu.Lock()
	defer mu.Unlock()

	expr, exists := expressions[id]
	if !exists {
		log.Printf("Expression with ID %s not found", id)
		http.Error(w, "expression not found", http.StatusNotFound)
		return
	}

	log.Printf("Returning expression with ID %s", id)
	response := struct {
		Expression ExpressionStatus `json:"expression"`
	}{
		Expression: ExpressionStatus{
			ID:     expr.ID,
			Status: expr.Status,
			Result: expr.Result,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (a *Application) RunServer() error {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("web/"))
	mux.Handle("/web/", http.StripPrefix("/web/", fs))
	mux.Handle("/", LoggingMiddleware(http.HandlerFunc(NotFoundHandler)))
	mux.Handle("/api/v1/calculate", LoggingMiddleware(http.HandlerFunc(CalcHandler)))
	mux.Handle("/api/v1/expressions", LoggingMiddleware(http.HandlerFunc(GetExpressionsHandler)))
	mux.Handle("/api/v1/expression/{id}", LoggingMiddleware(http.HandlerFunc(GetExpressionByIdHandler)))
	mux.Handle("GET /internal/task", LoggingMiddleware(http.HandlerFunc(a.GetTaskHandler)))
	mux.Handle("POST /internal/task", LoggingMiddleware(http.HandlerFunc(PostTaskHandler)))
	log.Printf("Web server run on port: %s\n", a.config.Addr)
	return http.ListenAndServe(":"+a.config.Addr, mux)
}
