package calculation

import (
	"strconv"
	"strings"
)

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
				return nil, ErrInvalidExpression	}
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
