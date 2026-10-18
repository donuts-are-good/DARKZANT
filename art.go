package main

import (
	"bytes"
	"embed"
	"image"
	_ "image/png"
	"log"

	"github.com/hajimehoshi/ebiten/v2"
)

// The art is cropped and palette-reduced from the four reference screens.
//
//go:embed assets/*.png
var assetFS embed.FS

var (
	imgTown, imgTownFree, imgForest, imgMap *ebiten.Image
	imgTitle, imgOver                       *ebiten.Image
	imgBrazgant, imgPazuzu, imgLord         *ebiten.Image
	imgGhoul                                *ebiten.Image
	imgBoss                                 [4]*ebiten.Image
)

func loadArt() {
	imgTown = loadPNG("town")
	imgTownFree = loadPNG("town_free")
	imgForest = loadPNG("forest")
	imgMap = loadPNG("map")
	imgTitle = loadPNG("title")
	imgOver = loadPNG("gameover")
	imgBrazgant = loadPNG("brazgant")
	imgPazuzu = loadPNG("pazuzu")
	imgLord = loadPNG("lord")
	imgGhoul = loadPNG("ghoul")
	for i := range imgBoss {
		imgBoss[i] = loadPNG("boss" + string(rune('1'+i)))
	}
}

func loadPNG(name string) *ebiten.Image {
	b, err := assetFS.ReadFile("assets/" + name + ".png")
	if err != nil {
		log.Fatal(err)
	}
	m, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		log.Fatal(err)
	}
	return ebiten.NewImageFromImage(m)
}

func rectWH(w, h int) image.Rectangle { return image.Rect(0, 0, w, h) }
