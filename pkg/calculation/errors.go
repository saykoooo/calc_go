package calculation

import "errors"

var (
	ErrInvalidExpression = errors.New("invalid expression")
	ErrDivByZero = errors.New("division by zero")
)
