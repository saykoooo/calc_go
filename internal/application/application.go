package application

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/saykoooo/calc_go/internal/calc"
	"github.com/saykoooo/calc_go/internal/db"
	"github.com/saykoooo/calc_go/proto"
	"google.golang.org/grpc"

	"github.com/golang-jwt/jwt/v5"
)

type Config struct {
	Addr               string
	GRPC               string
	JwtSecret          string
	JwtExpiration      time.Duration
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
	ID         string  `json:"id"`
	Status     string  `json:"status"`
	Result     float64 `json:"result"`
	Expression string  `json:"expression"`
}

type grpcServer struct {
	app *Application
	proto.UnimplementedOrchestratorServer
}

var (
	mu sync.Mutex
)

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		log.Println("Missing PORT environment variable. Using default value: 8080")
		config.Addr = "8080"
	}
	config.GRPC = os.Getenv("GRPC_PORT")
	if config.GRPC == "" {
		log.Println("Missing GRPC_PORT environment variable. Using default value: 5000")
		config.GRPC = "5000"
	}
	config.JwtSecret = os.Getenv("JWT_SECRET")
	if config.JwtSecret == "" {
		config.JwtSecret = "dumbSecretForLaziest"
	}
	config.JwtExpiration = 5 * time.Minute
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

func (s *grpcServer) GetTask(ctx context.Context, req *proto.GetTaskRequest) (*proto.TaskResponse, error) {
	task, err := db.SelectNodeAsTask()
	if task.ID == "" || err != nil {
		return nil, fmt.Errorf("no task available")
	}
	var opTime time.Duration
	switch task.Oper {
	case "+":
		opTime = s.app.config.TimeAddition
	case "-":
		opTime = s.app.config.TimeSubtraction
	case "*":
		opTime = s.app.config.TimeMultiplication
	case "/":
		opTime = s.app.config.TimeDivision
	default:
		return nil, fmt.Errorf("invalid operation")
	}
	return &proto.TaskResponse{
		Id:            task.ID,
		Arg1:          task.Arg1,
		Arg2:          task.Arg2,
		Operation:     task.Oper,
		OperationTime: int32(opTime.Milliseconds()),
	}, nil
}

func (s *grpcServer) SubmitResult(ctx context.Context, req *proto.ResultRequest) (*proto.SubmitResultResponse, error) {
	mu.Lock()
	defer mu.Unlock()

	err := db.SetNodeResult(req.Id, req.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to set node result")
	}

	node, err := db.SelectNode(req.Id)
	if err != nil {
		return nil, fmt.Errorf("failed to get node")
	}

	expr, err := db.SelectExpression(node.ExprID)
	if err != nil {
		return nil, fmt.Errorf("failed to get expression")
	}

	if node.ID == expr.RootNodeID {
		db.SetExpressionResult(expr.ExprID, req.Result)
		db.DeleteNodes(expr.ExprID)
	}

	return &proto.SubmitResultResponse{}, nil
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func (a *Application) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bearer, err := ExtractToken(r)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		log.Printf("Got bearer: %s", bearer)

		tokenFromString, err := jwt.Parse(bearer, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(a.config.JwtSecret), nil
		})
		if err != nil {
			log.Println(err)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		claims, ok := tokenFromString.Claims.(jwt.MapClaims)
		if ok {
			log.Println("Request from user: ", claims["name"])
		} else {
			log.Println("invalid jwt token")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		r.Header.Set("username", claims["name"].(string))
		next.ServeHTTP(w, r)
	})
}

// func (a *Application) GetTaskHandler(w http.ResponseWriter, r *http.Request) {
// 	mu.Lock()
// 	defer mu.Unlock()
// 	log.Println("Searching for tasks...")
// 	task, err := db.SelectNodeAsTask()
// 	if task.ID == "" {
// 		log.Println("No pending tasks available")
// 		http.Error(w, "no task", http.StatusNotFound)
// 		return
// 	}
// 	if err != nil {
// 		log.Printf("Error while getting task from db")
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}
// 	if task.Oper == "/" && task.Arg2 == 0 {
// 		log.Printf("Division by zero in task %s", task.ID)
// 		_, err = db.SetNodeStatus(task.ID, "error")
// 		if err != nil {
// 			log.Printf("Error changing status for node %s", task.ID)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}
// 		err = db.SetExpressionStatus(task.ExprID, "error")
// 		if err != nil {
// 			log.Printf("Error changing status for expression %s", task.ExprID)
// 		}
// 		log.Printf("Task %s contains division by zero", task.ID)
// 		err = db.DeleteNodes(task.ExprID)
// 		if err != nil {
// 			log.Printf("Error deleting nodes for expression %s", task.ExprID)
// 		}
// 		http.Error(w, "division by zero", http.StatusInternalServerError)
// 		return
// 	}
// 	var opTime time.Duration
// 	switch task.Oper {
// 	case "+":
// 		opTime = a.config.TimeAddition
// 	case "-":
// 		opTime = a.config.TimeSubtraction
// 	case "*":
// 		opTime = a.config.TimeMultiplication
// 	case "/":
// 		opTime = a.config.TimeDivision
// 	default:
// 		log.Printf("Invalid operation %s in task %s", task.Oper, task.ID)
// 		http.Error(w, "invalid operation", http.StatusInternalServerError)
// 		return
// 	}
// 	_, err = db.SetNodeStatus(task.ID, "in_progress")
// 	if err != nil {
// 		log.Printf("Error changing status for node %s", task.ID)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}
// 	log.Printf("Task %s marked as in_progress", task.ID)
// 	json.NewEncoder(w).Encode(map[string]interface{}{
// 		"task": map[string]interface{}{
// 			"id":             task.ID,
// 			"arg1":           task.Arg1,
// 			"arg2":           task.Arg2,
// 			"operation":      task.Oper,
// 			"operation_time": opTime.Milliseconds(),
// 		},
// 	})
// }

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	log.Printf("Got registration request from: %s", req.Login)
	password, err := db.GenerateHash(req.Password)
	if err != nil {
		log.Printf("Error while generating hash: %v", err)
		http.Error(w, "Error while generating hash", http.StatusInternalServerError)
		return
	}
	user := &db.User{
		Name:           req.Login,
		Password:       password,
		OriginPassword: req.Password,
	}
	userID, err := db.InsertUser(user)
	if err != nil {
		log.Printf("Error while registering user: %s", req.Login)
		http.Error(w, "Error while registering user", http.StatusInternalServerError)
		return
	} else if userID == 0 {
		log.Printf("User already exists: %s", req.Login)
		http.Error(w, "User already exists", http.StatusBadRequest)
		return
	} else {
		user.ID = userID
		log.Printf("User registered: %s (id:%v)", req.Login, user.ID)
	}
	w.WriteHeader(http.StatusOK)
}

func ExtractToken(req *http.Request) (string, error) {
	tokenHeader := req.Header.Get("Authorization")
	if len(tokenHeader) < 7 || !strings.EqualFold(tokenHeader[:7], "bearer ") {
		return "", fmt.Errorf("Invalid authorization header")
	}
	return tokenHeader[7:], nil
}

func (a *Application) LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Login    string `json:"login"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Invalid request body: %v", err)
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	userFromDB, err := db.SelectUser(req.Login)
	if err != nil {
		log.Printf("Error while getting user %s: %v", req.Login, err)
		http.Error(w, "Auth failed", http.StatusUnauthorized)
		return
	}
	password, err := db.GenerateHash(req.Password)
	if err != nil {
		log.Printf("Error while generating hash: %v", err)
		http.Error(w, "Error while generating hash", http.StatusInternalServerError)
		return
	}
	user := &db.User{
		Name:           req.Login,
		Password:       password,
		OriginPassword: req.Password,
	}
	if ok := user.ComparePassword(userFromDB); ok != nil {
		log.Printf("Auth failed: %s", user.Name)
		http.Error(w, "Auth failed", http.StatusUnauthorized)
		return
	}
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"name": req.Login,
		"nbf":  now.Unix(),
		"exp":  now.Add(a.config.JwtExpiration).Unix(),
		"iat":  now.Unix(),
	})
	tokenString, err := token.SignedString([]byte(a.config.JwtSecret))
	if err != nil {
		log.Printf("Error while generating token string: %s", err)
		http.Error(w, "Error while generating token string", http.StatusInternalServerError)
		return
	}
	log.Printf("Token generated for user: %s", req.Login)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"token":      tokenString,
		"expires_in": strconv.FormatInt(int64(a.config.JwtExpiration.Seconds()), 10),
	})
}

// func PostTaskHandler(w http.ResponseWriter, r *http.Request) {
// 	var req struct {
// 		ID     string  `json:"id"`
// 		Result float64 `json:"result"`
// 	}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		log.Printf("Invalid request body: %v", err)
// 		http.Error(w, "invalid request", http.StatusBadRequest)
// 		return
// 	}
// 	mu.Lock()
// 	defer mu.Unlock()
// 	err := db.SetNodeResult(req.ID, req.Result)
// 	if err != nil {
// 		log.Printf("Error setting result for node %s", req.ID)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}
// 	log.Printf("Task %s completed with result %f", req.ID, req.Result)
// 	node, err := db.SelectNode(req.ID)
// 	if err != nil {
// 		log.Printf("Error getting node %s", req.ID)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}
// 	expr, err := db.SelectExpression(node.ExprID)
// 	if err != nil {
// 		log.Printf("Error getting expression %s", node.ExprID)
// 		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 		return
// 	}
// 	if node.ID == expr.RootNodeID {
// 		err = db.SetExpressionResult(expr.ExprID, req.Result)
// 		if err != nil {
// 			log.Printf("Error setting result for expression %s", expr.ExprID)
// 			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
// 			return
// 		}
// 		log.Printf("Expression %s completed with result %f", expr.ExprID, req.Result)
// 		err := db.DeleteNodes(expr.ExprID)
// 		if err != nil {
// 			log.Printf("Error deleting used nodes for expr: %s", expr.ExprID)
// 		}
// 	}
// 	w.WriteHeader(http.StatusOK)
// }

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

	user := r.Header.Get("username")

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

	log.Printf("Calculate expression for user: %s", user)
	exprID := calc.GenerateID()
	mu.Lock()
	defer mu.Unlock()

	expr := db.Expression{
		ExprID:     exprID,
		Username:   user,
		Status:     "processing",
		RootNodeID: root.ID,
		Expr:       request.Expression,
	}
	expr_num, err := db.InsertExpression(expr)
	if err != nil {
		log.Printf("Error adding expression to db: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	for i := range result {
		result[i].ExprID = exprID
	}

	num, err := db.InsertNodes(result)
	if err != nil {
		log.Printf("Error saving nodes to db: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	log.Printf("Expression with ID %s created and processing started", exprID)

	log.Printf("Nodes added: %d / Expr added: %d", num, expr_num)
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": exprID})
}

func GetExpressionsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	user := r.Header.Get("username")

	exprs, err := db.SelectExpressionsByUser(user)
	if err != nil {
		log.Printf("Error while getting expressions: %v", err)
	}

	response := struct {
		Expressions []ExpressionStatus `json:"expressions"`
	}{
		Expressions: make([]ExpressionStatus, 0, len(exprs)),
	}

	for _, expr := range exprs {
		response.Expressions = append(response.Expressions, ExpressionStatus{
			ID:         expr.ExprID,
			Status:     expr.Status,
			Result:     expr.Result,
			Expression: expr.Expr,
		})
	}

	log.Println("Returning list of expressions")
	w.Header().Set("Content-Type", "application/json")
	log.Println("Expressions: ", len(exprs))
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

	expr, err := db.SelectExpression(id)
	if err != nil {
		log.Printf("Error while getting expression (%s): %v", id, err)
		http.Error(w, "Expression not found", http.StatusNotFound)
		return
	}
	user := r.Header.Get("username")
	if user != expr.Username {
		log.Printf("Invalid username: %s, expect: %s", user, expr.Username)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	log.Printf("Returning expression with ID %s", id)
	response := struct {
		Expression ExpressionStatus `json:"expression"`
	}{
		Expression: ExpressionStatus{
			ID:         expr.ExprID,
			Status:     expr.Status,
			Result:     expr.Result,
			Expression: expr.Expr,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (a *Application) RunGRPCServer() error {
	lis, err := net.Listen("tcp", ":"+a.config.GRPC)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	server := grpc.NewServer()
	proto.RegisterOrchestratorServer(server, &grpcServer{app: a})

	log.Printf("Starting gRPC server on port %s", a.config.GRPC)
	return server.Serve(lis)
}

func (a *Application) RunServer() error {
	mux := http.NewServeMux()
	fs := http.FileServer(http.Dir("web/"))
	mux.Handle("/web/", http.StripPrefix("/web/", fs))
	mux.Handle("/", LoggingMiddleware(http.HandlerFunc(NotFoundHandler)))
	mux.Handle("/api/v1/calculate", LoggingMiddleware(a.AuthMiddleware(http.HandlerFunc(CalcHandler))))
	mux.Handle("/api/v1/expressions", LoggingMiddleware(a.AuthMiddleware(http.HandlerFunc(GetExpressionsHandler))))
	mux.Handle("/api/v1/expressions/{id}", LoggingMiddleware(a.AuthMiddleware(http.HandlerFunc(GetExpressionByIdHandler))))
	// mux.Handle("GET /internal/task", LoggingMiddleware(http.HandlerFunc(a.GetTaskHandler)))
	// mux.Handle("POST /internal/task", LoggingMiddleware(http.HandlerFunc(PostTaskHandler)))
	mux.Handle("POST /api/v1/register", LoggingMiddleware(http.HandlerFunc(RegisterHandler)))
	mux.Handle("POST /api/v1/login", LoggingMiddleware(http.HandlerFunc(a.LoginHandler)))
	log.Printf("Web server run on port: %s\n", a.config.Addr)
	err := db.Init("data/store.db")
	if err != nil {
		log.Panicln(err)
	}
	defer db.Stop()
	return http.ListenAndServe(":"+a.config.Addr, mux)
}
