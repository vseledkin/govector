package index

import (
	"fmt"
)

type HeapItem struct {
	Item  interface{}
	Dist  float32
	Index int
}

// A PriorityQueue implements heap.Interface and holds Items.
type PriorityQueue []*HeapItem

func (pq PriorityQueue) Len() int { return len(pq) }

func (pq PriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the highest, not lowest, priority so we use greater than here.
	return pq[i].Dist > pq[j].Dist
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq PriorityQueue) Top() *HeapItem {
	return pq[0]
}

func (pq *PriorityQueue) Push(x interface{}) {
	item := x.(*HeapItem)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *PriorityQueue) Find(item interface{}) *HeapItem {
	for i := range *pq {
		if (*pq)[i].Item == item {
			return (*pq)[i]
		}
	}
	panic(fmt.Errorf("Item not found"))
}

func (pq *PriorityQueue) Print(label func(*HeapItem) string) {
	for i, hi := range *pq {
		fmt.Printf("\t%d %d %f %s\n", i, hi.Index, hi.Dist, label(hi))
	}
}

// A PriorityQueue implements heap.Interface and holds Items.
type MinPriorityQueue []*HeapItem

func (pq MinPriorityQueue) Len() int { return len(pq) }

func (pq MinPriorityQueue) Less(i, j int) bool {
	// We want Pop to give us the lowest, priority so we use less than here.
	return pq[i].Dist < pq[j].Dist
}

func (pq MinPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].Index = i
	pq[j].Index = j
}

func (pq MinPriorityQueue) Top() *HeapItem {
	return pq[0]
}

func (pq *MinPriorityQueue) Push(x interface{}) {
	item := x.(*HeapItem)
	item.Index = len(*pq)
	*pq = append(*pq, item)
}

func (pq *MinPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.Index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *MinPriorityQueue) Find(item interface{}) *HeapItem {
	for i := range *pq {
		if (*pq)[i].Item == item {
			return (*pq)[i]
		}
	}
	panic(fmt.Errorf("Item not found"))
}

func (pq *MinPriorityQueue) Print(label func(*HeapItem) string) {
	for i, hi := range *pq {
		fmt.Printf("\t%d %d %f %s\n", i, hi.Index, hi.Dist, label(hi))
	}
}
