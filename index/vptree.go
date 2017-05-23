package index

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
)

//Node VPTree node
type Node struct {
	Threshold float32
	Left      *Node
	Right     *Node
	Size      int
	Item      interface{}
}

// A VPTree struct represents a Vantage-point tree. Vantage-point trees are
// useful for nearest-neighbour searches in high-dimensional metric spaces.
type VPTree struct {
	root           *Node
	distanceMetric func(x, y interface{}) float32
}

//MetricCalls increases every time metric of two vectors evaluated
var MetricCalls int

// NewVPTree creates a new VP-tree using the metric and items provided. The metric
// measures the distance between two items, so that the VP-tree can find the
// nearest neighbour(s) of a target item.
func NewVPTree(metric func(x, y interface{}) float32, items []interface{}) (t *VPTree) {
	// make copy of items to not damage original data
	t = &VPTree{
		distanceMetric: metric,
	}
	treeItems := make([]interface{}, len(items))
	copy(treeItems, items)
	t.root = t.buildFromPoints(treeItems)
	return
}

//PrintTree prints tree structure on the console
func (vp *VPTree) PrintTree(n *Node, level, maxLevel int) {
	if n == nil {
		n = vp.root
		fmt.Printf("Tree:\n")
	}
	shift := strings.Repeat(" ", level)
	fmt.Printf("%sNode: %s Size:%d Level:%d Thr:%0.2f\n", shift, n.Item, n.Size, level, n.Threshold)

	if level < maxLevel {
		level = level + 1
		shift = strings.Repeat(" ", level)
		if n.Left != nil {
			fmt.Printf("%sleft: %d\n", shift, n.Left.Size)
			vp.PrintTree(n.Left, level, maxLevel)
		}
		if n.Right != nil {
			fmt.Printf("%sright: %d\n", shift, n.Right.Size)
			vp.PrintTree(n.Right, level, maxLevel)
		}
	}

}

func (vp *VPTree) buildFromPoints(items []interface{}) (n *Node) {
	if len(items) == 0 {
		return nil
	}

	n = &Node{}

	// Take a random item out of the items slice and make it this node's item
	idx := rand.Intn(len(items))
	n.Item = items[idx]
	n.Size = len(items)

	// put last element instead of item
	// remove slice length by 1
	items[idx], items = items[len(items)-1], items[:len(items)-1]

	if len(items) > 0 {
		// Now partition the items into two equal-sized sets, one
		// closer to the node's item than the median, and one farther
		// away.
		median := len(items) / 2
		MetricCalls++
		// distance to random median item
		pivotDist := vp.distanceMetric(items[median], n.Item)
		//fmt.Printf("Median l:%d %d %s pivotd: %0.2f\n", len(items), median, items[median], pivotDist)

		// put median item to the end of slice and
		// end item replaces previous median
		items[median], items[len(items)-1] = items[len(items)-1], items[median]

		storeIndex := 0
		// go thought all items excluding median and now excluding item itself
		for i := 0; i < len(items)-1; i++ {
			MetricCalls++
			if vp.distanceMetric(items[i], n.Item) <= pivotDist {
				// if some item closer than median to the item itself
				// then put this item to the starting part of a slice
				// and item at storeindex (farer than median) instead of item
				items[storeIndex], items[i] = items[i], items[storeIndex]
				storeIndex++
			}
		}
		// swap median item (which is at the end of slice) and item at the end of closer items list
		items[len(items)-1], items[storeIndex] = items[storeIndex], items[len(items)-1]
		// so now median is at storeIndex position of a slice
		median = storeIndex
		MetricCalls++
		// we can reuse threshold
		n.Threshold = pivotDist
		//fmt.Printf("It %s pivotd thresh: %0.2f\n", n.Item, pivotDist)
		n.Left = vp.buildFromPoints(items[:median])
		n.Right = vp.buildFromPoints(items[median:])
	}
	return
}

/*
//ComputeDensity compute density for all points for specific cutoff
func (vp *VPTree) computeDensity(progress bool, k int, cutoff float32, labelProvider func(id uint32) string) {
	if k < 1 {
		return
	}

	sync := make(chan *PriorityQueue, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		h := make(PriorityQueue, 0, k)
		sync <- &h
	}

	for _, n := range vp.ID2node {

		go func(n *Node, sync chan *PriorityQueue, pq *PriorityQueue) {
			tau := cutoff
			vp.search(vp.root, n.Item, k, pq, &tau)
			//println(len(*pq))
			for len(*pq) > 1 { // except itself is for 1
				//hi :=
				(*pq).Pop()
				//n.Item.SetDensity(n.Item.Density() + 1 + 1/(1+hi.Dist))
				//if vp.distanceMetric == P.CosineMetric {
				//	n.Item.Density += 1 + (1 - hi.Dist) // max distance is 1!!!!!!!
			}

			*pq = (*pq)[:0] // clear priority queue
			sync <- pq
		}(n, sync, <-sync)
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		<-sync
	}

}
*/
// Search searches the VP-tree for the k nearest neighbours of target. It
// returns the up to k narest neighbours and the corresponding distances in
// order of least distance to largest distance.
func (vp *VPTree) Search(target interface{}, k int, cutoff float32) (results []interface{}, distances []float32) {
	if k < 1 {
		return
	}

	h := make(PriorityQueue, 0, k)
	var tau float32 = math.MaxFloat32
	if cutoff > 0 {
		tau = cutoff
	}
	// we search k + 1 because we will exclude item itself from search result
	vp.search(vp.root, target, k+1, &h, &tau)

	for len(h) > 0 {
		hi := h.Pop()
		results = append(results, hi.Item)
		distances = append(distances, hi.Dist)
	}

	// Reverse results and distances, because we popped them from the heap
	// in large-to-small order
	for i, j := 0, len(results)-1; i < j; i, j = i+1, j-1 {
		results[i], results[j] = results[j], results[i]
		distances[i], distances[j] = distances[j], distances[i]
	}

	return
}

func (vp *VPTree) search(n *Node, target interface{}, k int, h *PriorityQueue, tau *float32) {
	var d float32
	if n.Item != target {
		MetricCalls++
		d = vp.distanceMetric(n.Item, target)
		if d < *tau {
			if len(*h) == k {
				h.Pop()
			}
			h.Push(&HeapItem{n.Item, d})

			if len(*h) == k {
				*tau = h.Top().Dist
			}
		}
	} else {
		d = 0
	}
	//fmt.Printf("%s %s %f \n", n.Item, target, d)
	if d < n.Threshold {
		if d-*tau <= n.Threshold && n.Left != nil {
			//fmt.Printf("11 d:%0.2f th:%0.2f tau:%0.2f domain:%s\n", d, n.Threshold, *tau, n.Item)
			vp.search(n.Left, target, k, h, tau)
		}

		if d+*tau >= n.Threshold && n.Right != nil {
			//fmt.Printf("12 d:%0.2f th:%0.2f tau:%0.2f domain:%s\n", d, n.Threshold, *tau, n.Item)
			vp.search(n.Right, target, k, h, tau)
		}
	} else {
		if d+*tau >= n.Threshold && n.Right != nil {
			//fmt.Printf("21 d:%0.2f th:%0.2f tau:%0.2f domain:%s\n", d, n.Threshold, *tau, n.Item)
			vp.search(n.Right, target, k, h, tau)
		}

		if d-*tau <= n.Threshold && n.Left != nil {
			//fmt.Printf("22 d:%0.2f th:%0.2f tau:%0.2f domain:%s\n", d, n.Threshold, *tau, n.Item)
			vp.search(n.Left, target, k, h, tau)
		}
	}
}

/*
//ComputeDelta compute delta for all points for specific cutoff
func (vp *VPTree) computeDelta(progress bool, k int, cutoff float32, labelProvider func(id uint32) string) {
	if k < 1 {
		return
	}

	sync := make(chan *PriorityQueue, runtime.NumCPU())
	for i := 0; i < runtime.NumCPU(); i++ {
		h := make(PriorityQueue, 0, k)
		sync <- &h
	}

	for _, n := range vp.ID2node {

		//if n.Item.Density > 0 && n.Item.Delta == -1 {
		go func(n *Node, sync chan *PriorityQueue, pq *PriorityQueue) {
			tau := cutoff
			vp.search(vp.root, n.Item, k, pq, &tau)

			for len(*pq) > 1 {
				//hi :=

				(*pq).Pop()
				//println(hi.Item.Density, n.Item.Density)
				//if hi.Item.Density() > n.Item.Density() {
				//	n.Item.SetDelta(hi.Dist)
				//	n.Item.SetNearest(hi.Item.ID())
				//	break
				//}
			}

			*pq = (*pq)[:0] // clear priority queue
			//if n.Item.Density > 1 {
			//	fmt.Println(n.Item.Density, n.Item.Delta, n.Item.Nearest, labelProvider(n.Item.ID))
			//}
			sync <- pq
		}(n, sync, <-sync)
		//}
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		<-sync
	}

}
*/
/*
//Assign assign point to cluster
func (vp *VPTree) assign(p P.Point, cutoff float32) uint32 {
	switch {
	case p.Cluster() != math.MaxUint32: // point already assigned to cluster
		return p.Cluster()
	case p.Density() == 0: // point is singleton
		return math.MaxUint32
	case p.Delta() >= cutoff: // this point is GROSS-CENTROID
		// mark all points under this point as part of a cluster within cutoff
		//fmt.Println("Centroid!", p.Density, p.ID)
		p.SetCluster(p.ID())
		suzerens, _ := vp.Search(p, 10000, cutoff)
		for _, suzeren := range suzerens {
			suzeren.SetCluster(p.ID())
		}
		return p.Cluster()
	case p.Delta() < cutoff: // point is part of a cluster
		//println(p.Delta(), cutoff, p.Nearest)
		p.SetCluster(vp.assign(vp.ID2node[p.Nearest()].Item, cutoff))
		return p.Cluster()

	/*case p.Density > 0 && p.Delta < cutoff: // point is part of a cluster
		p.Cluster = vp.assign(vp.ID2node[p.Nearest].Item, cutoff)
		return p.Cluster
	case p.Density > 0 && p.Delta > cutoff: // this is a small cluster
		// mark all points under this point as part of a cluster within cutoff
		p.Cluster = p.ID
		suzerens, _ := vp.Search(p, int(p.Density)+1, cutoff)
		for _, suzeren := range suzerens {
			suzeren.Cluster = p.ID
		}
		return p.Cluster*/
/*default:
		fmt.Println("PANICEBT")
		panic("HM!!!!")
	}
}

//ComputeClusters compute delta for all points for specific cutoff
func (vp *VPTree) computeClusters(progress bool, k int, cutoff float32, labelProvider func(id uint32) string) {

	bar := pb.New(len(vp.ID2node))
	bar.SetRefreshRate(time.Second)
	bar.ShowTimeLeft = true
	bar.ShowSpeed = true
	if progress {
		bar.Start()
	}
	for _, n := range vp.ID2node {
		bar.Increment()
		vp.assign(n.Item, cutoff)
	}
	if progress {
		bar.FinishPrint("Cluster computation finished!")
	}
}

//Clusterize - partition points into clusters
func Clusterize(progress bool, points []P.Point, dc float32, metric P.Metric) (clusters map[uint32][]P.Point) {
	for _, p := range points {
		p.SetDelta(float32(math.MaxUint32))
		p.SetNearest(math.MaxUint32)
		p.SetCluster(math.MaxUint32)
		p.SetDensity(0)
	}

	index := NewVPTree(metric, points)
	index.computeDensity(progress, len(points), dc, nil)
	index.computeDelta(progress, len(points), dc, nil)
	index.computeClusters(progress, len(points), dc, nil)

	clusters = make(map[uint32][]P.Point)
	// collect clusters
	for _, p := range points {
		if p.Cluster() != math.MaxUint32 {
			clusters[p.Cluster()] = append(clusters[p.Cluster()], p)
		}
	}
	return
}
*/
