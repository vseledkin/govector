package govector

import (
	"bytes"
	"encoding/binary"
	"log"
	"os"

	"github.com/vseledkin/bitcask"
)

type Manifold struct {
	dir string
	bc  *bitcask.BitCask
}

func NewManifold(dir string) (*Manifold, error) {
	_, err := os.Stat(dir)
	if err == nil {
		manifold := new(Manifold)
		manifold.dir = dir
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

func (m *Manifold) GetVector(s string) (v []float32, e error) {
	byteval, e := m.bc.Get([]byte(s))
	if e != nil {
		log.Printf("String %s not found in dictionary: %s", s, e)
	}
	var vector [128]float32
	buf := bytes.NewReader(byteval)
	e = binary.Read(buf, binary.LittleEndian, &vector)
	if e != nil {
		log.Println("String %s not found in dictionary: %s", s, e)
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

func (m *Manifold) Visit(visitor func(key string)) {
	m.bc.Visit(visitor)
}
