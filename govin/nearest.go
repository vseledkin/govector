package main

import (
	"log"

	"time"

	"bytes"
	"encoding/binary"

	"github.com/vseledkin/bitcask"
)

func Nearest() (e error) {
	bc, err := bitcask.Open(input, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer bc.Close()
	start := time.Now()
	byteval, e := bc.Get([]byte(word))
	if e != nil {
		log.Printf("Error %s", e)
	}
	var vector [128]float32
	buf := bytes.NewReader(byteval)
	e = binary.Read(buf, binary.LittleEndian, &vector)
	if e != nil {
		log.Println("binary.Read failed:", e)
	}

	log.Printf("%s\n%v\n", word, byteval)
	log.Printf("%s\n%v\n%s\n", word, vector, time.Now().Sub(start))

	return
}
