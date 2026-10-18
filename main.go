// Darkzant: a retro CGA/EGA-style fantasy RPG built from four screenshots.
package main

import (
	_ "embed"
	"log"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// The game draws at this size; the CRT pass scales it to the window.
const (
	W = 480
	H = 360
)

//go:embed crt.kage
var crtSrc []byte

type scene int

const (
	sTitle scene = iota
	sTown
	sShop
	sItems
	sForest
	sMap
	sBattle
	sOver
	sWin
)

type Game struct {
	scene  scene
	canvas *ebiten.Image
	crt    *ebiten.Shader
	tick   int

	st     State
	log    []string
	menu   menu
	battle *battle
	back   scene // where a battle returns to
}

func main() {
	loadArt()
	initSound()
	g := &Game{canvas: ebiten.NewImageWithOptions(rectWH(W, H), &ebiten.NewImageOptions{Unmanaged: true})}
	var err error
	if g.crt, err = ebiten.NewShader(crtSrc); err != nil {
		log.Fatal(err)
	}
	g.goTitle()
	ebiten.SetWindowTitle("DARKZANT")
	ebiten.SetWindowSize(W*2, H*2)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}

func (g *Game) Update() error {
	g.tick++
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (ebiten.IsKeyPressed(ebiten.KeyAlt) && inpututil.IsKeyJustPressed(ebiten.KeyEnter)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
		return nil
	}
	switch g.scene {
	case sTitle:
		return g.updateTitle()
	case sTown:
		g.updateTown()
	case sShop:
		g.updateShop()
	case sItems:
		g.updateItems()
	case sForest:
		g.updateForest()
	case sMap:
		g.updateMap()
	case sBattle:
		g.battle.update(g)
	case sOver:
		g.updateOver()
	case sWin:
		g.updateWin()
	}
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	c := g.canvas
	c.Fill(black)
	switch g.scene {
	case sTitle:
		g.drawTitle(c)
	case sTown, sShop, sItems:
		g.drawTown(c)
	case sForest:
		g.drawForest(c)
	case sMap:
		g.drawMap(c)
	case sBattle:
		g.battle.draw(g, c)
	case sOver:
		g.drawOver(c)
	case sWin:
		g.drawWin(c)
	}
	g.present(screen)
}

// present scales the canvas into the window through the CRT shader.
func (g *Game) present(screen *ebiten.Image) {
	sw, sh := float32(screen.Bounds().Dx()), float32(screen.Bounds().Dy())
	s := min(sw/W, sh/H)
	dw, dh := W*s, H*s
	x0, y0 := (sw-dw)/2, (sh-dh)/2
	// Screen shake when the party is hit.
	if g.scene == sBattle && g.battle.shake > 0 {
		x0 += float32(rand.IntN(5)-2) * s
		y0 += float32(rand.IntN(5)-2) * s
	}
	v := []ebiten.Vertex{
		{DstX: x0, DstY: y0, SrcX: 0, SrcY: 0},
		{DstX: x0 + dw, DstY: y0, SrcX: W, SrcY: 0},
		{DstX: x0, DstY: y0 + dh, SrcX: 0, SrcY: H},
		{DstX: x0 + dw, DstY: y0 + dh, SrcX: W, SrcY: H},
	}
	for i := range v {
		v[i].ColorR, v[i].ColorG, v[i].ColorB, v[i].ColorA = 1, 1, 1, 1
	}
	op := &ebiten.DrawTrianglesShaderOptions{Uniforms: map[string]any{"Scale": s}}
	op.Images[0] = g.canvas
	screen.DrawTrianglesShader(v, []uint16{0, 1, 2, 1, 3, 2}, g.crt, op)
}

func (g *Game) Layout(ow, oh int) (int, int) {
	f := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(ow) * f), int(float64(oh) * f)
}

// Input helpers. Arrows move, Enter/Space/Z confirm, Esc/X/Backspace go back.
func pressed(keys ...ebiten.Key) bool {
	for _, k := range keys {
		if inpututil.IsKeyJustPressed(k) {
			return true
		}
		// Gentle auto-repeat for held arrows.
		if d := inpututil.KeyPressDuration(k); d > 18 && d%5 == 0 && isArrow(k) {
			return true
		}
	}
	return false
}

func isArrow(k ebiten.Key) bool {
	return k == ebiten.KeyUp || k == ebiten.KeyDown || k == ebiten.KeyLeft || k == ebiten.KeyRight
}

func confirm() bool { return pressed(ebiten.KeyEnter, ebiten.KeySpace, ebiten.KeyZ) }
func cancel() bool  { return pressed(ebiten.KeyEscape, ebiten.KeyX, ebiten.KeyBackspace) }
func up() bool      { return pressed(ebiten.KeyUp, ebiten.KeyW) }
func down() bool    { return pressed(ebiten.KeyDown, ebiten.KeyS) }

func chance(p float64) bool { return rand.Float64() < p }

// say adds a line to the message log, wrapped to the log width.
func (g *Game) say(s string) {
	g.log = append(g.log, wrap(s, logCols)...)
	if len(g.log) > 40 {
		g.log = g.log[len(g.log)-40:]
	}
}
