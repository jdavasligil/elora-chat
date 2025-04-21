package cache

// https://arxiv.org/pdf/2310.06663
type Heap struct {
	blockDepth   int64
	blockWidth   int64
	blockSize    int64
	subBlockSize int64
	blockChCount int64
	intraChCount int64
	n            int64
	data         []int
}

// @param blockDepth   Tree depth (6-10) are typically optimal
// @param intraChCount Children per block (2) is typically optimal
func NewHeap(blockDepth int64, intraChCount int64) *Heap {
	h := &Heap{
		blockDepth:   blockDepth,
		blockWidth:   1,
		blockSize:    1,
		intraChCount: intraChCount,
	}

	for range blockDepth {
		h.subBlockSize += h.blockWidth
		h.blockWidth *= intraChCount
		h.blockSize += h.blockWidth
	}
	h.blockChCount = h.blockWidth

	return h
}

// @param I Supernode index
// @param localI Index within the current supernode
func (h *Heap) heapify(I int64, localI int64) {
	// calculate the actual index of the current node in the data array
	index := I*h.blockSize + localI
	// check if it's a block leaf node
	if localI >= h.subBlockSize {
		// the current node is a leaf node
		//
		// calculate the index of its child
		childI := I*h.blockChCount + 1 + (localI - h.subBlockSize)
		childIndex := (childI) * h.blockSize
		// terminate heapify if the node has no child
		if childIndex >= h.n {
			return
		}
		// terminate heapify if no child of the parent has a greater value
		if h.data[childIndex] <= h.data[index] {
			return
		}
		// swap & recursively call heapify
		h.data[index], h.data[childIndex] = h.data[childIndex], h.data[index]
		h.heapify(childI, 0)
		return
	}
	// if it's not a block leaf node, then the mapping to children is done
	// the same way as the regular d-ary heap:
	//
	// calculate the range of its childs
	start := I*h.blockSize + localI*h.intraChCount + 1
	end := min(start+h.intraChCount, h.n)
	// iterate over the calculated range
	mx := h.data[index]
	mx_index := index
	for start < end {
		// compare (set max)
		if h.data[start] > mx {
			mx = h.data[start]
			mx_index = start
		}
		start++
	}
	// terminate heapify if no child of the parent has a greater value
	if mx_index == index {
		return
	}
	// swap & recursively call heapify
	h.data[mx_index], h.data[index] = h.data[index], h.data[mx_index]
	h.heapify(I, mx_index-I*h.blockSize)
}

func (h *Heap) Sort(s []int) {
	h.data = s
	h.n = int64(len(s))
	// build heap (rearranges the array)
	for i := h.n - 1; i >= 0; i-- {
		I := i / h.blockSize
		h.heapify(I, i-I*h.blockSize)
	}

	// extract elements from heap one by one
	for i := h.n - 1; i >= 0; i-- {
		// move current root to end
		s[0], s[i] = s[i], s[0]
		h.n = i

		// call max heapify on the reduced heap
		h.heapify(0, 0)
	}
}
