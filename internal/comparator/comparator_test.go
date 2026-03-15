package comparator

import (
	"fmt"
	"testing"
	"time"

	"github.com/imgdup/image-dupl-detector/internal/hasher"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
)

func makeImageResult(path string, size int64, md5 string, phash uint64) hasher.HashResult {
	return hasher.HashResult{
		FileInfo: scanner.FileInfo{
			Path:      path,
			MediaType: scanner.MediaImage,
			Size:      size,
			ModTime:   time.Now(),
		},
		MD5:   md5,
		PHash: phash,
	}
}

func TestCompare_ExactMD5Duplicates(t *testing.T) {
	results := []hasher.HashResult{
		makeImageResult("/a/img1.jpg", 5000, "abc123", 0xDEADBEEF00000000),
		makeImageResult("/a/img2.jpg", 4800, "abc123", 0xDEADBEEF00000000), // same MD5
		makeImageResult("/a/img3.jpg", 3000, "xyz999", 0x1111111111111111), // different
	}

	groups := Compare(results, 90)

	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Files) != 2 {
		t.Errorf("expected 2 files in group, got %d", len(groups[0].Files))
	}
	if groups[0].Similarity != 100.0 {
		t.Errorf("expected 100%% similarity for MD5 match, got %.1f%%", groups[0].Similarity)
	}
}

func TestCompare_KeepIsLargest(t *testing.T) {
	results := []hasher.HashResult{
		makeImageResult("/small.jpg", 1000, "dup", 0xFFFFFFFFFFFFFFFF),
		makeImageResult("/large.jpg", 9000, "dup", 0xFFFFFFFFFFFFFFFF),
	}

	groups := Compare(results, 90)

	if len(groups) == 0 {
		t.Fatal("expected 1 group")
	}
	keep := groups[0].Files[0]
	if !keep.IsKeep {
		t.Error("first file should be IsKeep=true")
	}
	if keep.Info.Path != "/large.jpg" {
		t.Errorf("largest file should be KEEP, got %s", keep.Info.Path)
	}
}

func TestCompare_NoDuplicates(t *testing.T) {
	// All very different pHashes
	results := []hasher.HashResult{
		makeImageResult("/a.jpg", 1000, "md5a", 0x0000000000000000),
		makeImageResult("/b.jpg", 1000, "md5b", 0xFFFFFFFFFFFFFFFF),
		makeImageResult("/c.jpg", 1000, "md5c", 0x5555555555555555),
	}

	groups := Compare(results, 90)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for distinct images, got %d", len(groups))
	}
}

func TestCompare_PHashSimilarity(t *testing.T) {
	// Hamming distance 6 → similarity 90.6% — should be grouped at 90% threshold
	// maxDist = int(64 * (1 - 90/100)) = int(6.4) = 6
	base := uint64(0xFFFFFFFFFFFFFFFF)
	close := base ^ 0x3F // flip 6 bits (distance=6, sim=90.6%)

	results := []hasher.HashResult{
		makeImageResult("/orig.jpg", 5000, "md5a", base),
		makeImageResult("/similar.jpg", 4000, "md5b", close),
	}

	groups := Compare(results, 90)

	if len(groups) != 1 {
		t.Errorf("expected 1 group for distance-6 pHash pair at 90%% threshold, got %d", len(groups))
	}
	if len(groups) > 0 && groups[0].Similarity < 90.0 {
		t.Errorf("group similarity %.1f%% is below the 90%% threshold", groups[0].Similarity)
	}
}

func TestCompare_GroupSimilarityNeverBelowThreshold(t *testing.T) {
	// Three files: A-B at 95%, A-C at 91%, B-C at 78% (below threshold)
	// Union-Find groups all three, but reported similarity should be >= 90%
	a := uint64(0xFFFFFFFFFFFFFFFF)
	b := a ^ 0x3  // dist 2 → 96.9%
	c := a ^ 0x3F // dist 6 → 90.6%

	results := []hasher.HashResult{
		makeImageResult("/a.jpg", 5000, "md5a", a),
		makeImageResult("/b.jpg", 4000, "md5b", b),
		makeImageResult("/c.jpg", 3000, "md5c", c),
	}

	groups := Compare(results, 90)

	for _, g := range groups {
		if g.Similarity < 90.0 {
			t.Errorf("group similarity %.1f%% is below the 90%% threshold", g.Similarity)
		}
	}
}

func TestCompare_PHashBelowThreshold(t *testing.T) {
	// Hamming distance 10 → similarity 84.4% — should NOT be grouped at 90%
	base := uint64(0xFFFFFFFFFFFFFFFF)
	far := base ^ 0x3FF // flip 10 bits

	results := []hasher.HashResult{
		makeImageResult("/orig.jpg", 5000, "md5a", base),
		makeImageResult("/different.jpg", 4000, "md5b", far),
	}

	groups := Compare(results, 90)

	if len(groups) != 0 {
		t.Errorf("expected 0 groups for distance-10 pair at 90%% threshold, got %d", len(groups))
	}
}

func TestCompare_EmptyInput(t *testing.T) {
	groups := Compare(nil, 90)
	if groups != nil && len(groups) != 0 {
		t.Error("expected empty result for nil input")
	}
}

func TestCompare_SingleFile(t *testing.T) {
	results := []hasher.HashResult{
		makeImageResult("/only.jpg", 5000, "md5x", 0xDEADBEEF),
	}
	groups := Compare(results, 90)
	if len(groups) != 0 {
		t.Errorf("single file cannot be a duplicate, expected 0 groups, got %d", len(groups))
	}
}

func TestHammingDistance(t *testing.T) {
	cases := []struct {
		a, b uint64
		want int
	}{
		{0, 0, 0},
		{0xFFFFFFFFFFFFFFFF, 0, 64},
		{0xFF, 0x00, 8},
		{0b1010, 0b1100, 2},
	}
	for _, tc := range cases {
		got := hammingDistance(tc.a, tc.b)
		if got != tc.want {
			t.Errorf("hammingDistance(%064b, %064b) = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestPHashSimilarity(t *testing.T) {
	// Identical hashes → 100%
	if sim := pHashSimilarity(0xABC, 0xABC); sim != 100.0 {
		t.Errorf("identical hashes: expected 100%%, got %.2f%%", sim)
	}
	// Completely different → 0%
	if sim := pHashSimilarity(0, 0xFFFFFFFFFFFFFFFF); sim != 0.0 {
		t.Errorf("max distance: expected 0%%, got %.2f%%", sim)
	}
	// 6 bits different → ~90.6%
	a := uint64(0xFFFFFFFFFFFFFFFF)
	b := a ^ 0x3F // 6 bits flipped
	sim := pHashSimilarity(a, b)
	if sim < 90.0 || sim > 91.5 {
		t.Errorf("6-bit distance: expected ~90.6%%, got %.2f%%", sim)
	}
}

func TestTotalRecoverableSpace(t *testing.T) {
	groups := []DuplicateGroup{
		{Files: []RankedFile{
			{Info: scanner.FileInfo{Size: 10000}, IsKeep: true},
			{Info: scanner.FileInfo{Size: 5000}, IsKeep: false},
			{Info: scanner.FileInfo{Size: 3000}, IsKeep: false},
		}},
	}
	space := TotalRecoverableSpace(groups)
	if space != 8000 {
		t.Errorf("expected 8000 recoverable bytes, got %d", space)
	}
}

func TestDuplicateFileCount(t *testing.T) {
	groups := []DuplicateGroup{
		{Files: []RankedFile{{IsKeep: true}, {IsKeep: false}, {IsKeep: false}}},
		{Files: []RankedFile{{IsKeep: true}, {IsKeep: false}}},
	}
	count := DuplicateFileCount(groups)
	if count != 3 {
		t.Errorf("expected 3 duplicate files, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// TestCompare_AllSamePHash
// ---------------------------------------------------------------------------

// All n files carry an identical pHash and distinct MD5s: they must collapse
// into a single group of n with 100 % similarity.
func TestCompare_AllSamePHash(t *testing.T) {
	const n = 6
	const sharedHash = uint64(0xABCDEF0123456789)

	results := make([]hasher.HashResult, n)
	for i := range results {
		results[i] = makeImageResult(
			fmt.Sprintf("/img%d.jpg", i),
			int64(1000+i*500), // distinct sizes
			fmt.Sprintf("md5-%d", i),
			sharedHash,
		)
	}

	groups := Compare(results, 90)

	if len(groups) != 1 {
		t.Fatalf("all-same pHash: expected 1 group, got %d", len(groups))
	}
	if len(groups[0].Files) != n {
		t.Errorf("all-same pHash: expected %d files in group, got %d", n, len(groups[0].Files))
	}
	if groups[0].Similarity != 100.0 {
		t.Errorf("all-same pHash: expected 100%% similarity, got %.2f%%", groups[0].Similarity)
	}
}

// ---------------------------------------------------------------------------
// TestCompare_LargestFileIsAlwaysKeep
// ---------------------------------------------------------------------------

// Even when the input order is shuffled the file with the largest byte-size
// must always be designated IsKeep=true (i.e. first in group.Files).
func TestCompare_LargestFileIsAlwaysKeep(t *testing.T) {
	// All share the same pHash so they all land in one group.
	const sharedHash = uint64(0xFFFFFFFFFFFFFFFF)

	sizes := []int64{2048, 8192, 1024, 16384, 4096} // 16384 is the largest

	results := make([]hasher.HashResult, len(sizes))
	for i, sz := range sizes {
		results[i] = makeImageResult(
			fmt.Sprintf("/file%d.jpg", i),
			sz,
			fmt.Sprintf("md5-%d", i), // distinct so MD5 grouping doesn't interfere
			sharedHash,
		)
	}

	groups := Compare(results, 90)

	if len(groups) == 0 {
		t.Fatal("expected at least 1 group")
	}

	keep := groups[0].Files[0]
	if !keep.IsKeep {
		t.Error("first file in group must be IsKeep=true")
	}

	// Verify it really is the largest.
	maxSize := keep.Info.Size
	for _, f := range groups[0].Files[1:] {
		if f.Info.Size > maxSize {
			t.Errorf("KEEP file (size=%d) is not the largest; found larger file size=%d",
				maxSize, f.Info.Size)
		}
	}

	// Cross-check the actual maximum.
	var want int64
	for _, sz := range sizes {
		if sz > want {
			want = sz
		}
	}
	if keep.Info.Size != want {
		t.Errorf("KEEP file size=%d, expected largest=%d", keep.Info.Size, want)
	}
}

// ---------------------------------------------------------------------------
// TestCompare_GroupsSortedByRecoverableSpace
// ---------------------------------------------------------------------------

// When multiple groups exist the slice returned by Compare must be ordered so
// that groups[0] has the highest recoverable space (sum of non-KEEP file sizes).
func TestCompare_GroupsSortedByRecoverableSpace(t *testing.T) {
	// Group A: one large duplicate → high recoverable space
	//   files: 100 000 B (KEEP) + 90 000 B (dup) → recoverable = 90 000
	// Group B: two tiny duplicates → low recoverable space
	//   files: 2 048 B (KEEP) + 1 500 B (dup) → recoverable = 1 500

	groupAHash := uint64(0x0F0F0F0F0F0F0F0F)
	groupBHash := uint64(0xF0F0F0F0F0F0F0F0)

	// Group A — same MD5 so they are exact duplicates and always grouped
	resultA1 := makeImageResult("/a-keep.jpg", 100_000, "md5-groupA", groupAHash)
	resultA2 := makeImageResult("/a-dup.jpg", 90_000, "md5-groupA", groupAHash)

	// Group B — same MD5
	resultB1 := makeImageResult("/b-keep.jpg", 2_048, "md5-groupB", groupBHash)
	resultB2 := makeImageResult("/b-dup.jpg", 1_500, "md5-groupB", groupBHash)

	groups := Compare([]hasher.HashResult{resultA1, resultA2, resultB1, resultB2}, 90)

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d", len(groups))
	}

	spaceFirst := recoverableSpace(groups[0])
	spaceSecond := recoverableSpace(groups[1])

	if spaceFirst < spaceSecond {
		t.Errorf("groups not sorted by recoverable space: groups[0]=%d < groups[1]=%d",
			spaceFirst, spaceSecond)
	}

	// Sanity: the first group should correspond to group A (recoverable = 90 000)
	if spaceFirst != 90_000 {
		t.Errorf("expected groups[0] recoverable space = 90000, got %d", spaceFirst)
	}
}

// ---------------------------------------------------------------------------
// TestCompare_StressLargeInput
// ---------------------------------------------------------------------------

// Insert a large number of image results split into several identical-hash
// clusters. Verify that exactly that many groups are found and every group's
// KEEP file is the largest one.
func TestCompare_StressLargeInput(t *testing.T) {
	const clustersCount = 10
	const filesPerCluster = 20

	// Each cluster uses a unique hash generated by a simple deterministic spread.
	clusterHashes := make([]uint64, clustersCount)
	for c := range clusterHashes {
		v := uint64(c+1) * 6364136223846793005
		v ^= v >> 27
		clusterHashes[c] = v
	}

	var results []hasher.HashResult
	for c, h := range clusterHashes {
		for f := 0; f < filesPerCluster; f++ {
			size := int64((f + 1) * 1000) // f=filesPerCluster-1 is largest
			results = append(results, makeImageResult(
				fmt.Sprintf("/cluster%d/file%d.jpg", c, f),
				size,
				fmt.Sprintf("md5-%d-%d", c, f), // all distinct MD5s, grouped by pHash
				h,
			))
		}
	}

	groups := Compare(results, 90)

	if len(groups) != clustersCount {
		t.Fatalf("stress: expected %d groups, got %d", clustersCount, len(groups))
	}

	for gi, g := range groups {
		if len(g.Files) != filesPerCluster {
			t.Errorf("group %d: expected %d files, got %d", gi, filesPerCluster, len(g.Files))
		}
		if !g.Files[0].IsKeep {
			t.Errorf("group %d: Files[0].IsKeep is false", gi)
		}
		keepSize := g.Files[0].Info.Size
		for _, f := range g.Files[1:] {
			if f.Info.Size > keepSize {
				t.Errorf("group %d: KEEP size=%d but found larger dup size=%d", gi, keepSize, f.Info.Size)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// TestCompare_ThresholdBoundary
// ---------------------------------------------------------------------------

// Pairs that are exactly at the similarity threshold must be grouped;
// pairs that are one bit outside must not.
func TestCompare_ThresholdBoundary(t *testing.T) {
	// At 90% threshold: maxDist = int(64 * 0.10) = 6
	// A pair at Hamming distance 6 → similarity 90.625% → must group
	// A pair at Hamming distance 7 → similarity 89.0625% → must NOT group

	base := uint64(0xFFFFFFFFFFFFFFFF)
	atBoundary := base ^ 0x3F   // distance 6 — flip 6 lowest bits
	outsideBoundary := base ^ 0x7F // distance 7 — flip 7 lowest bits

	t.Run("exactly at boundary is grouped", func(t *testing.T) {
		results := []hasher.HashResult{
			makeImageResult("/base.jpg", 5000, "md5a", base),
			makeImageResult("/boundary.jpg", 4000, "md5b", atBoundary),
		}
		groups := Compare(results, 90)
		if len(groups) != 1 {
			t.Errorf("distance-6 pair at 90%% threshold: expected 1 group, got %d", len(groups))
		}
	})

	t.Run("one bit outside boundary is not grouped", func(t *testing.T) {
		results := []hasher.HashResult{
			makeImageResult("/base.jpg", 5000, "md5a", base),
			makeImageResult("/outside.jpg", 4000, "md5b", outsideBoundary),
		}
		groups := Compare(results, 90)
		if len(groups) != 0 {
			t.Errorf("distance-7 pair at 90%% threshold: expected 0 groups, got %d", len(groups))
		}
	})
}

// ---------------------------------------------------------------------------
// TestCompare_GroupIDsSequential
// ---------------------------------------------------------------------------

// After sorting, group IDs must be reassigned as 1, 2, 3, … without gaps.
func TestCompare_GroupIDsSequential(t *testing.T) {
	const nGroups = 5
	var results []hasher.HashResult
	for g := 0; g < nGroups; g++ {
		// Each pair uses the same MD5 so they are guaranteed to be grouped.
		md5 := fmt.Sprintf("md5-group%d", g)
		hash := uint64(g+1) * 0x1111111111111111
		results = append(results,
			makeImageResult(fmt.Sprintf("/g%d-a.jpg", g), int64(5000+g*1000), md5, hash),
			makeImageResult(fmt.Sprintf("/g%d-b.jpg", g), int64(4000+g*1000), md5, hash),
		)
	}

	groups := Compare(results, 90)

	if len(groups) != nGroups {
		t.Fatalf("expected %d groups, got %d", nGroups, len(groups))
	}
	for i, g := range groups {
		if g.ID != i+1 {
			t.Errorf("groups[%d].ID = %d, want %d", i, g.ID, i+1)
		}
	}
}

// ---------------------------------------------------------------------------
// TestCompare_MixedMD5AndPHash
// ---------------------------------------------------------------------------

// Verify that MD5-identical files and pHash-similar-but-not-identical files
// end up in separate groups when they truly do not overlap.
func TestCompare_MixedMD5AndPHash(t *testing.T) {
	// Group 1 (MD5 exact): two files, same MD5, same hash
	exactMD5 := "exactMD5"
	exactHash := uint64(0xAAAAAAAAAAAAAAAA)

	// Group 2 (pHash similar): two files, distinct MD5s, Hamming distance 4
	base2 := uint64(0xBBBBBBBBBBBBBBBB)
	close2 := base2 ^ 0xF // distance 4 → 93.75% similarity

	// Unrelated singleton
	singleton := uint64(0x0000000000000000)

	results := []hasher.HashResult{
		makeImageResult("/e1.jpg", 9000, exactMD5, exactHash),
		makeImageResult("/e2.jpg", 8000, exactMD5, exactHash),
		makeImageResult("/p1.jpg", 7000, "md5-p1", base2),
		makeImageResult("/p2.jpg", 6000, "md5-p2", close2),
		makeImageResult("/solo.jpg", 5000, "md5-solo", singleton),
	}

	groups := Compare(results, 90)

	// Expect exactly 2 groups (the singleton is alone).
	if len(groups) != 2 {
		t.Errorf("expected 2 groups (MD5-exact + pHash-similar), got %d", len(groups))
	}

	// Every group must have exactly 2 files.
	for _, g := range groups {
		if len(g.Files) != 2 {
			t.Errorf("group %d: expected 2 files, got %d", g.ID, len(g.Files))
		}
	}
}
