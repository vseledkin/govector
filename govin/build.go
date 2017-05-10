package main

import (
	"bufio"
	"bytes"
	"encoding/binary"
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
