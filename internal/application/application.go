package application

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Addr string
}

type RespOk struct {
	Result string `json:"result"`
}

type RespError struct {
	Error string `json:"error"`
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

func ConfigFromEnv() *Config {
	config := new(Config)
	config.Addr = os.Getenv("PORT")
	if config.Addr == "" {
		config.Addr = "8080"
	}
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
	result, err := Calc(request.Expression)
	if err != nil {
		if errors.Is(err, ErrInvalidExpression) || errors.Is(err, ErrDivByZero) {
			w.WriteHeader(http.StatusUnprocessableEntity)
			errJsonData, _ := json.Marshal(RespError{Error: ErrInvalidExpression.Error()})
			w.Write(errJsonData)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			errJsonData, _ := json.Marshal(RespError{Error: "Internal server error"})
			w.Write(errJsonData)
		}
	} else {
		w.WriteHeader(http.StatusOK)
		s := fmt.Sprintf("%f", result)
		okJsonData, _ := json.Marshal(RespOk{Result: s})
		w.Write(okJsonData)
	}
}

func (a *Application) RunServer() error {
	mux := http.NewServeMux()
	mux.Handle("/", LoggingMiddleware(http.HandlerFunc(NotFoundHandler)))
	mux.Handle("/api/v1/calculate", LoggingMiddleware(http.HandlerFunc(CalcHandler)))
	log.Printf("Web server run on port: %s\n", a.config.Addr)
	return http.ListenAndServe(":"+a.config.Addr, mux)
}

func Calc(expression string) (float64, error) {
	tokens := splitToTokens(expression)
	rp, err := toReversePolish(tokens)
	if err != nil {
		return 0, err
	}
	return evaluate(rp)
}

func splitToTokens(expression string) []string {
	var tokens []string
	var current strings.Builder

	for _, char := range expression {
		if char == ' ' {
			continue
		}
		if isOperator(string(char)) || char == '(' || char == ')' {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			tokens = append(tokens, string(char))
		} else {
			current.WriteRune(char)
		}
	}
	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}
	return tokens
}

func toReversePolish(tokens []string) ([]string, error) {
	var output []string
	var operators []string

	for _, token := range tokens {
		if isNumber(token) {
			output = append(output, token)
		} else if token == "(" {
			operators = append(operators, token)
		} else if token == ")" {
			for len(operators) > 0 && operators[len(operators)-1] != "(" {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}
			if len(operators) == 0 {
				return nil, ErrInvalidExpression
			}
			operators = operators[:len(operators)-1]
		} else if isOperator(token) {
			for len(operators) > 0 && priority(operators[len(operators)-1]) >= priority(token) {
				output = append(output, operators[len(operators)-1])
				operators = operators[:len(operators)-1]
			}
			operators = append(operators, token)
		} else {
			return nil, ErrInvalidExpression
		}
	}

	for len(operators) > 0 {
		if operators[len(operators)-1] == "(" {
			return nil, ErrInvalidExpression
		}
		output = append(output, operators[len(operators)-1])
		operators = operators[:len(operators)-1]
	}

	return output, nil
}

func evaluate(rp []string) (float64, error) {
	var stack []float64

	for _, token := range rp {
		if isNumber(token) {
			num, err := strconv.ParseFloat(token, 64)
			if err != nil {
				return 0, ErrInvalidExpression
			}
			stack = append(stack, num)
		} else if isOperator(token) {
			if len(stack) < 2 {
				return 0, ErrInvalidExpression
			}
			b := stack[len(stack)-1]
			a := stack[len(stack)-2]
			stack = stack[:len(stack)-2]

			var result float64
			switch token {
			case "+":
				result = a + b
			case "-":
				result = a - b
			case "*":
				result = a * b
			case "/":
				if b == 0 {
					return 0, ErrDivByZero
				}
				result = a / b
			}
			stack = append(stack, result)
		} else {
			return 0, ErrInvalidExpression
		}
	}

	if len(stack) != 1 {
		return 0, ErrInvalidExpression
	}
	return stack[0], nil
}

func isOperator(token string) bool {
	return token == "+" || token == "-" || token == "*" || token == "/"
}

func isNumber(token string) bool {
	_, err := strconv.ParseFloat(token, 64)
	return err == nil
}

func priority(op string) int {
	switch op {
	case "+", "-":
		return 1
	case "*", "/":
		return 2
	}
	return 0
}
