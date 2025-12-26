package tmdb

import (
	"encoding/json"
	"strconv"
)

func getInt(m map[string]any, key string) (int, bool) {
	val, ok := m[key]
	if !ok {
		return 0, false
	}
	switch v := val.(type) {
	case float64:
		return int(v), true
	case int:
		return v, true
	case json.Number:
		i, err := strconv.Atoi(v.String())
		if err != nil {
			return 0, false
		}
		return i, true
	default:
		return 0, false
	}
}

func getString(m map[string]any, key string) (string, bool) {
	val, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := val.(string)
	return s, ok
}

func getEpisodeRuntime(details map[string]any) (int, bool) {
	val, ok := details["episode_run_time"]
	if !ok {
		return 0, false
	}

	switch v := val.(type) {
	case []any:
		if len(v) == 0 {
			return 0, false
		}
		switch first := v[0].(type) {
		case float64:
			return int(first), true
		case int:
			return first, true
		case string:
			i, err := strconv.Atoi(first)
			if err != nil {
				return 0, false
			}
			return i, true
		default:
			return 0, false
		}
	case []int:
		if len(v) == 0 {
			return 0, false
		}
		return v[0], true
	default:
		return 0, false
	}
}
