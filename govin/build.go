package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/boltdb/bolt"
	"github.com/vseledkin/bitcask"
	"github.com/vseledkin/govector"
)

/*
LinesFromFile read file line by line skipping empty ones
*/
func LinesFromFile(f string, ch chan string) error {
	in, e := os.Open(f)
	if e != nil {
		return e
	}
	LinesFrom(bufio.NewReader(in), ch)
	return nil
}

/*
LinesFromStdin read file line by line skipping empty ones
*/
func LinesFromStdin(ch chan string) {
	LinesFrom(bufio.NewReader(os.Stdin), ch)
}

/*
LinesFromStdin read file line by line skipping empty ones
*/
func LinesFrom(reader *bufio.Reader, ch chan string) {
	for {
		line, e := reader.ReadString('\n')
		if e != nil {
			break
		}
		line = strings.TrimSpace(line)
		if len(line) > 0 {
			ch <- line
		}
	}
	close(ch)
}

func BuildText() (e error) {
	bc, err := bitcask.Open(output, nil)
	if err != nil {
		log.Printf("Problem opening output directory %s", output)
		log.Fatal(err)
	}
	defer bc.Close()

	var w sync.WaitGroup
	lines := make(chan string, threads)
	// read input file thread
	go func(chan string) {
		e = LinesFromFile(input, lines)
	}(lines)

	// vector maker thread
	w.Add(1)
	go func(chan string) {
		count := 0
		for line := range lines {

			lineParts := strings.Fields(line)
			var vector [128]float32
			for i, strVal := range lineParts[1:] {
				v, err := strconv.ParseFloat(strVal, 32)
				if err != nil {
					log.Println(strVal)
				}
				vector[i] = float32(v)
			}
			// normalize vector

			govector.Sscale(1/govector.L2(vector[:]), vector[:])
			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.LittleEndian, vector)
			if err != nil {
				log.Println(err)
			}
			bc.Put([]byte(lineParts[0]), buf.Bytes())
			count++
			if count%10000 == 0 {
				log.Printf("Found %d vectors\n", count)
			}
		}
		log.Printf("Found %d vectors total\n", count)
		w.Done()
	}(lines)
	w.Wait()
	return
}

func BuildFastText() (e error) {

	db, err := bolt.Open(output, 0600, &bolt.Options{Timeout: 1 * time.Second})

	if err != nil {
		log.Printf("Problem opening db %s", output)
		log.Fatal(err)
	}
	defer db.Close()

	var w sync.WaitGroup
	lines := make(chan string, threads)
	// read input file thread
	go func(chan string) {
		e = LinesFromFile(input, lines)
	}(lines)

	// vector maker thread
	w.Add(1)
	go func(chan string) {
		count := 0
		var wordCount, nGramCount, wGramCount uint32

		type value struct {
			k []byte
			v []byte
		}
		var values []value
		write := func() {

			err = db.Update(func(tx *bolt.Tx) error {
				b, e := tx.CreateBucketIfNotExists([]byte(govector.WORDS_BUCKET))
				if e != nil {
					return fmt.Errorf("create bucket error: %s", e)
				}
				for _, v := range values {
					if e = b.Put(v.k, v.v); e != nil {
						return e
					}
				}
				return nil // commit transaction
			})
			if err != nil {
				log.Panicln("error", err)
				panic(err)
			}
			values = values[:0]

		}
		for line := range lines {

			lineParts := strings.Fields(line)
			var vector [128]float32

			for i, strVal := range lineParts[2:] {
				v, err := strconv.ParseFloat(strVal, 32)
				if err != nil {
					log.Println(err, strVal)
					panic(err)
				}
				vector[i] = float32(v)
			}
			// normalize vector

			//govector.Sscale(1/govector.L2(vector[:]), vector[:])
			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.LittleEndian, vector)
			if err != nil {
				log.Println(err)
				panic(err)
			}
			switch lineParts[0] {
			case "@FWoRd":
				values = append(values, value{[]byte("0" + lineParts[1]), buf.Bytes()})
				wordCount++
			case "@WoRd":
				values = append(values, value{[]byte("1" + lineParts[1]), buf.Bytes()})
				wGramCount++
			case "@NgRaM":
				values = append(values, value{[]byte("2" + lineParts[1]), buf.Bytes()})
				nGramCount++
			default:
				panic(fmt.Errorf("Wrong format %s", lineParts[0]))
			}

			if len(values) == 100000 {
				write()
			}
			count++
			if count%10000 == 0 {
				log.Printf("Found %d vectors %d words %d ngrams %d wgrams\n", count, wordCount, nGramCount, wGramCount)
			}
		}
		if len(values) > 0 {
			write()
		}
		// write wordcount
		err = db.Update(func(tx *bolt.Tx) error {
			b, e := tx.CreateBucketIfNotExists([]byte(govector.WORDS_BUCKET))

			if e != nil {
				return fmt.Errorf("create bucket error: %s", e)
			}
			buf := new(bytes.Buffer)
			err := binary.Write(buf, binary.LittleEndian, wordCount)
			if err != nil {
				log.Fatal(err)
				return err
			}
			if err = b.Put([]byte(govector.WORD_COUNT_KEY), buf.Bytes()); err != nil {
				log.Fatal(err)
				return err
			}

			// write ngramcount
			buf = new(bytes.Buffer)
			err = binary.Write(buf, binary.LittleEndian, nGramCount)
			if err != nil {
				log.Fatal(err)
				return err
			}
			if err = b.Put([]byte(govector.NGRAM_COUNT_KEY), buf.Bytes()); err != nil {
				log.Fatal(err)
				return err
			}

			buf = new(bytes.Buffer)
			err = binary.Write(buf, binary.LittleEndian, wGramCount)
			if err != nil {
				log.Fatal(err)
				return err
			}
			if err = b.Put([]byte(govector.WGRAM_COUNT_KEY), buf.Bytes()); err != nil {
				log.Fatal(err)
				return err
			}

			return nil // commit transaction
		})
		if err != nil {
			log.Fatal(err)
		}

		log.Printf("Found %d vectors total\n", count)
		log.Printf("Found %d words total\n", wordCount)
		log.Printf("Found %d n-grams total\n", nGramCount)
		log.Printf("Found %d wn-grams total\n", wGramCount)

		w.Done()
	}(lines)
	w.Wait()
	return
}
