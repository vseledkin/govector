package govector

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"sort"

	"github.com/vseledkin/govector/annoy"
	"github.com/vseledkin/govector/index"
)

type Manifold struct {
	dbfile string
	bc     *Store
	cache  *Cache
	//c := cache.New(5*time.Minute, 30*time.Second)
}

func NewManifold(dbfile string) (*Manifold, error) {
	_, err := os.Stat(dbfile)
	if err == nil {
		manifold := new(Manifold)
		manifold.dbfile = dbfile
		manifold.cache = NewCache()
		return manifold, nil
	}
	return nil, err
}

func (m *Manifold) Open() (err error) {
	m.bc = new(Store)
	err = m.bc.Open(m.dbfile)
	return
}

func (m *Manifold) Close() {
	m.bc.Close()
}

var CacheHit, CacheMiss int

func сomputeNGrams(s string, minn, maxn int) (ngrams []string) {

	runes := []rune(s)
	L := len(runes)
	for i := 0; i < L; i++ {
		ngram := ""
		if (runes[i] & 0xC0) == 0x80 {
			continue
		}
		for j, n := i, 1; j < L && n <= maxn; n++ {
			ngram += string(runes[j])
			j++
			for j < L && (runes[j]&0xC0) == 0x80 {
				ngram += string(runes[j])
				j++
			}
			if n >= minn && !(n == 1 && (i == 0 || j == L)) {
				ngrams = append(ngrams, ngram)
			}
		}
	}
	return
}

func (m *Manifold) ComputeNGrams(s string) (ngrams []string) {
	return сomputeNGrams(s, 3, 6)
}

func (m *Manifold) WordID(word string) int32 {
	offset, ok := m.bc.index["0"+word]
	if ok {
		return int32(offset)
	} else {
		return -1
	}
}

func (m *Manifold) IDWord(id int32) string {
	return m.bc.rindex[id][1:]
}

func (m *Manifold) llget(key []byte) (v []float32, ok bool, err error) {
	offset, ok := m.bc.index[string(key)]
	if !ok {
		return nil, false, nil
	}
	offs := 4*4 + int64(offset)*128*4
	var off int64
	off, err = m.bc.Reader.Seek(offs, 0)
	if err != nil {
		return nil, false, fmt.Errorf("Seek error: want %d got %d ", offs, off)
	}
	var vector [128]float32

	if err = binary.Read(m.bc.Reader, binary.LittleEndian, &vector); err != nil {
		return nil, false, err
	}

	v = vector[:]
	//fmt.Printf("[%s]: [%#v]", string(key), v)
	ok = true
	return
}

func (m *Manifold) GetVector(s string) (v []float32, e error) {
	if len(s) == 0 {
		return []float32{}, fmt.Errorf("Empty word")
	}
	var found bool
	v, found = m.cache.Get(s)
	if found {
		CacheHit++
		//log.Printf("Hit %s %d", s, len(m.cache.cache))
		return v, nil
	}

	if v, found, e = m.llget([]byte("0" + s)); e != nil {
		log.Printf("Error geting vector for word [%s] %s", s, e)
		return
	} else if found {
		CacheMiss++
		//log.Printf("Found in dictionary %s\n%#v\n", s, v)
		m.cache.Set(s, v)
		return
	}
	//log.Printf("Word [%s] not found\n", s)
	// we have not found ready vector so compute it from ngrams
	// get wgram
	if v, found, e = m.llget([]byte("1" + s)); e != nil {
		log.Printf("Error geting vector for word [%s] %s", s, e)
		return
	}

	// get ngrams
	var nv []float32
	for _, ngram := range m.ComputeNGrams("<" + s + ">") {
		if nv, found, e = m.llget([]byte("2" + ngram)); e != nil {
			log.Printf("Error geting vector for word [%s] %s", s, e)
			return
		} else if found {
			if len(v) > 0 {
				Sxpy(nv, v)
			} else {
				v = nv
			}
		}
	}

	if len(v) == 0 {
		// we have no such characters!!!! at all
		var emptyVector [128]float32
		v = emptyVector[:]
		return v, nil
	}
	if len(v) > 0 {
		Sscale(1/L2(v), v)
	}

	CacheMiss++
	m.cache.Set(s, v)
	return
}

func (m *Manifold) Dim() int {
	return 128
}

func (m *Manifold) HasWord(s string) (has bool) {
	_, has = m.bc.index["0"+s]
	return
}

func (m *Manifold) HasWGram(s string) (has bool) {
	_, has = m.bc.index["1"+s]
	return
}

func (m *Manifold) HasNGram(s string) (has bool) {
	_, has = m.bc.index["2"+s]
	return
}

/*
func (m *Manifold) VisitKeys(visitor func(key string)) {
	m.bc.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket([]byte(WORDS_BUCKET))

		b.ForEach(func(k, _ []byte) error {
			visitor(string(k))
			return nil
		})
		return nil
	})
}*/

func (m *Manifold) VisitWordsAndVectors(visitor func(key string, vector []float32)) {
	rindex := make([]struct {
		s string
		o uint32
	}, m.bc.WordCount)
	i := 0
	for k, v := range m.bc.index {
		if k[0] == '0' {
			rindex[i] = struct {
				s string
				o uint32
			}{k[1:], v}
			i++
		}
	}
	sort.Slice(rindex, func(i, j int) bool {
		return rindex[i].o < rindex[j].o
	})
	for _, w := range rindex {
		vector, e := m.GetVector(w.s)
		if e != nil {
			panic(e)
		}
		visitor(w.s, vector)
	}
}

func (m *Manifold) VisitWords(visitor func(key string) bool) {
	count := 0
	for _, k := range m.bc.rindex {
		if k[0] == '0' {
			count++
			if !visitor(string(k[1:])) {
				break
			}
			if count == int(m.bc.WordCount) {
				break
			}
		}

	}
}

func (m *Manifold) VisitNGrams(visitor func(key string, vector []float32)) {
	rindex := make([]string, 0)
	for k, v := range m.bc.index {
		if k[0] == '2' {
			rindex[v] = k[1:]
		}
	}
	for _, w := range rindex {
		vector, e := m.GetVector(w)
		if e != nil {
			panic(e)
		}
		visitor(w, vector)
	}
}

func (m *Manifold) VisitWGrams(visitor func(key string, vector []float32)) {
	rindex := make([]string, 0)
	for k, v := range m.bc.index {
		if k[0] == '1' {
			rindex[v] = k[1:]
		}
	}
	for _, w := range rindex {
		vector, e := m.GetVector(w)
		if e != nil {
			panic(e)
		}
		visitor(w, vector)
	}
}

func (m *Manifold) Count() (count uint32) {
	return m.bc.TotalCount
}

func (m *Manifold) WordCount() uint32 {
	return m.bc.WordCount
}

func (m *Manifold) NGramCount() uint32 {
	return m.bc.NGramCount
}

func (m *Manifold) WGramCount() uint32 {
	return m.bc.WGramCount
}

//Angular - cosine distance in the case of vector components are positive or negative
func (m *Manifold) Angular(x, y interface{}) (d float32) {
	var e error
	switch t := x.(type) {
	case []float32:
		d = Sdot(t, y.([]float32))
		if d > 1 {
			log.Printf("Dot of normalized vector > 1 %f", d)
			return 0
		}
		if d < -1 {
			log.Printf("Dot of normalized vector < -1 %f", d)
			return 1
		}
		d = float32(math.Acos(float64(d)) / math.Pi)
		return
	case *Point:
		var xv, yv []float32
		if t.Vector == nil {
			xv, e = m.GetVector(t.Item)
			if e != nil {
				panic(e)
			}
			t.Vector = xv
		} else {
			xv = t.Vector
		}

		if y.(*Point).Vector == nil {
			yv, e = m.GetVector(y.(*Point).Item)
			if e != nil {
				panic(e)
			}
			y.(*Point).Vector = yv
		} else {
			yv = y.(*Point).Vector
		}
		d = Sdot(xv, yv)
		if d > 1 {
			log.Printf("Dot of normalized vector > 1 %f", d)
			return 0
		}
		if d < -1 {
			log.Printf("Dot of normalized vector < -1 %f", d)
			return 1
		}
		d = float32(math.Acos(float64(d)) / math.Pi)
		return
	case string:
		xv, e := m.GetVector(t)
		if e != nil {
			panic(e)
		}
		yv, e := m.GetVector(y.(string))
		if e != nil {
			panic(e)
		}
		d = Sdot(xv, yv)
		if d > 1 {
			log.Printf("Dot of normalized vector > 1 %f", d)
			return 0
		}
		if d < -1 {
			log.Printf("Dot of normalized vector < -1 %f", d)
			return 1
		}
		d = float32(math.Acos(float64(d)) / math.Pi)
		return
	default:
		panic(fmt.Errorf("Wrong data type"))
	}
}

/*
//Euclidean
func (m *Manifold) Euclidean(x, y []float32) (d float32) {
	//cosine := float64(Sdot(x, y) / L2(x) / L2(y))
	d = Sdot(x, y)
	if d > 1 {
		log.Printf("Dot of normalized vector > 1", d)
		d = 1
	}
	if d < -1 {
		log.Printf("Dot of normalized vector < -1", d)
		d = -1
	}
	d = float32(math.Sqrt(float64(2.0 * (1.0 - d))))
	return
}


func (m *Manifold) EuclideanStr(x, y string) (d float32) {
	xv, e := m.GetVector(x)
	if e != nil {
		panic(e)
	}
	yv, e := m.GetVector(y)
	if e != nil {
		panic(e)
	}
	d = m.Euclidean(xv, yv)

	return
}
*/
func (m *Manifold) AnnoyIndex() (annoy.AnnoyIndexAngular, []interface{}) {
	/*defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			fmt.Println("Failed to create index")

		}
	}()*/
	var max uint32 = m.WordCount()
	if m.WordCount() < max {
		max = m.WordCount()
	}
	keys := make([]interface{}, max)
	log.Printf("Reading %d words", max)
	var i uint32 = 0
	m.VisitWords(func(key string) bool {
		if len(key) == 0 {
			panic(fmt.Errorf("Empty key"))
		}
		keys[i] = key
		i++
		if i == max {
			return false
		}
		if i%1e4 == 0 {
			log.Printf("Read %d words", i)
		}
		return true
	})
	// check triangular inequality
	/*
		for i := 0; i < len(keys)-3; i++ {
			x := keys[i]
			y := keys[i+1]
			z := keys[i+2]
			dxy := m.Angular(x, y)
			dxz := m.Angular(x, z)
			dyz := m.Angular(y, z)

			if dxy > dxz+dyz {
				panic("Distance function is not metric!")
			}

			if dxz > dxy+dyz {
				panic("Distance function is not metric!")
			}
			if dyz > dxy+dxz {
				panic("Distance function is not metric!")
			}
		}*/

	log.Printf("Read %d words", i)

	idx := annoy.NewAnnoyIndexAngular(m.Dim())
	for i, key := range keys {
		v, e := m.GetVector(key.(string))
		if e != nil {
			panic(e)
		}
		idx.AddItem(i, v)
	}
	idx.Build(16)
	//idx.Save(m.dbfile + ".tree")
	return idx, keys
}

func (m *Manifold) MakeVPIndex() *index.VPTree {
	/*defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovered in f", r)
			fmt.Println("Failed to create index")

		}
	}()*/
	//var max uint32 = 1000
	var max uint32 = m.WordCount()
	if m.WordCount() < max {
		max = m.WordCount()
	}
	keys := make([]interface{}, max)
	log.Printf("Reading %d words", max)
	var i uint32 = 0
	m.VisitWords(func(key string) bool {
		if len(key) == 0 {
			panic(fmt.Errorf("Empty key"))
		}
		keys[i] = key
		i++
		if i == max {
			return false
		}
		if i%1e4 == 0 {
			log.Printf("Read %d words", i)
		}
		return true
	})
	// check triangular inequality
	/*
		for i := 0; i < len(keys)-3; i++ {
			x := keys[i]
			y := keys[i+1]
			z := keys[i+2]
			dxy := m.Angular(x, y)
			dxz := m.Angular(x, z)
			dyz := m.Angular(y, z)

			if dxy > dxz+dyz {
				panic("Distance function is not metric!")
			}

			if dxz > dxy+dyz {
				panic("Distance function is not metric!")
			}
			if dyz > dxy+dxz {
				panic("Distance function is not metric!")
			}
		}
	*/
	log.Printf("Read %d words", i)
	idx := index.NewVPTree(m.Angular, keys)

	//idx.PrintTree(nil, 0, 100)
	return idx
}
