package application

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Config struct {
	Addr               string
	TimeAddition       time.Duration
	TimeSubtraction    time.Duration
	TimeMultiplication time.Duration
	TimeDivision       time.Duration
}

// type RespOk struct {
// 	Result string `json:"result"`
// }

// type RespError struct {
// 	Error string `json:"error"`
// }

// Токен для парсинга
type Token struct {
	Type  string // num, op, paren
	Value string
	Num   float64
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

type Node struct {
	ID        string
	ExprID    string
	Type      string
	Value     float64
	Left      string
	Right     string
	Operation string
	Status    string
	Result    float64
	Parents   []string
}

var (
	expressions = make(map[string]*Expression)
	nodes       = make(map[string]*Node)
	mu          sync.Mutex
	idCounter   int
	idMutex     sync.Mutex
)

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
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
	root, result, err := parseExpression(request.Expression)
	if err != nil {
		log.Printf("Error parsing expression: %v", err)
		http.Error(w, "invalid expression", http.StatusUnprocessableEntity)
		return
	}

	exprID := generateID()
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

func (a *Application) RunServer() error {
	mux := http.NewServeMux()
	mux.Handle("/", LoggingMiddleware(http.HandlerFunc(NotFoundHandler)))
	mux.Handle("/api/v1/calculate", LoggingMiddleware(http.HandlerFunc(CalcHandler)))
	log.Printf("Web server run on port: %s\n", a.config.Addr)
	return http.ListenAndServe(":"+a.config.Addr, mux)
}

func parseExpression(expression string) (*Node, []*Node, error) {
	tokens, err := splitToTokens(expression)
	if err != nil {
		return nil, nil, err
	}
	rp, err := toReversePolish(tokens)
	if err != nil {
		return nil, nil, err
	}
	return evaluate(rp)
}

func splitToTokens(expr string) ([]Token, error) {
	var tokens []Token
	expr = strings.ReplaceAll(expr, " ", "")
	buf := new(bytes.Buffer)

	for _, r := range expr {
		switch {
		case unicode.IsDigit(r) || r == '.':
			buf.WriteRune(r)
		case isOperator(r):
			if buf.Len() > 0 {
				num, err := strconv.ParseFloat(buf.String(), 64)
				if err != nil {
					return nil, fmt.Errorf("invalid number: %s", buf.String())
				}
				tokens = append(tokens, Token{Type: "num", Num: num})
				buf.Reset()
			}
			tokens = append(tokens, Token{Type: "op", Value: string(r)})
		case r == '(' || r == ')':
			if buf.Len() > 0 {
				num, err := strconv.ParseFloat(buf.String(), 64)
				if err != nil {
					return nil, fmt.Errorf("invalid number: %s", buf.String())
				}
				tokens = append(tokens, Token{Type: "num", Num: num})
				buf.Reset()
			}
			tokens = append(tokens, Token{Type: "paren", Value: string(r)})
		default:
			return nil, fmt.Errorf("invalid character: %c", r)
		}
	}

	if buf.Len() > 0 {
		num, err := strconv.ParseFloat(buf.String(), 64)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, Token{Type: "num", Num: num})
	}

	return tokens, nil
}

func toReversePolish(tokens []Token) ([]Token, error) {
	var output []Token
	var stack []Token

	priority := map[string]int{
		"+": 1,
		"-": 1,
		"*": 2,
		"/": 2,
	}

	for _, token := range tokens {
		switch token.Type {
		case "num":
			output = append(output, token)
		case "op":
			for len(stack) > 0 && stack[len(stack)-1].Type == "op" &&
				priority[token.Value] <= priority[stack[len(stack)-1].Value] &&
				stack[len(stack)-1].Value != "(" {
				output = append(output, stack[len(stack)-1])
				stack = stack[:len(stack)-1]
			}
			stack = append(stack, token)
		case "paren":
			if token.Value == "(" {
				stack = append(stack, token)
			} else if token.Value == ")" {
				for len(stack) > 0 && stack[len(stack)-1].Value != "(" {
					output = append(output, stack[len(stack)-1])
					stack = stack[:len(stack)-1]
				}
				if len(stack) == 0 {
					return nil, fmt.Errorf("mismatched parentheses")
				}
				stack = stack[:len(stack)-1]
			}
		}
	}

	for len(stack) > 0 {
		if stack[len(stack)-1].Value == "(" {
			return nil, fmt.Errorf("mismatched parentheses")
		}
		output = append(output, stack[len(stack)-1])
		stack = stack[:len(stack)-1]
	}

	return output, nil
}

func evaluate(rp []Token) (*Node, []*Node, error) {
	var stack []*Node
	var allNodes []*Node

	for _, token := range rp {
		if token.Type == "num" {
			node := &Node{
				ID:     generateID(),
				Type:   "number",
				Value:  token.Num,
				Status: "done",
				Result: token.Num,
			}
			stack = append(stack, node)
			allNodes = append(allNodes, node)
		} else if token.Type == "op" {
			if len(stack) < 2 {
				return nil, nil, fmt.Errorf("invalid expression")
			}

			right := stack[len(stack)-1]
			left := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			node := &Node{
				ID:        generateID(),
				Type:      "operation",
				Operation: token.Value,
				Left:      left.ID,
				Right:     right.ID,
				Status:    "pending",
				Parents:   []string{},
			}

			left.Parents = append(left.Parents, node.ID)
			right.Parents = append(right.Parents, node.ID)

			stack = append(stack, node)
			allNodes = append(allNodes, node)
		}
	}

	if len(stack) != 1 {
		return nil, nil, fmt.Errorf("invalid expression")
	}

	return stack[0], allNodes, nil
}

func isOperator(token rune) bool {
	return token == '+' || token == '-' || token == '*' || token == '/'
}

// Генерация UID
func generateID() string {
	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), idCounter)
}
