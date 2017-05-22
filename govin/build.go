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
	f, err := os.Create(output)
	if err != nil {
		log.Printf("Problem opening file %s", output)
		log.Fatal(err)
	}

	// write total
	// write words
	// write wgrams
	// write ngrams
	var i uint32
	for ; i < 5; i++ {
		err := binary.Write(f, binary.LittleEndian, i)
		if err != nil {
			log.Println(err)
			panic(err)
		}
	}

	defer f.Close()

	index := make([]string, 0)
	var w sync.WaitGroup
	lines := make(chan string, threads)
	// read input file thread
	go func(chan string) {
		e = LinesFromFile(input, lines)
	}(lines)

	// vector maker thread
	w.Add(1)
	go func(chan string) {
		var count uint32
		var wordCount, nGramCount, wGramCount uint32

		write := func(key string, vector [128]float32) {
			index = append(index, key)
			err := binary.Write(f, binary.LittleEndian, vector)
			if err != nil {
				log.Println(err)
				panic(err)
			}

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

			//buf := new(bytes.Buffer)
			//err := binary.Write(buf, binary.LittleEndian, vector)
			//if err != nil {
			//	log.Println(err)
			//	panic(err)
			//}
			switch lineParts[0] {
			case "@FWoRd":
				write("0"+lineParts[1], vector)
				wordCount++
			case "@WoRd":
				write("1"+lineParts[1], vector)
				wGramCount++
			case "@NgRaM":
				write("2"+lineParts[1], vector)
				nGramCount++
			default:
				panic(fmt.Errorf("Wrong format %s", lineParts[0]))
			}

			count++
			if count%100000 == 0 {
				log.Printf("Found %d vectors %d words %d ngrams %d wgrams\n", count, wordCount, nGramCount, wGramCount)
			}
		}
		// write words
		for _, w := range index {
			f.WriteString(w)
			f.WriteString("\n")
		}
		// write index totols
		f.Seek(0, 0)
		err = binary.Write(f, binary.LittleEndian, count)
		if err != nil {
			log.Fatalln(err)
		}

		err = binary.Write(f, binary.LittleEndian, wordCount)
		if err != nil {
			log.Fatalln(err)
		}

		err = binary.Write(f, binary.LittleEndian, wGramCount)
		if err != nil {
			log.Fatalln(err)
		}

		err = binary.Write(f, binary.LittleEndian, nGramCount)
		if err != nil {
			log.Fatalln(err)
		}

		/*
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
		*/
		log.Printf("Found %d vectors total\n", count)
		log.Printf("Found %d words total\n", wordCount)
		log.Printf("Found %d n-grams total\n", nGramCount)
		log.Printf("Found %d wn-grams total\n", wGramCount)

		w.Done()
	}(lines)
	w.Wait()
	return
}
