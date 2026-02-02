package cmdutil

import (
	"reflect"
	"strings"
	"time"
	"unicode"
)

// StructToMapOptions configures StructToMap behavior.
type StructToMapOptions struct {
	OmitFields       map[string]bool
	KeyOverrides     map[string]string
	JoinStringSlices bool
}

// StructToMap converts a struct into a map keyed by snake_case field names.
// It supports optional field omission, key overrides, and joining string slices.
func StructToMap[T any](value T, opts StructToMapOptions) map[string]any {
	result := make(map[string]any)
	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return result
		}
		v = v.Elem()
	}

	appendStructFields(v, result, opts)
	return result
}

func appendStructFields(v reflect.Value, result map[string]any, opts StructToMapOptions) {
	if v.Kind() != reflect.Struct {
		return
	}

	t := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			continue
		}
		if opts.OmitFields != nil && opts.OmitFields[field.Name] {
			continue
		}

		value := v.Field(i)
		if field.Anonymous && value.Kind() == reflect.Struct {
			appendStructFields(value, result, opts)
			continue
		}

		key := toSnakeCase(field.Name)
		if override, ok := opts.KeyOverrides[field.Name]; ok {
			key = override
		}

		result[key] = normalizeValue(value, opts)
	}
}

func normalizeValue(value reflect.Value, opts StructToMapOptions) any {
	if !value.IsValid() {
		return nil
	}

	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	if value.Type() == reflect.TypeOf(time.Time{}) {
		return value.Interface().(time.Time).String()
	}

	if opts.JoinStringSlices && value.Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.String {
		items := make([]string, value.Len())
		for i := 0; i < value.Len(); i++ {
			items[i] = value.Index(i).String()
		}
		return strings.Join(items, ",")
	}

	return value.Interface()
}

func toSnakeCase(input string) string {
	if input == "" {
		return ""
	}

	runes := []rune(input)
	var builder strings.Builder
	builder.Grow(len(runes) + 4)

	for i, r := range runes {
		if unicode.IsUpper(r) {
			if i > 0 {
				prev := runes[i-1]
				var next rune
				var nextNext rune
				if i+1 < len(runes) {
					next = runes[i+1]
				}
				if i+2 < len(runes) {
					nextNext = runes[i+2]
				}
				if unicode.IsLower(prev) || unicode.IsDigit(prev) {
					builder.WriteRune('_')
				} else if unicode.IsUpper(prev) && next != 0 && unicode.IsLower(next) {
					if nextNext == 0 || !unicode.IsUpper(nextNext) {
						builder.WriteRune('_')
					}
				}
			}
			builder.WriteRune(unicode.ToLower(r))
			continue
		}

		builder.WriteRune(unicode.ToLower(r))
	}

	return builder.String()
}
