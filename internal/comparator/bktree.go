package comparator

// bkNode is a node in a BK-tree for Hamming-distance nearest-neighbour search.
type bkNode struct {
	hash     uint64
	indices  []int      // all result indices sharing this exact hash
	children map[int]*bkNode
}

// bkTree indexes uint64 pHashes by Hamming distance.
// Build: O(n log n) average. Query: O(log n) average.
// Correct because Hamming distance satisfies the triangle inequality.
type bkTree struct {
	root *bkNode
}

func (t *bkTree) insert(hash uint64, idx int) {
	if t.root == nil {
		t.root = &bkNode{hash: hash, indices: []int{idx}, children: make(map[int]*bkNode)}
		return
	}
	cur := t.root
	for {
		dist := hammingDistance(cur.hash, hash)
		if dist == 0 {
			// identical hash — store alongside existing node
			cur.indices = append(cur.indices, idx)
			return
		}
		child, ok := cur.children[dist]
		if !ok {
			cur.children[dist] = &bkNode{hash: hash, indices: []int{idx}, children: make(map[int]*bkNode)}
			return
		}
		cur = child
	}
}

// search returns all indices whose hash is within maxDist of query.
// Pruning rule: if dist(node, query)=d, only children at keys [d-maxDist, d+maxDist]
// can possibly contain results (triangle inequality on Hamming distance).
func (t *bkTree) search(query uint64, maxDist int) []int {
	if t.root == nil {
		return nil
	}
	var results []int
	stack := []*bkNode{t.root}
	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		dist := hammingDistance(node.hash, query)
		if dist <= maxDist {
			results = append(results, node.indices...)
		}
		lo := dist - maxDist
		hi := dist + maxDist
		for d, child := range node.children {
			if d >= lo && d <= hi {
				stack = append(stack, child)
			}
		}
	}
	return results
}
