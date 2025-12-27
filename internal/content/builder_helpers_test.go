package content

import (
	"testing"
)

func TestHelperExtractors(t *testing.T) {
	t.Run("intVal parses formats", func(t *testing.T) {
		if got, ok := intVal(map[string]any{"value": "42"}, "value"); !ok || got != 42 {
			t.Fatalf("intVal string parse = %d,%v want 42,true", got, ok)
		}
		if got, ok := intVal(map[string]any{"value": float64(10)}, "value"); !ok || got != 10 {
			t.Fatalf("intVal float parse = %d,%v want 10,true", got, ok)
		}
	})

	t.Run("floatVal handles ints and floats", func(t *testing.T) {
		if got, ok := floatVal(map[string]any{"value": 3}, "value"); !ok || got != 3 {
			t.Fatalf("floatVal int parse = %v,%v want 3,true", got, ok)
		}
		if got, ok := floatVal(map[string]any{"value": float32(1.5)}, "value"); !ok || got != 1.5 {
			t.Fatalf("floatVal float32 parse = %v,%v want 1.5,true", got, ok)
		}
	})

	t.Run("boolVal covers strings and numbers", func(t *testing.T) {
		if !boolVal(map[string]any{"value": "true"}, "value") {
			t.Fatalf("boolVal should treat \"true\" as true")
		}
		if boolVal(map[string]any{"value": 0}, "value") {
			t.Fatalf("boolVal should treat 0 as false")
		}
	})

	t.Run("string helpers", func(t *testing.T) {
		if got := stringVal(map[string]any{"value": 123}, "value"); got != "" {
			t.Fatalf("stringVal non-string = %q, want empty", got)
		}

		if got := nestedString(map[string]any{"outer": map[string]any{"inner": "yes"}}, "outer", "inner"); got != "yes" {
			t.Fatalf("nestedString = %q, want yes", got)
		}

		if got := firstStringFromArray(map[string]any{"arr": []any{
			map[string]any{"name": ""},
			map[string]any{"name": "first"},
			map[string]any{"name": "second"},
		}}, "arr", "name"); got != "first" {
			t.Fatalf("firstStringFromArray = %q, want first", got)
		}

		slice := stringSlice(map[string]any{"arr": []any{"a", 2, "b"}}, "arr")
		if len(slice) != 2 || slice[0] != "a" || slice[1] != "b" {
			t.Fatalf("stringSlice mixed content = %v, want [a b]", slice)
		}
	})

	t.Run("usContentRating picks US entry", func(t *testing.T) {
		data := map[string]any{
			"content_ratings": map[string]any{
				"results": []any{
					map[string]any{"iso_3166_1": "GB", "rating": "15"},
					map[string]any{"iso_3166_1": "US", "rating": "TV-MA"},
				},
			},
		}
		if got := usContentRating(data); got != "TV-MA" {
			t.Fatalf("usContentRating = %q, want TV-MA", got)
		}
	})

	t.Run("format helpers", func(t *testing.T) {
		if got := countryFlag("fi"); got != "🇫🇮" {
			t.Fatalf("countryFlag lower case = %q, want 🇫🇮", got)
		}
		if got := countryFlag("xx"); got != "🌐" {
			t.Fatalf("countryFlag unknown = %q, want globe", got)
		}
		if got := formatNumber(1234567); got != "1,234,567" {
			t.Fatalf("formatNumber = %q, want 1,234,567", got)
		}
	})
}
