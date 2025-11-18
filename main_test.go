package main

import "testing"

func TestMainDelegatesToExecute(t *testing.T) {
	called := false
	orig := execute
	execute = func() { called = true }
	t.Cleanup(func() { execute = orig })

	main()

	if !called {
		t.Fatalf("expected execute to be called")
	}
}
