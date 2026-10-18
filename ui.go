package main

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/basicfont"
)

var (
	black = color.RGBA{0, 0, 0, 255}
	white = color.RGBA{255, 255, 255, 255}
	red   = color.RGBA{255, 70, 60, 255}
	face  = text.NewGoXFace(basicfont.Face7x13)
)

const (
	charW   = 7
	lineH   = 13
	logCols = 44
	logRows = 8
)

// Scene art always sits in this 320x200 window, framed by the HUD.
const (
	sceneX = 80
	sceneY = 30
	sceneW = 320
	sceneH = 200
)

// theme is the palette of one screen, after the reference shot it came from.
type theme struct {
	frame, dim, accent, text, hot color.RGBA
}

var (
	thTown   = theme{rgb(120, 230, 190), rgb(30, 90, 70), rgb(255, 90, 70), rgb(90, 255, 110), white}
	thForest = theme{rgb(150, 230, 90), rgb(50, 90, 30), rgb(230, 90, 230), rgb(150, 230, 90), rgb(255, 170, 255)}
	thMap    = theme{rgb(170, 220, 150), rgb(60, 90, 55), rgb(255, 60, 255), rgb(120, 255, 255), white}
	thDemon  = theme{rgb(255, 85, 190), rgb(110, 30, 90), rgb(90, 90, 255), rgb(85, 255, 85), white}
)

func rgb(r, g, b uint8) color.RGBA { return color.RGBA{r, g, b, 255} }

func drawText(dst *ebiten.Image, s string, x, y float64, clr color.Color) {
	drawTextScaled(dst, s, x, y, 1, clr)
}

func drawTextScaled(dst *ebiten.Image, s string, x, y, scale float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}

// drawCentered draws text centred on cx.
func drawCentered(dst *ebiten.Image, s string, cx, y, scale float64, clr color.Color) {
	drawTextScaled(dst, s, cx-float64(len(s)*charW)*scale/2, y, scale, clr)
}

func wrap(s string, cols int) []string {
	var lines []string
	line := ""
	for _, w := range strings.Fields(s) {
		if line != "" && len(line)+1+len(w) > cols {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += w
	}
	return append(lines, line)
}

func fill(dst *ebiten.Image, x, y, w, h float64, clr color.Color) {
	vector.FillRect(dst, float32(x), float32(y), float32(w), float32(h), clr, false)
}

func stroke(dst *ebiten.Image, x, y, w, h, t float64, clr color.Color) {
	vector.StrokeRect(dst, float32(x), float32(y), float32(w), float32(h), float32(t), clr, false)
}

func line(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color) {
	vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), 1, clr, false)
}

// box draws a black panel with the double border the reference HUDs use.
func box(dst *ebiten.Image, x, y, w, h float64, th theme) {
	fill(dst, x, y, w, h, black)
	stroke(dst, x+1, y+1, w-2, h-2, 2, th.frame)
	stroke(dst, x+4.5, y+4.5, w-9, h-9, 1, th.dim)
	for _, p := range [][2]float64{{x, y}, {x + w - 4, y}, {x, y + h - 4}, {x + w - 4, y + h - 4}} {
		fill(dst, p[0], p[1], 4, 4, th.accent)
	}
}

// bar is a segmented gauge, like the life strips on the reference screens.
func bar(dst *ebiten.Image, x, y, w, h float64, frac float64, on, off color.Color) {
	segs := int(w / 3)
	lit := int(math.Ceil(frac * float64(segs)))
	if frac <= 0 {
		lit = 0
	}
	for i := range segs {
		c := off
		if i < lit {
			c = on
		}
		fill(dst, x+float64(i*3), y, 2, h, c)
	}
}

// arch is a half-ring gauge built from blocks, as in the forest screen.
func arch(dst *ebiten.Image, cx, cy, r float64, frac float64, on, off color.Color) {
	const segs = 14
	lit := int(math.Round(frac * segs))
	for i := range segs {
		a := math.Pi + (float64(i)+0.5)/segs*math.Pi
		c := off
		if i < lit {
			c = on
		}
		for d := -3.0; d <= 3; d++ {
			x := cx + math.Cos(a)*(r+d)
			y := cy + math.Sin(a)*(r+d)
			fill(dst, math.Round(x)-1, math.Round(y)-1, 3, 3, c)
		}
	}
}

// dim returns a colour at a fraction of its brightness.
func dim(c color.RGBA, f float64) color.RGBA {
	return color.RGBA{uint8(float64(c.R) * f), uint8(float64(c.G) * f), uint8(float64(c.B) * f), 255}
}

// drawScene draws art into the scene window, scaled to fill it.
func drawScene(dst, img *ebiten.Image, bright float32) {
	op := &ebiten.DrawImageOptions{}
	b := img.Bounds()
	op.GeoM.Scale(sceneW/float64(b.Dx()), sceneH/float64(b.Dy()))
	op.GeoM.Translate(sceneX, sceneY)
	op.ColorScale.Scale(bright, bright, bright, 1)
	dst.DrawImage(img, op)
}

// hud draws every panel around the scene window.
func (g *Game) hud(dst *ebiten.Image, th theme, title string, active, target int, flash [3]int) {
	// Top bar.
	box(dst, 0, 0, 78, 28, th)
	for i := range 4 {
		x := 7.0 + float64(i)*17
		stroke(dst, x, 8, 13, 13, 1, th.frame)
		glyph(dst, x+3, 11, i+g.tick/40, th.accent)
	}
	box(dst, 78, 0, 324, 28, th)
	drawCentered(dst, title, W/2, 1, 2, th.frame)
	box(dst, 402, 0, 78, 28, th)
	drawCentered(dst, fmt.Sprintf("DAY %d", g.st.Day), 441, 8, 1, th.text)

	// Scene frame.
	stroke(dst, sceneX-2, sceneY-2, sceneW+4, sceneH+4, 2, th.frame)
	for x := sceneX + 4; x < sceneX+sceneW-4; x += 8 {
		fill(dst, float64(x), sceneY+sceneH+1, 1, 2, th.accent)
	}

	g.partyPanel(dst, th, active, target, flash)
	g.statPanel(dst, th)
}

func (g *Game) partyPanel(dst *ebiten.Image, th theme, active, target int, flash [3]int) {
	box(dst, 0, 28, 78, 206, th)
	for i := range g.st.Heroes {
		h := &g.st.Heroes[i]
		y := 34 + float64(i)*66
		nc := th.text
		if h.HP <= 0 {
			nc = dim(th.text, 0.4)
		}
		if i == active && g.tick/12%2 == 0 {
			nc = th.hot
		}
		if i == target {
			fill(dst, 7, y, 64, 62, th.dim)
		}
		if flash[i] > 0 && flash[i]/3%2 == 0 {
			fill(dst, 7, y, 64, 62, dim(red, 0.6))
		}
		mark := " "
		if i == active {
			mark = ">"
		}
		drawText(dst, mark+h.Name, 6, y, nc)
		drawText(dst, h.Class, 14, y+11, dim(th.frame, 0.7))
		bar(dst, 8, y+26, 63, 4, float64(h.HP)/float64(h.MaxHP), th.accent, th.dim)
		drawText(dst, fmt.Sprintf("HP%7s", fmt.Sprintf("%d/%d", h.HP, h.MaxHP)), 6, y+30, th.text)
		bar(dst, 8, y+44, 63, 3, float64(h.MP)/float64(max(h.MaxMP, 1)), th.frame, th.dim)
		drawText(dst, fmt.Sprintf("MP%7s", fmt.Sprintf("%d/%d", h.MP, h.MaxMP)), 6, y+47, th.text)
	}
}

func (g *Game) statPanel(dst *ebiten.Image, th theme) {
	s := &g.st
	box(dst, 402, 28, 78, 206, th)
	drawCentered(dst, fmt.Sprintf("LV%d", s.Level), 441, 33, 2, th.frame)
	arch(dst, 441, 94, 24, float64(s.XP)/float64(xpNeed(s.Level)), th.accent, th.dim)
	drawCentered(dst, "XP", 441, 80, 1, th.dim)
	drawCentered(dst, fmt.Sprintf("%d/%d", s.XP, xpNeed(s.Level)), 441, 97, 1, th.text)
	drawCentered(dst, fmt.Sprintf("G %d", s.Gold), 441, 112, 1, th.hot)
	drawCentered(dst, fmt.Sprintf("P%d E%d X%d", s.Items[0], s.Items[1], s.Items[2]), 441, 126, 1, th.text)
	g.sigilWheel(dst, 441, 177, 24, th)
	drawCentered(dst, "SIGILS", 441, 207, 1, th.dim)
}

// sigilWheel lights one star point per slain boss, as in the Brazgant screen.
func (g *Game) sigilWheel(dst *ebiten.Image, cx, cy, r float64, th theme) {
	vector.StrokeCircle(dst, float32(cx), float32(cy), float32(r+4), 1, th.dim, false)
	pts := make([][2]float64, 5)
	for i := range pts {
		a := -math.Pi/2 + float64(i)*2*math.Pi/5
		pts[i] = [2]float64{cx + math.Cos(a)*r, cy + math.Sin(a)*r}
	}
	for i := range pts {
		j := (i + 2) % 5
		line(dst, pts[i][0], pts[i][1], pts[j][0], pts[j][1], th.dim)
	}
	for i, p := range pts {
		c := th.dim
		if g.st.Slain[i] {
			c = th.accent
		}
		fill(dst, p[0]-2, p[1]-2, 5, 5, c)
	}
	c := th.dim
	if g.st.Slain[bossLord] {
		c = th.hot
	}
	fill(dst, cx-2, cy-2, 5, 5, c)
}

// glyph draws a small rune; the HUDs are covered in these.
func glyph(dst *ebiten.Image, x, y float64, n int, c color.Color) {
	switch n % 4 {
	case 0:
		line(dst, x, y, x+6, y+6, c)
		line(dst, x+6, y, x, y+6, c)
	case 1:
		line(dst, x+3, y, x+3, y+7, c)
		line(dst, x, y+2, x+6, y+2, c)
	case 2:
		stroke(dst, x+1, y+1, 5, 5, 1, c)
	case 3:
		line(dst, x, y+6, x+3, y, c)
		line(dst, x+3, y, x+6, y+6, c)
	}
}

// bottom draws the menu box and the message log.
func (g *Game) bottom(dst *ebiten.Image, th theme, m *menu, header string) {
	box(dst, 0, 234, 152, 126, th)
	if header != "" {
		drawText(dst, header, 10, 238, th.frame)
	}
	if m != nil {
		m.draw(dst, 10, 238+lineH+1, th, g.tick)
	}
	box(dst, 152, 234, 328, 126, th)
	start := max(0, len(g.log)-logRows)
	for i, l := range g.log[start:] {
		c := th.text
		if start+i < len(g.log)-2 {
			c = dim(th.text, 0.65)
		}
		drawText(dst, l, 162, 240+float64(i)*lineH+1, c)
	}
}

// menu is a vertical list of choices; off marks choices that cannot be taken now.
type menu struct {
	items []string
	off   []bool
	sel   int
}

func newMenu(items ...string) menu { return menu{items: items, off: make([]bool, len(items))} }

// update returns the chosen index, or -1. It returns -2 on cancel.
func (m *menu) update() int {
	n := len(m.items)
	if n == 0 {
		return -1
	}
	m.sel = min(m.sel, n-1)
	switch {
	case up():
		m.sel = (m.sel + n - 1) % n
		play("move")
	case down():
		m.sel = (m.sel + 1) % n
		play("move")
	case confirm():
		if m.off[m.sel] {
			play("no")
			return -1
		}
		play("ok")
		return m.sel
	case cancel():
		return -2
	}
	return -1
}

func (m *menu) draw(dst *ebiten.Image, x, y float64, th theme, tick int) {
	const rows = 7
	top := max(0, min(m.sel-rows+1, len(m.items)-rows))
	for i := top; i < min(len(m.items), top+rows); i++ {
		c := th.text
		if m.off[i] {
			c = dim(th.text, 0.4)
		}
		yy := y + float64(i-top)*lineH
		if i == m.sel {
			fill(dst, x-3, yy+1, 138, lineH, th.dim)
			if tick/10%3 != 0 {
				drawText(dst, ">", x-2, yy, th.hot)
			}
			c = th.hot
		}
		drawText(dst, m.items[i], x+6, yy, c)
	}
}
