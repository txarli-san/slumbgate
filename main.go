package main

import (
	"flag"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

var godMode bool

func main() {
	flag.BoolVar(&godMode, "godmode", false, "Enable god mode for testing")
	flag.Parse()

	game := StartScreen(godMode)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate")
	ebiten.SetFullscreen(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
