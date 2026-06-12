package fileutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatDuration(t *testing.T) {
	testCases := []struct {
		minutes  int
		expected string
	}{
		{120, "2h 0m"},
		{90, "1h 30m"},
		{45, "0h 45m"},
		{0, "0h 0m"},
		{135, "2h 15m"},
		{180, "3h 0m"},
	}

	for _, tc := range testCases {
		t.Run(tc.expected, func(t *testing.T) {
			result := FormatDuration(tc.minutes)
			assert.Equal(t, tc.expected, result)
		})
	}
}
