package main

import (
	"flag"
	"log"
)

func main() {
	flag.Parse()

	if err := runGrpc(); err != nil {
		log.Fatal(err)
	}

	if err := runGrpc(); err != nil {
		log.Fatal(err)
	}
}
