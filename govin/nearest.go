package main

import (
	"log"

	"time"

	"bufio"
	"fmt"

	"os"

	"os/signal"
	"syscall"

	"github.com/vseledkin/govector"
	"github.com/vseledkin/govector/index"
)

func readline(fi *bufio.Reader) (string, bool) {
	s, err := fi.ReadString('\n')
	if err != nil {
		return "", false
	}
	return s[:len(s)-1], true
}

func Nearest() (e error) {
	var manifold *govector.Manifold
	manifold, e = govector.NewManifold(input)
	if e != nil {
		log.Printf("Error %s", e)
		return
	}
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Defer Panic:", r)
		}
		manifold.Close()
		log.Println("Closing manifold")
	}()
	e = manifold.Open()
	defer func() {
		manifold.Close()
	}()
	if e != nil {
		log.Printf("Error %s", e)
		return
	}
	go func() {
		start := time.Now()
		idx := manifold.MakeVPIndex()
		log.Printf("Index built for %d words %s %d metric calls\n", manifold.WordCount(), time.Now().Sub(start), index.MetricCalls)

		search := func(word string) {
			index.MetricCalls = 0
			govector.CacheHit = 0
			govector.CacheMiss = 0
			start := time.Now()
			words, distances := idx.Search(word, 35, 1)
			fmt.Printf("Search for:%s %d metric calls hit:%d miss: %d\n", time.Now().Sub(start), index.MetricCalls, govector.CacheHit, govector.CacheMiss)
			fmt.Println()
			fmt.Printf("%12s \n", "Angular")
			fmt.Println()
			for i, word := range words {
				fmt.Printf("%4d | %4.7f %s\n", i, distances[i], word)
			}
		}

		if len(word) > 0 {
			search(word)
		} else {
			fi := bufio.NewReader(os.NewFile(0, "stdin"))
			for {
				fmt.Printf("query: ")
				if query, ok := readline(fi); ok {
					search(query)
				} else {
					break
				}
			}
		}
	}()

	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	<-ch
	log.Println("Closing manifold")
	manifold.Close()
	return
}
