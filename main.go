package main

import (
	"flag"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

func main() {
	flags, err := newFlags()
	if err != nil {
		flag.Usage()
		fmt.Println()
		panic(err)
	}

	g, err := NewGameEditor(flags.readPath, flags.savePath)
	if err != nil {
		panic(err)
	}
	ebiten.SetWindowSize(800, 600)
	ebiten.SetWindowTitle("Sudaca Editor")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		panic(err)
	}
}
