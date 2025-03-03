package agent

import (
	"testing"
)

// Тестирование функции compute
func TestCompute(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		op       string
		expected float64
		err      bool
	}{
		{"Addition", 5, 3, "+", 8, false},
		{"Subtraction", 5, 3, "-", 2, false},
		{"Multiplication", 5, 3, "*", 15, false},
		{"Division", 6, 3, "/", 2, false},
		{"Division by zero", 6, 0, "/", 0, true},
		{"Unknown operation", 5, 3, "%", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := compute(tt.a, tt.b, tt.op)
			if (err != nil) != tt.err {
				t.Errorf("compute(%v, %v, %q) error = %v, expected error = %v", tt.a, tt.b, tt.op, err, tt.err)
				return
			}
			if result != tt.expected {
				t.Errorf("compute(%v, %v, %q) = %v, expected %v", tt.a, tt.b, tt.op, result, tt.expected)
			}
		})
	}
}
