package calc

import (
	"testing"
)

func TestEvaluate(t *testing.T) {
	tests := []struct {
		name    string
		rp      []Token
		wantErr bool
	}{
		{
			name: "valid expression",
			rp: []Token{
				{Type: "num", Num: 2},
				{Type: "num", Num: 3},
				{Type: "num", Num: 4},
				{Type: "op", Value: "*"},
				{Type: "op", Value: "+"},
			},
			wantErr: false,
		},
		{
			name: "invalid expression",
			rp: []Token{
				{Type: "op", Value: "+"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, nodes, err := evaluate(tt.rp)
			if (err != nil) != tt.wantErr {
				t.Errorf("evaluate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if root == nil {
					t.Error("Expected non-nil root node")
				}
				if len(nodes) == 0 {
					t.Error("Expected non-empty nodes list")
				}
			}
		})
	}
}
