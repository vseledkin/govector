package govector

import (
	"fmt"
	"log"

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
}

/*
ComputeClusters
	epsilon - the maximum distance (radius) to consider
	MinPts  - the number of points required to form a cluster.
*/
func (m *Manifold) ComputeClusters(epsilon float32, MinPts int) *index.VPTree {
	/*defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			fmt.Println("Failed to create index")

		}
	}()*/
	//var max uint32 = 1000
	var max uint32 = 100
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
		points[i] = &Point{key, UNDEFINED, UNDEFINED, false}
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
	idx := index.NewVPTree(m.Angular, points)
	// Optics
	for _, p := range points {
		N, distances := idx.Search(p, 10*MinPts, epsilon)
		p.(*Point).Processed = true
		if len(N) >= MinPts {
			p.(*Point).CoreDistance = distances[0]
		}
		if p.(*Point).CoreDistance != UNDEFINED {
			var seeds index.PriorityQueue
			m.update(N, p, seeds, epsilon, MinPts)
			for _, qhi := range seeds {
				q := qhi.Item.(*Point)
				NQ, nqd := idx.Search(q, 10*MinPts, epsilon)
				q.Processed = true
				if len(NQ) >= MinPts {
					q.CoreDistance = nqd[0]
				}
				if q.CoreDistance != UNDEFINED {
					m.update(NQ, q, seeds, epsilon, MinPts)
				}
			}
		}

		/*
			if len(nearestPoints) > MinPts {
				fmt.Printf("%d-%s %f\n", j, p.(*Point).Item, p.(*Point).CoreDistance)
				for i, np := range nearestPoints {
					fmt.Printf("\t%d %s %f\n", i, np.(*Point).Item, distances[i])
				}
			}*/
	}
	for j, p := range points {
		fmt.Printf("%d-%s %f %v\n", j, p.(*Point).Item, p.(*Point).CoreDistance, p.(*Point).Processed)
	}
	return idx
}

func (m *Manifold) update(N []interface{}, P interface{}, seeds index.PriorityQueue, epsilon float32, MinPts int) {
	p := P.(*Point)
	for _, oo := range N {
		o := oo.(*Point)
		if !o.Processed {
			newRreachDist := max(p.CoreDistance, m.Angular(p, o))
			if o.ReachabilityDistance == UNDEFINED { // o is not in Seeds
				o.ReachabilityDistance = newRreachDist
				seeds.Push(&index.HeapItem{o, o.ReachabilityDistance})
			} else {
				if newRreachDist < o.ReachabilityDistance { // o in Seeds, check for improvement
					o.ReachabilityDistance = newRreachDist
					seeds.FixItem(o)
				}
			}
		}
	}
}
