package govector

import (
	"fmt"
	"log"

	"container/heap"

	"time"

	"github.com/vseledkin/govector/annoy"
	"github.com/vseledkin/govector/index"
)

const (
	UNDEFINED float32 = -1
)

type Point struct {
	Item                 string
	ReachabilityDistance float32
	CoreDistance         float32
	Processed            bool
	NeiboursCount        int
	Vector               []float32
}

func next(pq index.MinPriorityQueue) *Point {
	for _, item := range pq {
		if !item.Item.(*Point).Processed {
			return item.Item.(*Point)
		}
	}
	return nil
}

/*
ComputeClusters
	epsilon - the maximum distance (radius) to consider
	MinPts  - the number of points required to form a cluster.
*/
func (m *Manifold) ComputeClusters(epsilon float32, MinPts int) {
	/*defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			fmt.Println("Failed to create index")

		}
	}()*/
	//var max uint32 = 1000
	var max uint32 = 100000
	if m.WordCount() < max {
		max = m.WordCount()
	}

	points := make([]interface{}, max)

	log.Printf("Reading %d words", max)
	var i uint32 = 0
	m.VisitWords(func(key string) bool {
		if len(key) == 0 {
			panic(fmt.Errorf("Empty key"))
		}
		points[i] = &Point{key, UNDEFINED, UNDEFINED, false, 0, nil}
		i++
		if i == max {
			return false
		}
		if i%1e4 == 0 {
			log.Printf("Read %d words", i)
		}
		return true
	})

	log.Printf("Read %d words", i)
	searchLimit := 200
	idx := annoy.NewAnnoyIndexAngular(128)
	start := time.Now()
	for i, pp := range points {
		v, e := m.GetVector(pp.(*Point).Item)
		if e != nil {
			panic(e)
		}
		pp.(*Point).Vector = v
		idx.AddItem(i, pp.(*Point).Vector)
	}
	readTime := time.Now().Sub(start).Seconds()
	start = time.Now()
	idx.Build(16)
	indexTime := time.Now().Sub(start).Seconds()
	search := func(v []float32) ([]interface{}, []float32) {
		var distances []float32
		var near []int
		var nearest []interface{}
		idx.GetNnsByVector(v, searchLimit, -1, &near, &distances)
		for i, j := range near {
			if distances[i] > epsilon {
				return nearest[:i], distances[:i]
			}
			nearest = append(nearest, points[j])
		}
		return nearest, distances
	}
	var orderedList []interface{}
	// Optics
	start = time.Now()
	for _, pp := range points {
		p := pp.(*Point)
		//fmt.Printf("PROCESS: p -> %d-%s CoreD:%f ReachD:%f N:%d %v\n", j, p.Item, p.CoreDistance, p.ReachabilityDistance, p.NeiboursCount, p.Processed)
		if p.Processed {
			continue
		}
		N, distances := search(p.Vector)
		//fmt.Printf("<<<<<<<<<\n")
		//fmt.Printf("Near: %s are:\n", p.Item)
		//for i, ppp := range N {
		//	fmt.Printf("%f %s\n", distances[i], ppp.(*Point).Item)

		//}
		p.Processed = true
		p.NeiboursCount = len(N)
		if len(N) >= MinPts {
			p.CoreDistance = distances[MinPts-1]
			p.ReachabilityDistance = 0 //p.CoreDistance // distance to itself is zero
		}
		//orderedList = append(orderedList, p)
		if p.CoreDistance != UNDEFINED {

			var seeds index.MinPriorityQueue
			heap.Push(&seeds, &index.HeapItem{p, p.ReachabilityDistance, -1})

			m.update(N, p, &seeds, epsilon, MinPts)
			for q := next(seeds); q != nil; q = next(seeds) {
				//for _, qhi := range seeds {
				//q := heap.Pop(&seeds).(*index.HeapItem).Item.(*Point)
				NQ, nqd := search(q.Vector)
				//fmt.Printf(">>>>>>>>> %d\n", len(seeds))
				//fmt.Printf("Near: %s are:\n", q.Item)
				//for i, ppp := range NQ {
				//	fmt.Printf("%f %s\n", nqd[i], ppp.(*Point).Item)

				//}
				q.Processed = true
				q.NeiboursCount = len(NQ)
				if len(NQ) >= MinPts {
					q.CoreDistance = nqd[MinPts-1]
				}
				//if q.ReachabilityDistance != UNDEFINED {
				//	heap.Push(&out, &index.HeapItem{q, q.ReachabilityDistance, -1})
				//}
				//orderedList = append(orderedList, q)
				if q.CoreDistance != UNDEFINED {
					m.update(NQ, q, &seeds, epsilon, MinPts)
				}
			}
			for len(seeds) > 0 {
				pout := heap.Pop(&seeds).(*index.HeapItem).Item.(*Point)
				orderedList = append(orderedList, pout)
			}
			orderedList = append(orderedList, nil)
			//fmt.Printf("Finish proc SEEDS l:%d\n", len(seeds))
		}

		/*
			if len(nearestPoints) > MinPts {
				fmt.Printf("%d-%s %f\n", j, p.(*Point).Item, p.(*Point).CoreDistance)
				for i, np := range nearestPoints {
					fmt.Printf("\t%d %s %f\n", i, np.(*Point).Item, distances[i])
				}
			}*/
	}
	//for j, p := range points {
	//	fmt.Printf("p -> %d-%s CoreD:%f ReachD:%f N:%d %v\n", j, p.(*Point).Item, p.(*Point).CoreDistance, p.(*Point).ReachabilityDistance, p.(*Point).NeiboursCount, p.(*Point).Processed)
	//}
	clusterTime := time.Now().Sub(start).Seconds()
	fmt.Printf("--------\n")
	outCount := 0
	for _, p := range orderedList {
		if p == nil {
			fmt.Printf("------------------\n")
			continue
		}
		if p.(*Point).ReachabilityDistance == UNDEFINED {
			fmt.Printf("o -> %s %d\n", p.(*Point).Item, p.(*Point).NeiboursCount)
			//fmt.Printf("o -> %s CoreD:%f ReachD:%f N:%d %v\n", p.(*Point).Item, p.(*Point).CoreDistance, p.(*Point).ReachabilityDistance, p.(*Point).NeiboursCount, p.(*Point).Processed)
		} else {
			fmt.Printf("o -> %d-%s CoreD:%f ReachD:%f N:%d %v\n", outCount, p.(*Point).Item, p.(*Point).CoreDistance, p.(*Point).ReachabilityDistance, p.(*Point).NeiboursCount, p.(*Point).Processed)
			outCount++
		}
	}
	fmt.Printf("Index time: %f s\n", indexTime)
	fmt.Printf("Read time: %f s\n", readTime)
	fmt.Printf("Cluster time: %f s\n", clusterTime)

}
func labeler(it *index.HeapItem) string {
	return fmt.Sprintf("%s %v", it.Item.(*Point).Item, it.Item.(*Point).Processed)
}
func (m *Manifold) update(N []interface{}, P interface{}, seeds *index.MinPriorityQueue, epsilon float32, MinPts int) {
	p := P.(*Point)
	for _, oo := range N {
		o := oo.(*Point)
		if o.Processed {
			continue
		}
		newReachDist := max(p.CoreDistance, m.Angular(p, o))
		if o.ReachabilityDistance == UNDEFINED { // o is not in Seeds
			o.ReachabilityDistance = newReachDist
			//fmt.Printf("Add %s to SEEDS l:%d with RD %f\n", o.Item, len(*seeds), o.ReachabilityDistance)
			heap.Push(seeds, &index.HeapItem{o, o.ReachabilityDistance, -1})
			//seeds.Print(labeler)
		} else {
			if newReachDist < o.ReachabilityDistance { // o in Seeds, check for improvement
				//fmt.Printf("Update RD of %s %v from %f to %f in l:%d \n", o.Item, o.Processed, o.ReachabilityDistance, newReachDist, len(*seeds))
				o.ReachabilityDistance = newReachDist
				/// find item position in pqueue
				///var i int
				///var hi *index.HeapItem
				///var ok bool
				//seeds.Print(labeler)
				//for i, hi := range *seeds {
				//	fmt.Printf("\t%d %d %s %f %f\n", i, hi.Index, hi.Item.(*Point).Item, hi.Item.(*Point).ReachabilityDistance, hi.Dist)
				//	if hi.Item.(*Point).Item == o.Item {
				//		hi.Dist = o.ReachabilityDistance
				//	}
				//}
				hi := seeds.Find(o)
				hi.Dist = o.ReachabilityDistance
				heap.Fix(seeds, hi.Index)

				//fmt.Printf("\n")
				//seeds.Print(labeler)
				//for i, hi := range *seeds {
				//	fmt.Printf("\t%d %d %s %f %f\n", i, hi.Index, hi.Item.(*Point).Item, hi.Item.(*Point).ReachabilityDistance, hi.Dist)
				//	if hi.Item.(*Point).Item == o.Item {
				//		hi.Dist = o.ReachabilityDistance
				//	}
				//}
			}
		}

	}
}
