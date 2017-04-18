package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

const (
	build   = "build"
	nearest = "nearest"
)

var threads int
var input, output string
var cwd string

var word string

func main() {
	buildCommand := flag.NewFlagSet(build, flag.ExitOnError)
	buildCommand.StringVar(&input, "input", "", "file to load vectors from")
	buildCommand.IntVar(&threads, "threads", 2, "paralelizm factor")
	buildCommand.StringVar(&output, "output", "", "dir to output index to")

	nearestCommand := flag.NewFlagSet(build, flag.ExitOnError)
	nearestCommand.StringVar(&input, "input", "", "dir to load vectors from")
	nearestCommand.IntVar(&threads, "threads", 2, "paralelizm factor")
	nearestCommand.StringVar(&word, "word", "", "word to search nearest to")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "utility <command> arguments\n")
		fmt.Fprintf(os.Stderr, "commands are:\n")
		fmt.Fprintf(os.Stderr, "%s\n", build)
		buildCommand.PrintDefaults()
		fmt.Fprintf(os.Stderr, "%s\n", nearest)
		nearestCommand.PrintDefaults()

		flag.PrintDefaults()
	}
	flag.Parse()
	log.SetOutput(os.Stderr)

	if len(os.Args) == 1 {
		flag.Usage()
		os.Exit(1)
	}

	cwd, _ = os.Getwd()
	log.Printf("Starting in %s directory.", cwd)
	switch os.Args[1] {
	case build:
		buildCommand.Parse(os.Args[2:])
	case nearest:
		nearestCommand.Parse(os.Args[2:])
	default:
		log.Printf("%q is not valid command.\n", os.Args[1])
		os.Exit(1)
	}

	// BUILD COMMAND ISSUED
	if buildCommand.Parsed() {
		if input == "" {
			buildCommand.PrintDefaults()
			return
		}
		if output == "" {
			output = input + ".govin"
		}
		BuildText()
		return
	}

	// NEAREST COMMAND ISSUED
	if nearestCommand.Parsed() {
		if word == "" {
			nearestCommand.PrintDefaults()
			return
		}
		Nearest()
		return
	}
}
