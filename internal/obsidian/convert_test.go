package obsidian

import "testing"

func TestIntFromAny(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
	}{
		{"int", 42, 42},
		{"int64", int64(123), 123},
		{"float64", float64(99.7), 99},
		{"string", "456", 456},
		{"string with spaces", "  789  ", 789},
		{"invalid string", "not a number", 0},
		{"nil", nil, 0},
		{"bool", true, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IntFromAny(tt.val)
			if got != tt.want {
				t.Errorf("IntFromAny(%v) = %d, want %d", tt.val, got, tt.want)
			}
		})
	}
}

func TestStringFromAny(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want string
	}{
		{"string", "hello", "hello"},
		{"string with spaces", "  world  ", "world"},
		{"int", 42, ""},
		{"nil", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StringFromAny(tt.val)
			if got != tt.want {
				t.Errorf("StringFromAny(%v) = %q, want %q", tt.val, got, tt.want)
			}
		})
	}
}
