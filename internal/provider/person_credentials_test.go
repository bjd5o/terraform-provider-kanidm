package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestCredentialsVersionChanged(t *testing.T) {
	tests := []struct {
		name        string
		plan, state types.Int64
		want        bool
	}{
		{"unset", types.Int64Null(), types.Int64Null(), false},
		{"first set", types.Int64Value(1), types.Int64Null(), true},
		{"unchanged", types.Int64Value(1), types.Int64Value(1), false},
		{"increased", types.Int64Value(2), types.Int64Value(1), true},
		{"decreased", types.Int64Value(1), types.Int64Value(2), true},
		{"removed", types.Int64Null(), types.Int64Value(1), false},
		{"unknown", types.Int64Unknown(), types.Int64Value(1), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := credentialsVersionChanged(tt.plan, tt.state); got != tt.want {
				t.Fatalf("credentialsVersionChanged(%v, %v) = %v, want %v", tt.plan, tt.state, got, tt.want)
			}
		})
	}
}
