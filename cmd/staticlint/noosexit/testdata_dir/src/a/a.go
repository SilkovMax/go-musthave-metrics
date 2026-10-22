//go:build ignore

package main

import (
	"log"
	"os"
)

func main() {
	os.Exit(1) // want "do not use os.Exit in main function"
	log.Fatal("error")
}

func helper() {
	os.Exit(1)
}