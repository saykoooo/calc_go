package calc

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
)

type Token struct {
	Type  string // num, op, paren
	Value string
	Num   float64
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
	idCounter int
	idMutex   sync.Mutex
)

func ParseExpression(expression string) (*Node, []*Node, error) {
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
				ID:     GenerateID(),
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
				ID:        GenerateID(),
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
func GenerateID() string {
	idMutex.Lock()
	defer idMutex.Unlock()
	idCounter++
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), idCounter)
}
