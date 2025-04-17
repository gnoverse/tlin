package trie

import (
	"math/rand"
	"testing"
)

func generateRandomSequences(count, maxLength int) [][]string {
	sequences := make([][]string, count)
	for i := range count {
		length := rand.Intn(maxLength) + 1
		sequence := make([]string, length)
		for j := range length {
			sequence[j] = string(rune('a' + rand.Intn(26)))
		}
		sequences[i] = sequence
	}
	return sequences
}

func BenchmarkInsert(b *testing.B) {
	sizes := []struct {
		name      string
		count     int
		maxLength int
	}{
		{"Small", 100, 5},
		{"Medium", 1000, 10},
		{"Large", 10000, 20},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			sequences := generateRandomSequences(size.count, size.maxLength)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				trie := New()
				for _, seq := range sequences {
					trie.Insert(seq)
				}
			}
		})
	}
}

func BenchmarkInsert2(b *testing.B) {
	sequences := [][]string{
		{"a", "b", "c"},
		{"a", "b", "d"},
		{"a", "e"},
		{"f"},
		{"g", "h", "i", "j"},
		{"g", "h", "k"},
		{"l", "m", "n"},
	}

	b.Run("Original", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			tr := New()
			for _, seq := range sequences {
				tr.Insert(seq)
			}
		}
	})
}

func BenchmarkEqual(b *testing.B) {
	sizes := []struct {
		name      string
		count     int
		maxLength int
	}{
		{"Small", 100, 5},
		{"Medium", 1000, 10},
		{"Large", 10000, 20},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			sequences := generateRandomSequences(size.count, size.maxLength)

			trie1 := New()
			trie2 := New()
			for _, seq := range sequences {
				trie1.Insert(seq)
				trie2.Insert(seq)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				trie1.Eq(trie2)
			}
		})
	}
}

func BenchmarkEqualDifferent(b *testing.B) {
	sizes := []struct {
		name      string
		count     int
		maxLength int
	}{
		{"Small", 100, 5},
		{"Medium", 1000, 10},
		{"Large", 10000, 20},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			sequences1 := generateRandomSequences(size.count, size.maxLength)
			sequences2 := generateRandomSequences(size.count, size.maxLength)

			trie1 := New()
			trie2 := New()
			for _, seq := range sequences1 {
				trie1.Insert(seq)
			}
			for _, seq := range sequences2 {
				trie2.Insert(seq)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				trie1.Eq(trie2)
			}
		})
	}
}

func BenchmarkDebugString(b *testing.B) {
	sizes := []struct {
		name      string
		count     int
		maxLength int
	}{
		{"Small", 100, 5},
		{"Medium", 1000, 10},
		{"Large", 10000, 20},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			sequences := generateRandomSequences(size.count, size.maxLength)
			trie := New()
			for _, seq := range sequences {
				trie.Insert(seq)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				trie.String()
			}
		})
	}
}
