package trie

import (
	"testing"
)

func TestEqCorrectness(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*Trie, *Trie)
		expectEq bool
	}{
		{
			name: "identical_empty_tries",
			setup: func() (*Trie, *Trie) {
				return New(), New()
			},
			expectEq: true,
		},
		{
			name: "identical_single_path",
			setup: func() (*Trie, *Trie) {
				t1, t2 := New(), New()
				path := []string{"a", "b", "c"}
				t1.Insert(path)
				t2.Insert(path)
				return t1, t2
			},
			expectEq: true,
		},
		{
			name: "identical_multiple_paths",
			setup: func() (*Trie, *Trie) {
				t1, t2 := New(), New()
				paths := [][]string{
					{"a", "b", "c"},
					{"a", "b", "d"},
					{"x", "y", "z"},
				}
				for _, path := range paths {
					t1.Insert(path)
					t2.Insert(path)
				}
				return t1, t2
			},
			expectEq: true,
		},
		{
			name: "different_paths",
			setup: func() (*Trie, *Trie) {
				t1, t2 := New(), New()
				t1.Insert([]string{"a", "b", "c"})
				t2.Insert([]string{"a", "b", "d"})
				return t1, t2
			},
			expectEq: false,
		},
		{
			name: "different_number_of_paths",
			setup: func() (*Trie, *Trie) {
				t1, t2 := New(), New()
				t1.Insert([]string{"a", "b", "c"})
				t2.Insert([]string{"a", "b", "c"})
				t2.Insert([]string{"x", "y", "z"})
				return t1, t2
			},
			expectEq: false,
		},
		{
			name: "different_path_lengths",
			setup: func() (*Trie, *Trie) {
				t1, t2 := New(), New()
				t1.Insert([]string{"a", "b", "c"})
				t2.Insert([]string{"a", "b"})
				return t1, t2
			},
			expectEq: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t1, t2 := tt.setup()

			structResult := t1.Eq(t2)
			if structResult != tt.expectEq {
				t.Errorf("StructEq returned %v, expected %v", structResult, tt.expectEq)
			}
		})
	}
}

func buildSpecificTrie(paths [][]string) *Trie {
	t := New()
	for _, path := range paths {
		t.Insert(path)
	}
	return t
}

func TestSpecificStructures(t *testing.T) {
	tests := []struct {
		name     string
		paths1   [][]string
		paths2   [][]string
		expectEq bool
	}{
		{
			name: "deep_vs_wide",
			paths1: [][]string{
				{"a", "b", "c", "d", "e"},
				{"a", "b", "c", "d", "f"},
			},
			paths2: [][]string{
				{"a", "b"},
				{"a", "c"},
				{"a", "d"},
				{"a", "e"},
				{"a", "f"},
			},
			expectEq: false,
		},
		{
			name: "different_order_same_result",
			paths1: [][]string{
				{"a", "b", "c"},
				{"x", "y", "z"},
			},
			paths2: [][]string{
				{"x", "y", "z"},
				{"a", "b", "c"},
			},
			expectEq: true,
		},
		{
			name: "prefix_overlap",
			paths1: [][]string{
				{"a", "b", "c"},
				{"a", "b"},
			},
			paths2: [][]string{
				{"a", "b", "c"},
			},
			expectEq: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t1 := buildSpecificTrie(tt.paths1)
			t2 := buildSpecificTrie(tt.paths2)

			hashResult := t1.Eq(t2)
			if hashResult != tt.expectEq {
				t.Errorf("HashEq got %v, want %v", hashResult, tt.expectEq)
			}
		})
	}
}

func TestArenaNotEqual(t *testing.T) {
	tr1 := New()
	tr2 := New()

	tr1.Insert([]string{"a", "b", "c"})
	tr2.Insert([]string{"a", "b", "d"})

	if tr1.Eq(tr2) {
		t.Error("inserted different sequences but trie is equal")
	}
}

func TestArenaDebugString(t *testing.T) {
	tr := New()
	tr.Insert([]string{"a", "b"})
	tr.Insert([]string{"a", "c"})

	expected := "a(b(*)c(*))"
	str := tr.String()

	if str != expected {
		t.Errorf("DebugString() = %q, expected %q", str, expected)
	}
}

func TestDirectArenaOperations(t *testing.T) {
	arena := newArena()

	sequences := [][]string{
		{"a", "b", "c"},
		{"a", "b", "d"},
		{"a", "e"},
		{"f"},
	}

	for _, seq := range sequences {
		arena.insert(seq)
	}

	expected := "a(b(c(*)d(*))e(*))f(*)"
	str := arena.string()

	if str != expected {
		t.Errorf("arena.DebugString() = %q, expected %q", str, expected)
	}

	arena2 := newArena()
	for _, seq := range sequences {
		arena2.insert(seq)
	}

	if !arena.eq(arena2) {
		t.Error("inserted same sequences but arena is not equal")
	}
}
