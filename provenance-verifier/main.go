package main

import (
	"fmt"
)

func usage(p string) {
	panic(fmt.Sprintf("Usage: %s TODO\n", p))
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	fmt.Println("verifier")
}
