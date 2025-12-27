package content

import (
	"testing"
)

func TestBuildTMDBPersonLink(t *testing.T) {
	t.Run("valid name and ID", func(t *testing.T) {
		got := buildTMDBPersonLink("Christopher Nolan", 525)
		want := "[Christopher Nolan](https://www.themoviedb.org/person/525)"
		if got != want {
			t.Fatalf("buildTMDBPersonLink = %q, want %q", got, want)
		}
	})

	t.Run("special characters in name", func(t *testing.T) {
		got := buildTMDBPersonLink("Robert Downey Jr.", 3223)
		want := "[Robert Downey Jr.](https://www.themoviedb.org/person/3223)"
		if got != want {
			t.Fatalf("buildTMDBPersonLink = %q, want %q", got, want)
		}
	})
}

func TestGetDirectors(t *testing.T) {
	t.Run("single director", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Christopher Nolan", "id": 525, "job": "Director", "department": "Directing"},
					map[string]any{"name": "Emma Thomas", "id": 1233, "job": "Producer", "department": "Production"},
				},
			},
		}
		got := getDirectors(details)
		want := "[Christopher Nolan](https://www.themoviedb.org/person/525)"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getDirectors = %v, want [%s]", got, want)
		}
	})

	t.Run("multiple directors", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Lana Wachowski", "id": 1271, "job": "Director", "department": "Directing"},
					map[string]any{"name": "Lilly Wachowski", "id": 1272, "job": "Director", "department": "Directing"},
				},
			},
		}
		got := getDirectors(details)
		want1 := "[Lana Wachowski](https://www.themoviedb.org/person/1271)"
		want2 := "[Lilly Wachowski](https://www.themoviedb.org/person/1272)"
		if len(got) != 2 || got[0] != want1 || got[1] != want2 {
			t.Fatalf("getDirectors = %v, want [%s, %s]", got, want1, want2)
		}
	})

	t.Run("no directors", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Emma Thomas", "job": "Producer", "department": "Production"},
				},
			},
		}
		got := getDirectors(details)
		if len(got) != 0 {
			t.Fatalf("getDirectors = %v, want []", got)
		}
	})

	t.Run("no credits", func(t *testing.T) {
		details := map[string]any{}
		got := getDirectors(details)
		if len(got) != 0 {
			t.Fatalf("getDirectors with no credits = %v, want []", got)
		}
	})

	t.Run("missing crew array", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{},
		}
		got := getDirectors(details)
		if len(got) != 0 {
			t.Fatalf("getDirectors with empty credits = %v, want []", got)
		}
	})

	t.Run("director without ID", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Director NoID", "job": "Director", "department": "Directing"},
					map[string]any{"name": "Director WithID", "id": 999, "job": "Director", "department": "Directing"},
				},
			},
		}
		got := getDirectors(details)
		// Should only return the director with an ID
		want := "[Director WithID](https://www.themoviedb.org/person/999)"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getDirectors (director without ID) = %v, want [%s]", got, want)
		}
	})
}

func TestGetWriters(t *testing.T) {
	t.Run("multiple writers with different roles", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Jane Smith", "id": 1001, "job": "Screenplay", "department": "Writing"},
					map[string]any{"name": "John Doe", "id": 1002, "job": "Novel", "department": "Writing"},
					map[string]any{"name": "Bob Jones", "id": 1003, "job": "Story", "department": "Writing"},
					map[string]any{"name": "Not A Writer", "id": 1004, "job": "Producer", "department": "Production"},
				},
			},
		}
		got := getWriters(details)
		if len(got) != 3 {
			t.Fatalf("getWriters count = %d, want 3", len(got))
		}
		want0 := "[Jane Smith](https://www.themoviedb.org/person/1001) (Screenplay)"
		if got[0] != want0 {
			t.Fatalf("getWriters[0] = %q, want %q", got[0], want0)
		}
		want1 := "[John Doe](https://www.themoviedb.org/person/1002) (Novel)"
		if got[1] != want1 {
			t.Fatalf("getWriters[1] = %q, want %q", got[1], want1)
		}
		want2 := "[Bob Jones](https://www.themoviedb.org/person/1003) (Story)"
		if got[2] != want2 {
			t.Fatalf("getWriters[2] = %q, want %q", got[2], want2)
		}
	})

	t.Run("single writer", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Aaron Sorkin", "id": 1776, "job": "Screenplay", "department": "Writing"},
				},
			},
		}
		got := getWriters(details)
		want := "[Aaron Sorkin](https://www.themoviedb.org/person/1776) (Screenplay)"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getWriters = %v, want [%s]", got, want)
		}
	})

	t.Run("no writers", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Director", "job": "Director", "department": "Directing"},
				},
			},
		}
		got := getWriters(details)
		if len(got) != 0 {
			t.Fatalf("getWriters = %v, want []", got)
		}
	})

	t.Run("no credits", func(t *testing.T) {
		details := map[string]any{}
		got := getWriters(details)
		if len(got) != 0 {
			t.Fatalf("getWriters with no credits = %v, want []", got)
		}
	})

	t.Run("writer with empty job field", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Mystery Writer", "id": 999, "job": "", "department": "Writing"},
				},
			},
		}
		got := getWriters(details)
		want := "[Mystery Writer](https://www.themoviedb.org/person/999)"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getWriters with empty job = %v, want [%s]", got, want)
		}
	})

	t.Run("writer without ID", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"crew": []any{
					map[string]any{"name": "Writer NoID", "job": "Screenplay", "department": "Writing"},
					map[string]any{"name": "Writer WithID", "id": 888, "job": "Story", "department": "Writing"},
				},
			},
		}
		got := getWriters(details)
		// Should only return the writer with an ID
		want := "[Writer WithID](https://www.themoviedb.org/person/888) (Story)"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getWriters (writer without ID) = %v, want [%s]", got, want)
		}
	})
}

func TestGetTopActors(t *testing.T) {
	t.Run("exactly 5 actors with character names", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"cast": []any{
					map[string]any{"name": "Actor 1", "id": 101, "character": "Character 1", "order": 0},
					map[string]any{"name": "Actor 2", "id": 102, "character": "Character 2", "order": 1},
					map[string]any{"name": "Actor 3", "id": 103, "character": "Character 3", "order": 2},
					map[string]any{"name": "Actor 4", "id": 104, "character": "Character 4", "order": 3},
					map[string]any{"name": "Actor 5", "id": 105, "character": "Character 5", "order": 4},
				},
			},
		}
		got := getTopActors(details)
		if len(got) != 5 {
			t.Fatalf("getTopActors count = %d, want 5", len(got))
		}
		want0 := "[Actor 1](https://www.themoviedb.org/person/101) as Character 1"
		if got[0] != want0 {
			t.Fatalf("getTopActors[0] = %q, want %q", got[0], want0)
		}
	})

	t.Run("more than 5 actors should return only top 5", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"cast": []any{
					map[string]any{"name": "Actor 1", "id": 101, "character": "Character 1", "order": 0},
					map[string]any{"name": "Actor 2", "id": 102, "character": "Character 2", "order": 1},
					map[string]any{"name": "Actor 3", "id": 103, "character": "Character 3", "order": 2},
					map[string]any{"name": "Actor 4", "id": 104, "character": "Character 4", "order": 3},
					map[string]any{"name": "Actor 5", "id": 105, "character": "Character 5", "order": 4},
					map[string]any{"name": "Actor 6", "id": 106, "character": "Character 6", "order": 5},
					map[string]any{"name": "Actor 7", "id": 107, "character": "Character 7", "order": 6},
				},
			},
		}
		got := getTopActors(details)
		if len(got) != 5 {
			t.Fatalf("getTopActors count = %d, want 5 (should skip order >= 5)", len(got))
		}
	})

	t.Run("fewer than 5 actors", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"cast": []any{
					map[string]any{"name": "Actor 1", "id": 101, "character": "Character 1", "order": 0},
					map[string]any{"name": "Actor 2", "id": 102, "character": "Character 2", "order": 1},
				},
			},
		}
		got := getTopActors(details)
		if len(got) != 2 {
			t.Fatalf("getTopActors count = %d, want 2", len(got))
		}
	})

	t.Run("actor with missing character name", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"cast": []any{
					map[string]any{"name": "Actor NoChar", "id": 999, "character": "", "order": 0},
				},
			},
		}
		got := getTopActors(details)
		want := "[Actor NoChar](https://www.themoviedb.org/person/999)"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getTopActors (no character) = %v, want [%s]", got, want)
		}
	})

	t.Run("actor without ID", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{
				"cast": []any{
					map[string]any{"name": "Actor NoID", "character": "Character", "order": 0},
					map[string]any{"name": "Actor WithID", "id": 888, "character": "Character 2", "order": 1},
				},
			},
		}
		got := getTopActors(details)
		// Should only return the actor with an ID
		want := "[Actor WithID](https://www.themoviedb.org/person/888) as Character 2"
		if len(got) != 1 || got[0] != want {
			t.Fatalf("getTopActors (actor without ID) = %v, want [%s]", got, want)
		}
	})

	t.Run("no cast data", func(t *testing.T) {
		details := map[string]any{
			"credits": map[string]any{},
		}
		got := getTopActors(details)
		if len(got) != 0 {
			t.Fatalf("getTopActors with no cast = %v, want []", got)
		}
	})

	t.Run("no credits", func(t *testing.T) {
		details := map[string]any{}
		got := getTopActors(details)
		if len(got) != 0 {
			t.Fatalf("getTopActors with no credits = %v, want []", got)
		}
	})
}
