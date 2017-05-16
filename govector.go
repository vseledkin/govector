package govector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"os"
	"runtime/debug"
	"time"

	"github.com/vseledkin/bitcask"
	"github.com/vseledkin/go-cache"
	"github.com/vseledkin/govector/index"
)

const (
	WORD_COUNT_KEY  = "3_WordCount_"
	NGRAM_COUNT_KEY = "3_NGramCount_"
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

func (m *Manifold) ComputeNGrams(s string) (ngrams []string) {
	minn, maxn := 3, 6
	runes := []rune(s)
	for i := 0; i < len(runes); i++ {
		ngram := ""

		for j, n := i, 1; j < len(runes) && n <= maxn; n++ {
			ngram += string(runes[j])
			j++
			for j < len(runes) {
				ngram += string(runes[j])
				j++
			}
			if n >= minn && !(n == 1 && (i == 0 || j == len(runes))) {
				ngrams = append(ngrams, ngram)
			}
		}
	}
	return
}

func (m *Manifold) GetVector(s string) (v []float32, e error) {
	vv, found := m.cache.Get(s)
	if found {
		CacheHit++
		//log.Printf("Hit %s %d", s, m.cache.ItemCount())
		return vv.([]float32), nil
	}
	//log.Printf("Miss %s %d", s, m.cache.ItemCount())
	byteval, e := m.bc.Get([]byte("0" + s))

	if e == nil {
		//log.Printf("String %s not found in dictionary: %s", s, e)
		v, e = m.getVector(byteval)
		if e != nil {
			log.Printf("Cannot get word vector %s %s", s, e)
		}
	}

	ngrams := m.ComputeNGrams("<" + s + ">")
	fmt.Printf("%#v", ngrams)
	CacheMiss++
	m.cache.Add(s, v, time.Second)
	return
}

func (m *Manifold) getVector(s []byte) (v []float32, e error) {
	var vector [128]float32
	buf := bytes.NewReader(s)
	e = binary.Read(buf, binary.LittleEndian, &vector)
	if e != nil {
		log.Println("Cannot deserialize to vector value with key", string(s), e)
	}
	v = vector[:]
	return
}

func (m *Manifold) Dim() int {
	return 128
}

func (m *Manifold) HasWord(s string) bool {
	return m.bc.HasKey("0" + s)
}

func (m *Manifold) HasNGram(s string) bool {
	return m.bc.HasKey("1" + s)
}

func (m *Manifold) VisitKeys(visitor func(key string)) {
	m.bc.VisitKeys(func(bcKey []byte) {
		visitor(string(bcKey))
	})
}

func (m *Manifold) VisitWords(visitor func(key string, vector []float32)) {
	m.bc.VisitKeysAndValues(func(bcKey, bcValue []byte) {
		if bcKey[0] == '0' {
			v, e := m.getVector(bcValue)
			if e == nil {
				visitor(string(bcKey[1:]), v)
			}
		}
	})
}

func (m *Manifold) VisitNGrams(visitor func(key string, vector []float32)) {
	m.bc.VisitKeysAndValues(func(bcKey, bcValue []byte) {
		if bcKey[0] == '1' {
			v, e := m.getVector(bcValue)
			if e == nil {
				visitor(string(bcKey[1:]), v)
			}
		}
	})
}

func (m *Manifold) Count() int {
	return m.bc.Count()
}

func (m *Manifold) WordCount() (count uint32) {
	b, e := m.bc.Get([]byte(WORD_COUNT_KEY))
	if e != nil {
		panic(e)
	}
	buf := bytes.NewReader(b)
	e = binary.Read(buf, binary.LittleEndian, &count)
	if e != nil {
		panic(e)
	}
	return count
}

func (m *Manifold) NGramCount() (count uint32) {
	b, e := m.bc.Get([]byte(NGRAM_COUNT_KEY))
	if e != nil {
		panic(e)
	}
	buf := bytes.NewReader(b)
	e = binary.Read(buf, binary.LittleEndian, &count)
	if e != nil {
		panic(e)
	}
	return count
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
