package govector

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"

	"os"

	"github.com/vseledkin/govector/mmap"
)

type Store struct {
	vectors    *mmap.ReaderAt
	index      map[string]uint32
	Reader     *bytes.Reader
	WordCount  uint32
	NGramCount uint32
	WGramCount uint32
	TotalCount uint32
}

func (s *Store) Open(name string) (e error) {
	s.vectors, e = mmap.Open(name)
	s.index = make(map[string]uint32)
	// read total number of vectors

	//reader := io.NewSectionReader(s.vectors, 0, 4*4)
	s.Reader = bytes.NewReader(s.vectors.Data)
	e = binary.Read(s.Reader, binary.LittleEndian, &s.TotalCount)
	if e != nil {
		log.Println(e)
	}
	println(s.TotalCount)

	e = binary.Read(s.Reader, binary.LittleEndian, &s.WordCount)
	if e != nil {
		log.Println(e)
	}
	println(s.WordCount)

	e = binary.Read(s.Reader, binary.LittleEndian, &s.WGramCount)
	if e != nil {
		log.Println(e)
	}
	println(s.WGramCount)

	e = binary.Read(s.Reader, binary.LittleEndian, &s.NGramCount)
	if e != nil {
		log.Println(e)
	}
	println(s.NGramCount)
	var offset int64
	offset, e = s.Reader.Seek(4*4+int64(s.TotalCount)*128*4+4, 0)
	println("Offset:", offset)

	//	reader = io.NewSectionReader(s.vectors, 4*4+int64(s.TotalCount)*128*4, int64(s.vectors.Len()))
	var count uint32
	var buff []byte
	for {
		b, e := s.Reader.ReadByte()
		if e != nil {
			break
		}
		if b != '\n' {
			buff = append(buff, b)
		} else {
			if count%100000 == 0 {
				fmt.Printf("[%s] %d\n", string(buff), count)
			}
			s.index[string(buff)] = count
			count++
			buff = buff[:0]
			//if count == 10 {
			//	break
			//}
		}
	}
	fmt.Printf("%d\n", count)

	return
}

func (s *Store) Close() (e error) {
	e = s.vectors.Close()
	return
}

type FStore struct {
	vectors    *os.File
	index      map[string]uint32
	Reader     *bytes.Reader
	WordCount  uint32
	NGramCount uint32
	WGramCount uint32
	TotalCount uint32
}

func (s *FStore) Open(name string) (e error) {
	s.vectors, e = os.Open(name)
	s.index = make(map[string]uint32)
	// read total number of vectors

	//reader := io.NewSectionReader(s.vectors, 0, 4*4)

	//s.Reader = bytes.NewReader(s.vectors)
	e = binary.Read(s.vectors, binary.LittleEndian, &s.TotalCount)
	if e != nil {
		log.Println(e)
	}
	println(s.TotalCount)

	e = binary.Read(s.vectors, binary.LittleEndian, &s.WordCount)
	if e != nil {
		log.Println(e)
	}
	println(s.WordCount)

	e = binary.Read(s.vectors, binary.LittleEndian, &s.WGramCount)
	if e != nil {
		log.Println(e)
	}
	println(s.WGramCount)

	e = binary.Read(s.vectors, binary.LittleEndian, &s.NGramCount)
	if e != nil {
		log.Println(e)
	}
	println(s.NGramCount)
	var offset int64
	offset, e = s.vectors.Seek(4*4+int64(s.TotalCount)*128*4+4, 0)
	println("Offset:", offset)

	//	reader = io.NewSectionReader(s.vectors, 4*4+int64(s.TotalCount)*128*4, int64(s.vectors.Len()))
	var count uint32
	buff_1 := []byte{' '}
	var buff []byte
	for {
		_, e := s.vectors.Read(buff_1)
		if e != nil {
			break
		}
		if buff_1[0] != '\n' {
			buff = append(buff, buff_1[0])
		} else {
			if count%100000 == 0 {
				fmt.Printf("[%s] %d\n", string(buff), count)
			}
			s.index[string(buff)] = count
			count++
			buff = buff[:0]
			//if count == 10 {
			//	break
			//}
		}
	}
	fmt.Printf("%d\n", count)

	return
}

func (s *FStore) Close() (e error) {
	e = s.vectors.Close()
	return
}
