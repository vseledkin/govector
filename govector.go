package govector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"runtime/debug"
	"time"

	"github.com/boltdb/bolt"
	"github.com/vseledkin/go-cache"
	"github.com/vseledkin/govector/index"
)

const (
	WORD_COUNT_KEY  = "3_WordCount_"
	NGRAM_COUNT_KEY = "3_NGramCount_"
	WGRAM_COUNT_KEY = "3_WGramCount_"

	WORDS_BUCKET = "WORDS_BUCKET"
)

type Manifold struct {
	dbfile string
	bc     *bolt.DB
	cache  *cache.Cache
	//c := cache.New(5*time.Minute, 30*time.Second)
}

func NewManifold(dbfile string) (*Manifold, error) {
	_, err := os.Stat(dbfile)
	if err == nil {
		manifold := new(Manifold)
		manifold.dbfile = dbfile
		manifold.cache = cache.New(time.Second, 30*time.Second)
		return manifold, nil
	}
	return nil, err
}

func (m *Manifold) Open() (err error) {
	m.bc, err = bolt.Open(m.dbfile, 0600, &bolt.Options{ReadOnly: true, Timeout: 10 * time.Second})
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

func (m *Manifold) llget(key []byte) (v []byte, err error) {
	err = m.bc.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WORDS_BUCKET))
		if b == nil {
			return fmt.Errorf("bucket %s not found", WORDS_BUCKET)
		}
		v = b.Get(key)
		return nil
	})
	return
}

func (m *Manifold) GetVector(s string) (v []float32, e error) {
	println("GET", s)
	vv, found := m.cache.Get(s)
	if found {
		CacheHit++
		//log.Printf("Hit %s %d", s, m.cache.ItemCount())
		return vv.([]float32), nil
	}
	//log.Printf("Miss %s %d", s, m.cache.ItemCount())
	var byteval []byte
	if byteval, e = m.llget([]byte("0" + s)); e == nil {
		v, e = m.decodeVector(byteval)
		if e == nil {
			// got precomputed vector

		}
		if e != io.EOF {
			log.Printf("Cannot 0 get word vector [%s] %s", string(byteval), e)
			return
		}
	} else {
		// no precomputed vector found
		// get wGram
		if byteval, e = m.llget([]byte("1" + s)); e == nil {
			v, e = m.decodeVector(byteval)

			if e != nil && e != io.EOF {
				log.Printf("Cannot 1 get word vector [%s] %s", string(byteval), e)
				return
			}
			if e == io.EOF {
				e = nil
			}
			log.Printf("VGram: %s %#v", s, v[:6])
			// got vGram vector vector
		}
		// get ngrams
		for _, ngram := range m.ComputeNGrams("<" + s + ">") {
			if byteval, e = m.llget([]byte("2" + ngram)); e == nil {
				var nv []float32
				if nv, e = m.decodeVector(byteval); e != nil && e != io.EOF {
					log.Printf("Cannot 2 get word vector %s %s", string(byteval), e)
					return
				} else {
					if e == io.EOF {
						e = nil
						// not found
					} else {
						log.Printf("NGram: %s %#v", ngram, nv[:6])
						// we got ngram vector
						if len(v) > 0 {
							Sxpy(nv, v)
						} else {
							v = nv
						}
					}
				}
			}
		}

		// normalize
		if len(v) > 0 {
			Sscale(1/L2(v), v)
		}

		log.Printf("Word: %s %#v", s, v)
		e = nil
	}

	//log.Printf("Vector: %s %#v", s, v)
	CacheMiss++
	m.cache.Add(s, v, time.Second)
	return
}

func (m *Manifold) decodeVector(vb []byte) (v []float32, e error) {
	var vector [128]float32
	buf := bytes.NewReader(vb)

	if e = binary.Read(buf, binary.LittleEndian, &vector); e != nil {
		return
	}
	v = vector[:]
	return
}

func (m *Manifold) Dim() int {
	return 128
}

func (m *Manifold) HasWord(s string) bool {
	v, _ := m.llget([]byte("0" + s))
	return v != nil
}

func (m *Manifold) HasWGram(s string) bool {
	v, _ := m.llget([]byte("1" + s))
	return v != nil
}

func (m *Manifold) HasNGram(s string) bool {
	v, _ := m.llget([]byte("2" + s))
	return v != nil
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

func (m *Manifold) VisitWords(visitor func(key string, vector []float32)) {
	m.bc.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WORDS_BUCKET))
		b.ForEach(func(k, v []byte) error {
			if k[0] == '0' {
				if vector, e := m.decodeVector(v); e == nil {
					visitor(string(k[1:]), vector)
				}
			}
			return nil
		})
		return nil
	})
}

func (m *Manifold) VisitNGrams(visitor func(key string, vector []float32)) {
	m.bc.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WORDS_BUCKET))
		b.ForEach(func(k, v []byte) error {
			if k[0] == '2' {
				if vector, e := m.decodeVector(v); e == nil {
					visitor(string(k[1:]), vector)
				}
			}
			return nil
		})
		return nil
	})
}

func (m *Manifold) VisitWGrams(visitor func(key string, vector []float32)) {
	m.bc.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(WORDS_BUCKET))
		b.ForEach(func(k, v []byte) error {
			if k[0] == '1' {
				if vector, e := m.decodeVector(v); e == nil {
					visitor(string(k[1:]), vector)
				}
			}
			return nil
		})
		return nil
	})
}

func (m *Manifold) Count() (count uint32) {
	return m.WordCount() + m.NGramCount()

}

func (m *Manifold) WordCount() (count uint32) {
	b, e := m.llget([]byte(WORD_COUNT_KEY))
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
	b, e := m.llget([]byte(NGRAM_COUNT_KEY))
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

func (m *Manifold) WGramCount() (count uint32) {
	b, e := m.llget([]byte(WGRAM_COUNT_KEY))
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
	m.VisitWords(func(key string, _ []float32) {
		keys[i] = key
		i++
	})
	idx := index.NewVPTree(m.AngularStr, keys)
	m.cache.DeleteExpired()
	debug.FreeOSMemory()
	return idx
}
