package comparator

import (
	"math/bits"
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// bruteForceSearch returns every index whose hash is within maxDist of query.
func bruteForceSearch(hashes []uint64, query uint64, maxDist int) []int {
	var out []int
	for i, h := range hashes {
		if bits.OnesCount64(h^query) <= maxDist {
			out = append(out, i)
		}
	}
	return out
}

// sortedInts returns a sorted copy so we can compare order-independent sets.
func sortedInts(s []int) []int {
	cp := make([]int, len(s))
	copy(cp, s)
	sort.Ints(cp)
	return cp
}

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// buildTree inserts every (hash, index) pair into a fresh bkTree and returns it.
func buildTree(hashes []uint64) *bkTree {
	t := &bkTree{}
	for i, h := range hashes {
		t.insert(h, i)
	}
	return t
}

// ---------------------------------------------------------------------------
// TestBKTree_Insert_Search_Exact
// ---------------------------------------------------------------------------

func TestBKTree_Insert_Search_Exact(t *testing.T) {
	hashes := []uint64{
		0xAAAAAAAAAAAAAAAA,
		0xBBBBBBBBBBBBBBBB,
		0xCCCCCCCCCCCCCCCC,
		0xDDDDDDDDDDDDDDDD,
	}
	tree := buildTree(hashes)

	for wantIdx, query := range hashes {
		got := tree.search(query, 0)
		if len(got) != 1 || got[0] != wantIdx {
			t.Errorf("exact search for hash[%d]: got indices %v, want [%d]", wantIdx, got, wantIdx)
		}
	}

	// A hash that was never inserted should return nothing at maxDist=0
	foreign := uint64(0x1234567890ABCDEF)
	if got := tree.search(foreign, 0); len(got) != 0 {
		t.Errorf("search for non-existent hash returned %v, want []", got)
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_Search_WithinThreshold
// ---------------------------------------------------------------------------

func TestBKTree_Search_WithinThreshold(t *testing.T) {
	base := uint64(0xFFFFFFFFFFFFFFFF)

	// Build controlled hashes at known Hamming distances from base.
	//   dist0 = base itself       (Hamming 0)
	//   dist4 = flip 4 bits       (Hamming 4)
	//   dist8 = flip 8 bits       (Hamming 8)
	//   dist16 = flip 16 bits     (Hamming 16)
	dist0 := base
	dist4 := base ^ 0xF               // lowest 4 bits → distance 4
	dist8 := base ^ 0xFF              // lowest 8 bits → distance 8
	dist16 := base ^ 0xFFFF           // lowest 16 bits → distance 16

	hashes := []uint64{dist0, dist4, dist8, dist16}
	tree := buildTree(hashes)

	cases := []struct {
		maxDist     int
		wantIndices []int // sorted indices of hashes[] that are within maxDist of base
	}{
		{0, []int{0}},           // only dist0
		{4, []int{0, 1}},        // dist0 + dist4
		{8, []int{0, 1, 2}},     // dist0 + dist4 + dist8
		{16, []int{0, 1, 2, 3}}, // all
	}

	for _, tc := range cases {
		got := sortedInts(tree.search(base, tc.maxDist))
		want := sortedInts(tc.wantIndices)
		if !intsEqual(got, want) {
			t.Errorf("search(base, maxDist=%d): got %v, want %v", tc.maxDist, got, want)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_Search_Empty
// ---------------------------------------------------------------------------

func TestBKTree_Search_Empty(t *testing.T) {
	tree := &bkTree{}
	got := tree.search(0xDEADBEEFDEADBEEF, 10)
	if got != nil {
		t.Errorf("search on empty tree: got %v, want nil", got)
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_Search_SameHash_Multiple
// ---------------------------------------------------------------------------

func TestBKTree_Search_SameHash_Multiple(t *testing.T) {
	const sharedHash = uint64(0x0F0F0F0F0F0F0F0F)
	const n = 5

	tree := &bkTree{}
	for i := 0; i < n; i++ {
		tree.insert(sharedHash, i)
	}

	got := sortedInts(tree.search(sharedHash, 0))
	want := []int{0, 1, 2, 3, 4}
	if !intsEqual(got, want) {
		t.Errorf("same-hash multi-insert: got %v, want %v", got, want)
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_NoPrunedMatches (brute-force vs BK-tree, 100 items)
// ---------------------------------------------------------------------------

// deterministicHash produces a spread-out but reproducible uint64 from an int
// seed without importing math/rand (keeps the test self-contained).
func deterministicHash(seed int) uint64 {
	// Knuth multiplicative hash, then mix a few more times for bit spread.
	v := uint64(seed+1) * 6364136223846793005
	v ^= v >> 27
	v *= 2685821657736338717
	v ^= v >> 31
	return v
}

func TestBKTree_NoPrunedMatches(t *testing.T) {
	const n = 100
	const maxDist = 6

	hashes := make([]uint64, n)
	for i := range hashes {
		hashes[i] = deterministicHash(i)
	}

	tree := buildTree(hashes)

	for qi := range hashes {
		query := hashes[qi]

		bkResult := sortedInts(tree.search(query, maxDist))
		bfResult := sortedInts(bruteForceSearch(hashes, query, maxDist))

		if !intsEqual(bkResult, bfResult) {
			t.Errorf("query index %d (hash %016x, maxDist=%d):\n  BK-tree : %v\n  brute   : %v",
				qi, query, maxDist, bkResult, bfResult)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_LargeInput (1000 items, brute-force correctness at several thresholds)
// ---------------------------------------------------------------------------

func TestBKTree_LargeInput(t *testing.T) {
	const n = 1000

	hashes := make([]uint64, n)
	for i := range hashes {
		hashes[i] = deterministicHash(i * 31)
	}

	tree := buildTree(hashes)

	// Test a sample of 20 evenly-spaced queries at three different thresholds.
	thresholds := []int{0, 4, 8}
	step := n / 20

	for _, maxDist := range thresholds {
		for qi := 0; qi < n; qi += step {
			query := hashes[qi]

			bkResult := sortedInts(tree.search(query, maxDist))
			bfResult := sortedInts(bruteForceSearch(hashes, query, maxDist))

			if !intsEqual(bkResult, bfResult) {
				t.Errorf("large-input: query[%d] maxDist=%d:\n  BK-tree : %v\n  brute   : %v",
					qi, maxDist, bkResult, bfResult)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_MaxDistZero_NoFalsePositives
// ---------------------------------------------------------------------------

// Ensures maxDist=0 never returns a node whose hash differs even by 1 bit.
func TestBKTree_MaxDistZero_NoFalsePositives(t *testing.T) {
	const n = 200

	hashes := make([]uint64, n)
	for i := range hashes {
		hashes[i] = deterministicHash(i*7 + 3)
	}

	tree := buildTree(hashes)

	for qi := 0; qi < n; qi++ {
		query := hashes[qi]
		got := tree.search(query, 0)
		for _, idx := range got {
			if hashes[idx] != query {
				t.Errorf("maxDist=0 returned idx %d with hash %016x, query was %016x (distance %d)",
					idx, hashes[idx], query, bits.OnesCount64(hashes[idx]^query))
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_SingleNode
// ---------------------------------------------------------------------------

func TestBKTree_SingleNode(t *testing.T) {
	const h = uint64(0xCAFEBABECAFEBABE)
	tree := &bkTree{}
	tree.insert(h, 42)

	// Exact match
	if got := tree.search(h, 0); len(got) != 1 || got[0] != 42 {
		t.Errorf("single node exact search: got %v, want [42]", got)
	}

	// Wide threshold still returns it
	if got := tree.search(h, 64); len(got) != 1 || got[0] != 42 {
		t.Errorf("single node wide search: got %v, want [42]", got)
	}

	// Hash that differs by 1 bit, maxDist=0 — must NOT return index 42
	neighbour := h ^ 1
	if got := tree.search(neighbour, 0); len(got) != 0 {
		t.Errorf("single node: neighbour with maxDist=0 returned %v, want []", got)
	}

	// Hash that differs by 1 bit, maxDist=1 — MUST return index 42
	if got := tree.search(neighbour, 1); len(got) != 1 || got[0] != 42 {
		t.Errorf("single node: neighbour with maxDist=1: got %v, want [42]", got)
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_AllIdenticalHashes
// ---------------------------------------------------------------------------

func TestBKTree_AllIdenticalHashes(t *testing.T) {
	const h = uint64(0x1234567812345678)
	const n = 50

	tree := &bkTree{}
	for i := 0; i < n; i++ {
		tree.insert(h, i)
	}

	got := sortedInts(tree.search(h, 0))
	if len(got) != n {
		t.Fatalf("all-identical: expected %d results, got %d", n, len(got))
	}
	for i, idx := range got {
		if idx != i {
			t.Errorf("all-identical: got[%d]=%d, want %d", i, idx, i)
		}
	}
}

// ---------------------------------------------------------------------------
// TestBKTree_TriangleInequalityBoundary
// ---------------------------------------------------------------------------

// Verifies the pruning boundary: items exactly at distance (maxDist) are included,
// items at (maxDist+1) are excluded — testing the off-by-one boundary.
func TestBKTree_TriangleInequalityBoundary(t *testing.T) {
	base := uint64(0xFFFFFFFFFFFFFFFF)

	// Construct hashes at distances 1..10 from base by flipping i lowest bits.
	const span = 10
	hashes := make([]uint64, span)
	for i := 0; i < span; i++ {
		mask := uint64((1 << uint(i+1)) - 1) // flip (i+1) lowest bits
		hashes[i] = base ^ mask
	}
	tree := buildTree(hashes)

	for maxDist := 1; maxDist <= span; maxDist++ {
		got := sortedInts(tree.search(base, maxDist))
		want := sortedInts(bruteForceSearch(hashes, base, maxDist))
		if !intsEqual(got, want) {
			t.Errorf("boundary maxDist=%d: BK-tree %v != brute-force %v", maxDist, got, want)
		}
	}
}
