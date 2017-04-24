package main

import (
	"log"

	"time"

	"github.com/vseledkin/govector"
)

func Nearest() (e error) {
	var manifold *govector.Manifold
	manifold, e = govector.NewManifold(input)
	if e != nil {
		log.Printf("Error %s", e)
		return
	}
	e = manifold.Open()
	defer func() {
		manifold.Close()
	}()
	if e != nil {
		log.Printf("Error %s", e)
		return
	}
	start := time.Now()
	var vector []float32
	vector, e = manifold.GetVector(word)
	if e != nil {
		log.Printf("Error %s", e)
		return
	}

	log.Printf("%s\n%v\n%s\n", word, vector, time.Now().Sub(start))

	manifold.Visit(func(key string) {
		vector, e = manifold.GetVector(key)
		if e != nil {
			log.Printf("Error %s", e)
			return
		}

		log.Printf(key, vector)
	})

	return
}
