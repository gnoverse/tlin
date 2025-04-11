package trie

import (
	"testing"
)

func TestInsertAndEqual(t *testing.T) {
	tr1 := New()
	tr2 := New()

	sequences := [][]string{
		{"a", "b", "c"},
		{"a", "b", "d"},
		{"a", "e"},
		{"f"},
	}

	for _, seq := range sequences {
		tr1.Insert(seq)
		tr2.Insert(seq)
	}

	if !tr1.Equal(tr2) {
		t.Error("inserted same sequences but trie is not equal")
	}

	tr3 := New()
	tr3.Insert([]string{"f"})
	tr3.Insert([]string{"a", "b", "d"})
	tr3.Insert([]string{"a", "e"})
	tr3.Insert([]string{"a", "b", "c"})

	if !tr1.Equal(tr3) {
		t.Error("inserted same sequences but trie is not equal")
	}
}

func TestArenaNotEqual(t *testing.T) {
	tr1 := New()
	tr2 := New()

	tr1.Insert([]string{"a", "b", "c"})
	tr2.Insert([]string{"a", "b", "d"})

	if tr1.Equal(tr2) {
		t.Error("inserted different sequences but trie is equal")
	}
}

func TestArenaDebugString(t *testing.T) {
	tr := New()
	tr.Insert([]string{"a", "b"})
	tr.Insert([]string{"a", "c"})

	expected := "a(b(*)c(*))"
	str := tr.DebugString()

	if str != expected {
		t.Errorf("DebugString() = %q, expected %q", str, expected)
	}
}

func TestDirectArenaOperations(t *testing.T) {
	arena := NewArena()

	sequences := [][]string{
		{"a", "b", "c"},
		{"a", "b", "d"},
		{"a", "e"},
		{"f"},
	}

	for _, seq := range sequences {
		arena.Insert(seq)
	}

	expected := "a(b(c(*)d(*))e(*))f(*)"
	str := arena.DebugString()

	if str != expected {
		t.Errorf("arena.DebugString() = %q, expected %q", str, expected)
	}

	arena2 := NewArena()
	for _, seq := range sequences {
		arena2.Insert(seq)
	}

	if !arena.Equal(arena2) {
		t.Error("inserted same sequences but arena is not equal")
	}
}
