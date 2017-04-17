package main

import (
	"bufio"
	"log"
	"os"
	"strings"
	"sync"
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
	var w sync.WaitGroup
	lines := make(chan string, threads)
	// read input file thread
	go func(chan string) {
		e = LinesFromFile(input, lines)
		close(lines)
	}(lines)

	// vector maker thread
	w.Add(1)
	go func(chan string) {
		for line := range lines {
			log.Printf("%s", line)
		}
		w.Done()
	}(lines)
	w.Wait()
	return
}
