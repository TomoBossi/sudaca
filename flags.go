package main

import (
	"flag"
)

type flags struct {
	readPath string
	savePath string
}

func newFlags() (*flags, error) {
	readPath := flag.String("read-path", "", "DEFAULT \"\" - Path of a saved Sudoku definition to keep working on. If not provided or \"\", starts a new Sudoku from scratch.")
	savePath := flag.String("save-path", "", "DEFAULT \"\" - Path where the Sudoku definition will be saved. If not provided or \"\", the read-path will be used as save-path. If read-path is not provided either, saving is disabled.")
	flag.Parse()

	return &flags{
		readPath: *readPath,
		savePath: *savePath,
	}, nil
}
