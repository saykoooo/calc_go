package calculation

import "errors"

var (
	ErrInvalidExpression = errors.New("Expression is not valid")
	ErrDivByZero         = errors.New("Expression is not valid (division by zero)")
)
