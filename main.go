package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/hanwen/go-enc2ly/encore"
)

func main() {
	debug := flag.Bool("debug", false, "debug")
	flag.Parse()
	content, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal("ReadFile", err)
	}

	d, err := encore.ReadData(content)
	if err != nil {
		log.Fatalf("readData %v", err)
	}

	if *debug {
		analyze(d)
	} else {
		Convert(d)
	}
}
