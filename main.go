package main

import (
	"flag"
	"io/ioutil"
	"log"
)

func main() {
	debug := flag.Bool("debug", false, "debug")
	flag.Parse()
	content, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal("ReadFile", err)
	}

	d := &Data{}
	err = readData(content, d)
	if err != nil {
		log.Fatalf("readData %v", err)
	}
	if (*debug) {
		analyze(d)
	} else {
		Convert(d)
	}
}
