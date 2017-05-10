package govector

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"

	"math"

	"time"

	"runtime/debug"

	"github.com/vseledkin/bitcask"
	"github.com/vseledkin/go-cache"
	"github.com/vseledkin/govector/index"
)

type Manifold struct {
	dir   string
	bc    *bitcask.BitCask
	cache *cache.Cache
	//c := cache.New(5*time.Minute, 30*time.Second)
}

func NewManifold(dir string) (*Manifold, error) {
	_, err := os.Stat(dir)
	if err == nil {
		manifold := new(Manifold)
		manifold.dir = dir
		manifold.cache = cache.New(time.Second, 30*time.Second)
		return manifold, nil
	}
	return nil, err
}

func (m *Manifold) Open() (err error) {
	m.bc, err = bitcask.Open(m.dir, nil)
	return
}

func (m *Manifold) Close() {
	m.bc.Close()
}

var CacheHit, CacheMiss int

func (m *Manifold) GetVector(s string) (v []float32, e error) {
	vv, found := m.cache.Get(s)
	if found {
		CacheHit++
		//log.Printf("Hit %s %d", s, m.cache.ItemCount())
		return vv.([]float32), nil
	}
	//log.Printf("Miss %s %d", s, m.cache.ItemCount())
	byteval, e := m.bc.Get([]byte(s))
	if e != nil {
		log.Printf("String %s not found in dictionary: %s", s, e)
	}
	v, e = m.getVector(byteval)
	if e != nil {
		log.Printf("String %s not found in dictionary: %s", s, e)
	}
	CacheMiss++
	m.cache.Add(s, v, time.Second)
	return
}

func (m *Manifold) getVector(s []byte) (v []float32, e error) {
	var vector [128]float32
	buf := bytes.NewReader(s)
	e = binary.Read(buf, binary.LittleEndian, &vector)
	if e != nil {
		log.Println("Cannot deserialize to vector %s", e)
	}
	v = vector[:]
	return
}

func (m *Manifold) Dim() int {
	return 128
}

func (m *Manifold) HasWord(s string) bool {
	return m.bc.HasKey(s)
}

func (m *Manifold) VisitKeys(visitor func(key string)) {
	m.bc.VisitKeys(func(bcKey []byte) {
		visitor(string(bcKey))
	})
}

func (m *Manifold) VisitFast(visitor func(key string, vector []float32)) {
	m.bc.VisitKeysAndValues(func(bcKey, bcValue []byte) {
		v, e := m.getVector(bcValue)
		if e == nil {
			visitor(string(bcKey), v)
		}
	})
}

func (m *Manifold) Count() int {
	return m.bc.Count()
}

//Angular - cosine distance
func (m *Manifold) Angular(x, y []float32) float32 {
	//cosine := float64(Sdot(x, y) / L2(x) / L2(y))
	return float32(2.0 * math.Acos(float64(Sdot(x, y))) / math.Pi)
}

func (m *Manifold) AngularStr(x, y string) float32 {
	xv, e := m.GetVector(x)
	if e != nil {
		return 1
	}
	yv, e := m.GetVector(y)
	if e != nil {
		return 1
	}
	return m.Angular(xv, yv)
}

func (m *Manifold) MakeVPIndex() *index.VPTree {
	keys := make([]string, m.Count())
	i := 0
	m.VisitKeys(func(key string) {
		keys[i] = key
		i++
	})
	idx := index.NewVPTree(m.AngularStr, keys)
	m.cache.DeleteExpired()
	debug.FreeOSMemory()
	return idx
}
