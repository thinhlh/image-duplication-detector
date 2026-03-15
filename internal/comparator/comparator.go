package comparator

import (
	"math/bits"
	"sort"

	"github.com/imgdup/image-dupl-detector/internal/hasher"
	"github.com/imgdup/image-dupl-detector/internal/scanner"
)

// DuplicateGroup represents a set of visually similar files.
type DuplicateGroup struct {
	ID         int
	Files      []RankedFile
	Similarity float64 // 0.0–100.0
}

// RankedFile is a file within a duplicate group with its rank information.
type RankedFile struct {
	Info   scanner.FileInfo
	Hash   hasher.HashResult
	IsKeep bool // true for the first/largest file in the group
}

// Compare takes all HashResults, groups duplicates, and returns sorted groups.
func Compare(results []hasher.HashResult, similarityPct int) []DuplicateGroup {
	if len(results) == 0 {
		return nil
	}

	// Phase 1: MD5 exact-match grouping (O(n))
	md5Groups := make(map[string][]int) // md5 -> indices
	for i, r := range results {
		if r.MD5 != "" {
			md5Groups[r.MD5] = append(md5Groups[r.MD5], i)
		}
	}

	// Union-Find for grouping
	parent := make([]int, len(results))
	for i := range parent {
		parent[i] = i
	}

	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}

	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[ra] = rb
		}
	}

	// Merge exact MD5 matches
	for _, indices := range md5Groups {
		if len(indices) > 1 {
			for i := 1; i < len(indices); i++ {
				union(indices[0], indices[i])
			}
		}
	}

	// Phase 2: BK-tree pHash search for images (O(n log n) average).
	// maxDist: largest Hamming distance that still meets the similarity threshold.
	// similarity = (1 - dist/64) * 100 >= similarityPct
	// → dist <= 64 * (1 - similarityPct/100)
	maxDist := int(float64(64) * (1.0 - float64(similarityPct)/100.0))

	tree := &bkTree{}
	for i, r := range results {
		if r.FileInfo.MediaType == scanner.MediaImage {
			tree.insert(r.PHash, i)
		}
	}
	for i, r := range results {
		if r.FileInfo.MediaType != scanner.MediaImage {
			continue
		}
		for _, j := range tree.search(r.PHash, maxDist) {
			if j <= i {
				continue // skip self and already-processed pairs
			}
			union(i, j)
		}
	}

	// Phase 3: Video comparison using frame hashes
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			ri := results[i]
			rj := results[j]

			if ri.FileInfo.MediaType != scanner.MediaVideo || rj.FileInfo.MediaType != scanner.MediaVideo {
				continue
			}
			if len(ri.FrameHashes) == 0 || len(rj.FrameHashes) == 0 {
				continue
			}

			sim := videoSimilarity(ri.FrameHashes, rj.FrameHashes)
			if sim >= float64(similarityPct) {
				union(i, j)
			}
		}
	}

	// Collect groups
	groupMap := make(map[int][]int) // root -> member indices
	for i := range results {
		root := find(i)
		groupMap[root] = append(groupMap[root], i)
	}

	// Build DuplicateGroups (only groups with 2+ members)
	var groups []DuplicateGroup
	groupID := 1

	for _, indices := range groupMap {
		if len(indices) < 2 {
			continue
		}

		// Sort members by size descending (largest = KEEP)
		sort.Slice(indices, func(a, b int) bool {
			return results[indices[a]].FileInfo.Size > results[indices[b]].FileInfo.Size
		})

		// Compute group similarity: minimum among directly-connected pairs only
		sim := groupSimilarity(results, indices, maxDist)

		files := make([]RankedFile, len(indices))
		for k, idx := range indices {
			files[k] = RankedFile{
				Info:   results[idx].FileInfo,
				Hash:   results[idx],
				IsKeep: k == 0,
			}
		}

		groups = append(groups, DuplicateGroup{
			ID:         groupID,
			Files:      files,
			Similarity: sim,
		})
		groupID++
	}

	// Sort groups by descending recoverable space
	sort.Slice(groups, func(i, j int) bool {
		return recoverableSpace(groups[i]) > recoverableSpace(groups[j])
	})

	// Re-number after sort
	for i := range groups {
		groups[i].ID = i + 1
	}

	return groups
}

// hammingDistance computes the number of differing bits between two uint64 values.
func hammingDistance(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// pHashSimilarity converts Hamming distance to a similarity percentage.
func pHashSimilarity(a, b uint64) float64 {
	dist := hammingDistance(a, b)
	return (1.0 - float64(dist)/64.0) * 100.0
}

// videoSimilarity computes mean per-frame similarity between two videos.
func videoSimilarity(a, b []uint64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	total := 0.0
	for i := 0; i < n; i++ {
		total += pHashSimilarity(a[i], b[i])
	}
	return total / float64(n)
}

// groupSimilarity returns the minimum similarity among pairs that are DIRECTLY
// within the threshold (Hamming distance ≤ maxDist). Transitive members that
// are not directly similar to each other are excluded from the minimum, so the
// reported value is always ≥ the user-specified threshold.
func groupSimilarity(results []hasher.HashResult, indices []int, maxDist int) float64 {
	if len(indices) < 2 {
		return 100.0
	}

	// All same MD5 → exact duplicates
	md5 := results[indices[0]].MD5
	allSameMD5 := md5 != ""
	for _, idx := range indices[1:] {
		if results[idx].MD5 != md5 {
			allSameMD5 = false
			break
		}
	}
	if allSameMD5 {
		return 100.0
	}

	// Minimum similarity only among pairs that are directly within the threshold
	minSim := 100.0
	for i := 0; i < len(indices); i++ {
		for j := i + 1; j < len(indices); j++ {
			ri := results[indices[i]]
			rj := results[indices[j]]

			var sim float64
			if ri.FileInfo.MediaType == scanner.MediaVideo && len(ri.FrameHashes) > 0 {
				sim = videoSimilarity(ri.FrameHashes, rj.FrameHashes)
				// Only count video pairs within threshold
				if sim < (1.0-float64(maxDist)/64.0)*100.0 {
					continue
				}
			} else {
				dist := hammingDistance(ri.PHash, rj.PHash)
				if dist > maxDist {
					continue // transitive member — skip for similarity display
				}
				sim = pHashSimilarity(ri.PHash, rj.PHash)
			}
			if sim < minSim {
				minSim = sim
			}
		}
	}
	return minSim
}

// recoverableSpace returns the total size of non-KEEP files in a group.
func recoverableSpace(g DuplicateGroup) int64 {
	var total int64
	for i, f := range g.Files {
		if i > 0 { // skip KEEP (index 0)
			total += f.Info.Size
		}
	}
	return total
}

// TotalRecoverableSpace sums recoverable space across all groups.
func TotalRecoverableSpace(groups []DuplicateGroup) int64 {
	var total int64
	for _, g := range groups {
		total += recoverableSpace(g)
	}
	return total
}

// DuplicateFileCount returns the total number of duplicate (non-KEEP) files.
func DuplicateFileCount(groups []DuplicateGroup) int {
	count := 0
	for _, g := range groups {
		if len(g.Files) > 1 {
			count += len(g.Files) - 1
		}
	}
	return count
}
