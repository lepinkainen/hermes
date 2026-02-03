package tmdb

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetInt(t *testing.T) {
	tests := []struct {
		name   string
		input  map[string]any
		key    string
		want   int
		wantOK bool
	}{
		{
			name:   "float64",
			input:  map[string]any{"runtime": float64(120)},
			key:    "runtime",
			want:   120,
			wantOK: true,
		},
		{
			name:   "int",
			input:  map[string]any{"runtime": 95},
			key:    "runtime",
			want:   95,
			wantOK: true,
		},
		{
			name:   "json number",
			input:  map[string]any{"runtime": json.Number("88")},
			key:    "runtime",
			want:   88,
			wantOK: true,
		},
		{
			name:   "invalid type",
			input:  map[string]any{"runtime": "oops"},
			key:    "runtime",
			want:   0,
			wantOK: false,
		},
		{
			name:   "missing key",
			input:  map[string]any{"runtime": 1},
			key:    "missing",
			want:   0,
			wantOK: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := getInt(tt.input, tt.key)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetString(t *testing.T) {
	value, ok := getString(map[string]any{"status": "Ended"}, "status")
	assert.True(t, ok)
	assert.Equal(t, "Ended", value)

	value, ok = getString(map[string]any{"status": 123}, "status")
	assert.False(t, ok)
	assert.Empty(t, value)
}

func TestGetEpisodeRuntime(t *testing.T) {
	tests := []struct {
		name    string
		details map[string]any
		want    int
		wantOK  bool
	}{
		{
			name:    "float64 slice",
			details: map[string]any{"episode_run_time": []any{float64(50)}},
			want:    50,
			wantOK:  true,
		},
		{
			name:    "int slice",
			details: map[string]any{"episode_run_time": []any{int(42)}},
			want:    42,
			wantOK:  true,
		},
		{
			name:    "string slice",
			details: map[string]any{"episode_run_time": []any{"60"}},
			want:    60,
			wantOK:  true,
		},
		{
			name:    "typed int slice",
			details: map[string]any{"episode_run_time": []int{30}},
			want:    30,
			wantOK:  true,
		},
		{
			name:    "empty slice",
			details: map[string]any{"episode_run_time": []any{}},
			want:    0,
			wantOK:  false,
		},
		{
			name:    "missing",
			details: map[string]any{},
			want:    0,
			wantOK:  false,
		},
		{
			name:    "invalid type",
			details: map[string]any{"episode_run_time": "bad"},
			want:    0,
			wantOK:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := getEpisodeRuntime(tt.details)
			assert.Equal(t, tt.wantOK, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}
