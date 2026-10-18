package main

import (
	"fmt"
	"image"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

var thAsh = theme{rgb(255, 130, 70), rgb(100, 40, 20), rgb(255, 60, 30), rgb(255, 200, 120), white}

// enter switches scene and builds its menu.
func (g *Game) enter(sc scene) {
	g.scene = sc
	sel := g.menu.sel
	switch sc {
	case sTown:
		inn := fmt.Sprintf("INN       %dG", innCost(g.st.Level))
		g.menu = newMenu("SMITH", "ITEMS", inn, "CATHEDRAL", "THE WILDS", "FACE PAZUZU")
		if g.st.Slain[bossPazuzu] {
			g.menu.items[5] = "THE GATES"
		}
		g.menu.sel = min(sel, 5)
	case sShop:
		g.menu = newMenu()
		for _, list := range [][]Gear{weapons, armours} {
			for _, gr := range list[1:] {
				g.menu.items = append(g.menu.items, fmt.Sprintf("%-12s%4d", gr.Name, gr.Price))
			}
		}
		g.menu.items = append(g.menu.items, "LEAVE")
		g.menu.off = make([]bool, len(g.menu.items))
		for i := range 3 {
			g.menu.off[i] = i+1 <= g.st.Weapon
			g.menu.off[i+3] = i+1 <= g.st.Armour
		}
	case sItems:
		g.menu = newMenu()
		for i, n := range itemNames {
			g.menu.items = append(g.menu.items, fmt.Sprintf("%-12s%4d", n, itemPrices[i]))
		}
		g.menu.items = append(g.menu.items, "LEAVE")
		g.menu.off = make([]bool, len(g.menu.items))
	case sForest:
		g.menu = newMenu("HUNT", "REST BY FIRE", "SEARCH", "TOWN")
	case sMap:
		g.menu = newMenu("HOODED ASH", "MAD GORMOTH", "WITCH VYRELLE", "KTHULGOR", "BRAZGANT", "TOWN")
		for i := range 4 {
			g.menu.off[i] = g.st.Slain[bossAsh+i]
		}
		g.menu.off[4] = g.st.bossesDown() < 4 || g.st.Slain[bossLord]
		g.menu.sel = min(sel, 5)
	}
}

func innCost(level int) int { return 5 + 5*level }

// ---- Title ----

func (g *Game) goTitle() {
	g.scene = sTitle
	g.menu = newMenu("NEW GAME", "CONTINUE")
	if canQuit {
		g.menu.items = append(g.menu.items, "QUIT")
		g.menu.off = append(g.menu.off, false)
	}
	_, ok := loadState()
	g.menu.off[1] = !ok
	if ok {
		g.menu.sel = 1
	}
}

func (g *Game) updateTitle() error {
	switch g.menu.update() {
	case 0:
		g.st = newState()
		g.log = nil
		g.say("Pazuzu, demon of all, holds Darkzant Town.")
		g.say("Past the gates wait four boss monsters, and BRAZGANT the Demon Lord.")
		g.say("Grow strong in THE WILDS. Then face Pazuzu.")
		g.enter(sTown)
	case 1:
		g.st, _ = loadState()
		g.log = nil
		g.say("The candles flicker. Your tale goes on.")
		g.enter(sTown)
	case 2:
		return ebiten.Termination
	}
	return nil
}

func (g *Game) drawTitle(dst *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	dst.DrawImage(imgTitle, op)
	th := thMap
	fill(dst, 178, 14, 128, 22, black)
	drawCentered(dst, "DARKZANT", W/2, 11, 2, th.hot)
	box(dst, 164, 280, 152, 64, th)
	g.menu.draw(dst, 182, 287, th, g.tick)
}

// ---- Town ----

func (g *Game) updateTown() {
	switch g.menu.update() {
	case 0:
		g.say(fmt.Sprintf("SMITH: Steel for the dark. You carry %s and %s.", weapons[g.st.Weapon].Name, armours[g.st.Armour].Name))
		g.enter(sShop)
	case 1:
		g.say("The sign says CANCER ITEMS. The keeper does not explain.")
		g.enter(sItems)
	case 2:
		cost := innCost(g.st.Level)
		if g.st.Gold < cost {
			play("no")
			g.say("INN: No coin, no bed.")
			return
		}
		g.st.Gold -= cost
		g.st.restore()
		g.st.Day++
		play("heal")
		g.say("You sleep at the inn. All are restored.")
	case 3:
		if err := g.st.save(); err != nil {
			g.say("The candles gutter. SAVE FAILED: " + err.Error())
			return
		}
		g.say("You pray in the cathedral. Your deeds are written. (SAVED)")
	case 4:
		g.say("You walk into the wilds. A fire burns in the clearing.")
		g.enter(sForest)
	case 5:
		if g.st.Slain[bossPazuzu] {
			g.say("The gates open. Four boss monsters wait, and the Demon Lord beyond.")
			g.enter(sMap)
			return
		}
		g.startBattle(bossFoe(bossPazuzu), sTown)
	}
}

func (g *Game) updateShop() {
	i := g.menu.update()
	switch {
	case i == -2 || i == 6:
		g.enter(sTown)
	case i >= 0:
		list, tier := weapons, &g.st.Weapon
		if i >= 3 {
			list, tier, i = armours, &g.st.Armour, i-3
		}
		gr := list[i+1]
		if g.st.Gold < gr.Price {
			play("no")
			g.say("SMITH: Come back with more gold.")
			return
		}
		g.st.Gold -= gr.Price
		*tier = i + 1
		play("coin")
		g.say(fmt.Sprintf("You buy %s. (+%d)", gr.Name, gr.Bonus))
		sel := g.menu.sel
		g.enter(sShop)
		g.menu.sel = sel
	}
}

func (g *Game) updateItems() {
	i := g.menu.update()
	switch {
	case i == -2 || i == 3:
		g.enter(sTown)
	case i >= 0:
		if g.st.Gold < itemPrices[i] {
			play("no")
			g.say("Not enough gold.")
			return
		}
		g.st.Gold -= itemPrices[i]
		g.st.Items[i]++
		play("coin")
		g.say(fmt.Sprintf("You buy a %s. You have %d.", itemNames[i], g.st.Items[i]))
	}
}

func (g *Game) drawTown(dst *ebiten.Image) {
	img := imgTown
	if g.st.Slain[bossPazuzu] {
		img = imgTownFree
	}
	drawScene(dst, img, 1)
	header := map[scene]string{sTown: "TOWN SQUARE", sShop: "SMITH", sItems: "ITEMS"}[g.scene]
	g.hud(dst, thTown, "DARKZANT TOWN", -1, -1, [3]int{})
	g.bottom(dst, thTown, &g.menu, header)
}

// ---- Forest ----

type wildFoe struct {
	name                 string
	tint                 [3]float32
	hp, atk, def, xp, gp int
	minLevel             int
}

var wildFoes = []wildFoe{
	{"GLOOM EYES", [3]float32{1, 1, 1}, 30, 7, 1, 8, 6, 1},
	{"WOOD GHOULS", [3]float32{0.6, 1.4, 0.7}, 48, 10, 3, 14, 11, 1},
	{"BARK IMPS", [3]float32{1.5, 0.7, 0.5}, 40, 13, 2, 16, 14, 2},
	{"MIRE BEASTS", [3]float32{1.4, 0.6, 1.5}, 80, 15, 5, 30, 25, 4},
	{"HOLLOW KINGS", [3]float32{0.5, 1.5, 1.5}, 140, 22, 8, 60, 50, 6},
}

func (g *Game) wildFoe() *Foe {
	var pool []wildFoe
	for _, w := range wildFoes {
		if w.minLevel <= g.st.Level {
			pool = append(pool, w)
		}
	}
	w := pool[rand.IntN(len(pool))]
	k := 1 + 0.12*float64(g.st.Level-1)
	sc := func(n int) int { return int(float64(n) * k) }
	return &Foe{
		Name: w.name, Art: imgGhoul, BG: imgForest, Tint: w.tint, Theme: thForest,
		HP: sc(w.hp), MaxHP: sc(w.hp), Atk: sc(w.atk), Def: w.def + g.st.Level/2,
		XP: sc(w.xp), Gold: sc(w.gp), Boss: -1,
		Moves: []move{{"claw", mvHit, 1, 6}, {"FRENZY", mvTwice, 0.7, 2}},
	}
}

func (g *Game) updateForest() {
	switch g.menu.update() {
	case 0:
		g.say("You stalk the black trees...")
		g.startBattle(g.wildFoe(), sForest)
	case 1:
		g.st.Day++
		if chance(0.3) {
			g.say("Eyes in the dark! You are ambushed at the fire!")
			g.startBattle(g.wildFoe(), sForest)
			return
		}
		for i := range g.st.Heroes {
			h := &g.st.Heroes[i]
			if h.HP > 0 {
				h.HP = min(h.MaxHP, h.HP+h.MaxHP*35/100)
				h.MP = min(h.MaxMP, h.MP+h.MaxMP*35/100)
			}
		}
		play("heal")
		g.say("You rest by the fire. The standing regain strength.")
	case 2:
		switch r := rand.Float64(); {
		case r < 0.4:
			n := 5 + rand.IntN(10*g.st.Level+5)
			g.st.Gold += n
			play("coin")
			g.say(fmt.Sprintf("Bones in the roots. You find %d gold.", n))
		case r < 0.6:
			g.st.Items[0]++
			play("coin")
			g.say("A dead pilgrim's pack. You find a POTION.")
		default:
			g.say("Something stirs in the leaves!")
			g.startBattle(g.wildFoe(), sForest)
		}
	case 3, -2:
		g.say("You return to Darkzant Town.")
		g.enter(sTown)
	}
}

func (g *Game) drawForest(dst *ebiten.Image) {
	drawScene(dst, imgForest, 1)
	// The campfire flickers.
	fire := imgForest.SubImage(image.Rect(133, 101, 205, 171)).(*ebiten.Image)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(sceneX+133, sceneY+101)
	f := float32(0.15 + 0.2*math.Abs(math.Sin(float64(g.tick)/5)*math.Sin(float64(g.tick)/13)))
	op.ColorScale.Scale(f, f*0.6, f*0.4, 1)
	op.Blend = ebiten.BlendLighter
	dst.DrawImage(fire, op)
	g.hud(dst, thForest, "THE WILDS", -1, -1, [3]int{})
	g.bottom(dst, thForest, &g.menu, "CAMPFIRE")
}

// ---- Map of the boss monsters ----

// Regions of the map art, in scene pixels: Ash, Gormoth, Vyrelle, Kthulgor, Lord.
var mapRegions = []image.Rectangle{
	image.Rect(10, 8, 106, 87),
	image.Rect(186, 8, 315, 87),
	image.Rect(8, 114, 110, 193),
	image.Rect(219, 122, 316, 193),
	image.Rect(112, 48, 210, 176),
}

func (g *Game) updateMap() {
	switch i := g.menu.update(); {
	case i == 5 || i == -2:
		g.say("You return to Darkzant Town.")
		g.enter(sTown)
	case i == 4:
		g.startBattle(bossFoe(bossLord), sMap)
	case i >= 0:
		g.startBattle(bossFoe(bossAsh+i), sMap)
	}
}

func (g *Game) drawMap(dst *ebiten.Image) {
	drawScene(dst, imgMap, 1)
	for i, r := range mapRegions {
		r = r.Add(image.Pt(sceneX, sceneY))
		x, y, w, h := float64(r.Min.X), float64(r.Min.Y), float64(r.Dx()), float64(r.Dy())
		if g.st.Slain[bossAsh+i] {
			for t := -1.0; t <= 1; t++ {
				line(dst, x+t, y, x+w+t, y+h, red)
				line(dst, x+w+t, y, x+t, y+h, red)
			}
		}
		if i == 4 && g.st.bossesDown() < 4 {
			fill(dst, x+w/2-30, y+h/2-8, 60, 15, black)
			drawCentered(dst, "SEALED", x+w/2, y+h/2-7, 1, red)
		}
		if i == g.menu.sel && g.tick/8%2 == 0 {
			stroke(dst, x, y, w, h, 2, white)
		}
	}
	g.hud(dst, thMap, "THE GATES", -1, -1, [3]int{})
	g.bottom(dst, thMap, &g.menu, fmt.Sprintf("BOSSES %d/4", g.st.bossesDown()))
}

// bossFoe builds a boss by its slot in State.Slain.
func bossFoe(i int) *Foe {
	f := &Foe{Boss: i, BG: imgMap, Tint: [3]float32{1, 1, 1}, Theme: thMap}
	switch i {
	case bossPazuzu:
		f.Name, f.Art, f.BG, f.Theme = "PAZUZU OF ALL", imgPazuzu, imgTown, thTown
		f.MaxHP, f.Atk, f.Def, f.XP, f.Gold = 320, 20, 6, 80, 150
		f.Moves = []move{{"rakes", mvHit, 1, 5}, {"ROOT GRASP", mvAll, 0.6, 3}, {"DRINK SAP", mvDrain, 0.9, 2}}
	case bossAsh:
		f.Name, f.Art, f.Theme = "HOODED ASH", imgBoss[0], thAsh
		f.MaxHP, f.Atk, f.Def, f.XP, f.Gold = 650, 30, 10, 150, 250
		f.Moves = []move{{"burns", mvHit, 1, 4}, {"ASHFIRE", mvAll, 0.7, 3}, {"EMBER CURSE", mvDrain, 1, 2}}
	case bossGormoth:
		f.Name, f.Art, f.Theme = "MAD GORMOTH", imgBoss[1], thAsh
		f.MaxHP, f.Atk, f.Def, f.XP, f.Gold = 850, 35, 13, 200, 300
		f.Moves = []move{{"smashes", mvHit, 1.1, 4}, {"MADNESS", mvTwice, 0.8, 3}, {"BONE QUAKE", mvAll, 0.6, 2}}
	case bossVyrelle:
		f.Name, f.Art, f.Theme = "WITCH VYRELLE", imgBoss[2], thForest
		f.MaxHP, f.Atk, f.Def, f.XP, f.Gold = 700, 31, 11, 170, 260
		f.Moves = []move{{"slashes", mvHit, 1, 3}, {"HEX", mvDrain, 1.2, 3}, {"DARK PACT", mvHeal, 0.18, 1}, {"PINK FLAME", mvAll, 0.65, 2}}
	case bossKthulgor:
		f.Name, f.Art, f.Theme = "KTHULGOR", imgBoss[3], thDemon
		f.MaxHP, f.Atk, f.Def, f.XP, f.Gold = 1000, 38, 14, 240, 350
		f.Moves = []move{{"TENTACLES", mvTwice, 0.85, 4}, {"DEEP SCREAM", mvAll, 0.7, 3}, {"DEVOUR", mvDrain, 1.1, 2}}
	case bossLord:
		f.Name, f.Art, f.Theme = "BRAZGANT", imgBrazgant, thDemon
		f.MaxHP, f.Atk, f.Def = 1900, 46, 17
		f.Moves = []move{{"cleaves", mvHit, 1.1, 4}, {"HELLFIRE", mvAll, 0.75, 3}, {"SOULREND", mvDrain, 1.2, 2}, {"DOOM CLAWS", mvTwice, 0.85, 2}, {"UNHOLY MEND", mvHeal, 0.08, 1}}
	}
	f.HP = f.MaxHP
	return f
}

// ---- Game over: the Brazgant screen ----

func (g *Game) goOver() {
	g.scene = sOver
	play("lose")
}

func (g *Game) updateOver() {
	switch {
	case pressed(ebiten.KeyY):
		g.goTitle()
	case pressed(ebiten.KeyN):
		g.st.restore()
		g.st.Gold /= 2
		g.st.Day++
		g.log = nil
		g.say("You wake at the inn. Half your gold is gone. Try again.")
		g.enter(sTown)
	}
}

func (g *Game) drawOver(dst *ebiten.Image) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(W/float64(imgOver.Bounds().Dx()), H/float64(imgOver.Bounds().Dy()))
	dst.DrawImage(imgOver, op)
	fill(dst, 96, 326, 296, 32, black)
	drawCentered(dst, "Stop trying??", W/2, 322, 2, thDemon.frame)
	blink := thDemon.hot
	if g.tick/20%2 == 0 {
		blink = thDemon.accent
	}
	drawCentered(dst, "<Y>es to quit  <N>o to rise", W/2, 346, 1, blink)
}

// ---- Victory ----

func (g *Game) goWin() {
	g.scene = sWin
	g.st.Slain[bossLord] = true
	g.say("BRAZGANT the Demon Lord falls. His sword shatters.")
	g.say("The dawn comes to Darkzant Town for the first time in an age.")
	g.say(fmt.Sprintf("You won on day %d at level %d. Press ENTER.", g.st.Day, g.st.Level))
	play("win")
}

func (g *Game) updateWin() {
	if confirm() {
		g.goTitle()
	}
}

func (g *Game) drawWin(dst *ebiten.Image) {
	t := float32(0.6 + 0.4*math.Sin(float64(g.tick)/30))
	drawScene(dst, imgTownFree, 1)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(sceneW/float64(imgTownFree.Bounds().Dx()), sceneH/float64(imgTownFree.Bounds().Dy()))
	op.GeoM.Translate(sceneX, sceneY)
	op.ColorScale.Scale(0.25*t, 0.2*t, 0.05*t, 1)
	op.Blend = ebiten.BlendLighter
	dst.DrawImage(imgTownFree, op)
	g.hud(dst, thTown, "DARKZANT IS FREE", -1, -1, [3]int{})
	g.bottom(dst, thTown, nil, "THE END")
}
