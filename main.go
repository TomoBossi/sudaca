package main

import (
	"flag"
	"fmt"
)

func main() {
	flags, err := newFlags()
	if err != nil {
		flag.Usage()
		fmt.Println()
		panic(err)
	}

	fmt.Println(flags.readPath)
	fmt.Println(flags.savePath)
}
