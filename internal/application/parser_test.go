package application

import (
	"testing"
)

func TestSplitToTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []Token
		wantErr  bool
	}{
		{
			name:  "valid expression",
			input: "2+3.5*(4-1)",
			expected: []Token{
				{Type: "num", Num: 2},
				{Type: "op", Value: "+"},
				{Type: "num", Num: 3.5},
				{Type: "op", Value: "*"},
				{Type: "paren", Value: "("},
				{Type: "num", Num: 4},
				{Type: "op", Value: "-"},
				{Type: "num", Num: 1},
				{Type: "paren", Value: ")"},
			},
			wantErr: false,
		},
		{
			name:    "invalid character",
			input:   "2 + a",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := splitToTokens(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("splitToTokens() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !compareTokens(tokens, tt.expected) {
				t.Errorf("splitToTokens() = %v, want %v", tokens, tt.expected)
			}
		})
	}
}

func compareTokens(a, b []Token) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Type != b[i].Type || a[i].Value != b[i].Value || a[i].Num != b[i].Num {
			return false
		}
	}
	return true
}

func TestToReversePolish(t *testing.T) {
	tokens := []Token{
		{Type: "num", Num: 2},
		{Type: "op", Value: "+"},
		{Type: "num", Num: 3},
		{Type: "op", Value: "*"},
		{Type: "num", Num: 4},
	}

	rp, err := toReversePolish(tokens)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expected := []Token{
		{Type: "num", Num: 2},
		{Type: "num", Num: 3},
		{Type: "num", Num: 4},
		{Type: "op", Value: "*"},
		{Type: "op", Value: "+"},
	}

	if !compareTokens(rp, expected) {
		t.Errorf("toReversePolish() = %v, want %v", rp, expected)
	}
}
