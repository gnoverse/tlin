package trie

import (
	"sort"
	"strings"
)

/*
Arena-based Trie Implementation

This implementation uses an arena-based memory allocation strategy to improve memory efficiency
and reduce garbage collection overhead in the trie data structure. Here's how it works:

1. Memory Allocation Efficiency:
	- This arena implementation pre-allocates a contiguous slice of nodes and manages them
	as a pool, dramatically reducing the number of separate allocations.
	- Nodes are stored in a single slice and referenced by index rather than pointers,
	which reduces memory overhead and improves locality.

2. Benefits:
	- Reduced GC Pressure: Fewer allocations mean less work for the garbage collector.
	- Improved Memory Locality: Related data is stored contiguously in memory, improving
		CPU cache utilization and reducing cache misses during traversal.
	- Reduced Memory Fragmentation: A single large allocation instead of many small ones
		minimizes memory fragmentation.
	- Smaller Memory Footprint: Using integer indices instead of pointers saves memory,
		especially on 64-bit systems where pointers are 8 bytes.

3. Implementation Details:
	- The Arena struct manages a slice of nodes where each node is referenced by its index.
	- New nodes are appended to the slice, and their index is used for referencing.
	- Child nodes are referenced by their index in the arena rather than by pointer.
*/

// NodeIndex represents the index of a trie node.
type NodeIndex int

// arena is a memory pool that stores all trie nodes.
type arena struct {
	// nodes is a slice that stores all trie nodes.
	nodes []arenaNode
}

// arenaNode is the internal representation of a trie node stored in the arena.
type arenaNode struct {
	// children stores child nodes. key is the path segment, value is the index of the child node.
	children map[string]NodeIndex
	// isEnd indicates whether this node is the end of a path.
	isEnd bool
}

// newArena creates a new arena.
func newArena() *arena {
	arena := &arena{
		nodes: make([]arenaNode, 0, 1024), // Set initial capacity
	}
	// root node (index 0)
	arena.nodes = append(arena.nodes, arenaNode{
		children: make(map[string]NodeIndex),
		isEnd:    false,
	})
	return arena
}

// newNode adds a new node to the arena and returns its index.
func (a *arena) newNode() NodeIndex {
	idx := NodeIndex(len(a.nodes))
	a.nodes = append(a.nodes, arenaNode{
		children: make(map[string]NodeIndex),
		isEnd:    false,
	})
	return idx
}

// insert inserts a sequence of strings (representing a path) into the trie.
func (a *arena) insert(sequence []string) {
	current := NodeIndex(0) // root node

	for _, part := range sequence {
		node := &a.nodes[current]
		childIdx, exists := node.children[part]

		if !exists {
			childIdx = a.newNode()
			node.children[part] = childIdx
		}

		current = childIdx
	}

	a.nodes[current].isEnd = true
}

// eq checks whether two tries are identical in structure and content.
func (a *arena) eq(b *arena) bool {
	if len(a.nodes) != len(b.nodes) {
		return false
	}

	return a.equalNodes(NodeIndex(0), b, NodeIndex(0))
}

// equalNodes recursively checks whether two nodes (and their subtrees) are identical.
func (a *arena) equalNodes(aIdx NodeIndex, b *arena, bIdx NodeIndex) bool {
	nodeA := a.nodes[aIdx]
	nodeB := b.nodes[bIdx]

	// Quick checks for obvious differences
	if nodeA.isEnd != nodeB.isEnd || len(nodeA.children) != len(nodeB.children) {
		return false
	}

	// Pre-allocate slice with exact capacity
	keys := make([]string, 0, len(nodeA.children))
	for key := range nodeA.children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Compare children in sorted order
	for _, key := range keys {
		childA := nodeA.children[key]
		childB, exists := nodeB.children[key]
		if !exists || !a.equalNodes(childA, b, childB) {
			return false
		}
	}

	return true
}

// string returns a string representation of the trie for debugging purposes.
func (a *arena) string() string {
	return a.debugStringNode(NodeIndex(0))
}

// debugStringNode recursively generates a string representation of a specific node (and its subtree).
func (a *arena) debugStringNode(idx NodeIndex) string {
	node := a.nodes[idx]
	var sb strings.Builder

	if node.isEnd {
		sb.WriteString("*")
	}

	// Sort keys for consistent order
	keys := make([]string, 0, len(node.children))
	for key := range node.children {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		sb.WriteString(key)
		sb.WriteString("(")
		sb.WriteString(a.debugStringNode(node.children[key]))
		sb.WriteString(")")
	}

	return sb.String()
}

// Trie is a wrapper for compatibility with existing API.
type Trie struct {
	arena *arena
}

// New returns an initialized Trie.
func New() *Trie {
	return &Trie{
		arena: newArena(),
	}
}

// Insert inserts a sequence of strings (representing a path) into the trie.
func (t *Trie) Insert(sequence []string) {
	t.arena.insert(sequence)
}

// Eq checks whether two tries are identical in structure and content.
func (t *Trie) Eq(other *Trie) bool {
	return t.arena.eq(other.arena)
}

// String returns a string representation of the trie for debugging purposes.
func (t *Trie) String() string {
	return t.arena.string()
}
