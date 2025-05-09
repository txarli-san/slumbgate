package main

import (
	"flag"
	"log"
	"math/rand"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

var godMode bool

func init() {
	flag.BoolVar(&godMode, "godmode", false, "Enable god mode for testing")
	flag.Parse()
}

func main() {
	rand.Seed(time.Now().UnixNano())

	game := StartScreen(godMode)
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Slumb Gate")
	ebiten.SetFullscreen(true)
	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
