package main

import "testing"

// TestPlaythrough lets a simple bot play the whole game headless:
// grind the wilds, shop, rest, and take on each boss when strong enough.
func TestPlaythrough(t *testing.T) {
	g := &Game{st: newState()}
	g.enter(sTown)
	wins := map[int]int{}
	order := []int{bossPazuzu, bossAsh, bossVyrelle, bossGormoth, bossKthulgor, bossLord}
	ready := map[int]int{bossPazuzu: 2, bossAsh: 4, bossVyrelle: 4, bossGormoth: 5, bossKthulgor: 6, bossLord: 7}
	deaths, fights := 0, 0
	for _, boss := range order {
		for !g.st.Slain[boss] {
			if fights > 3000 {
				t.Fatalf("stuck before boss %d at level %d", boss, g.st.Level)
			}
			g.shop()
			g.st.restore()
			var f *Foe
			if g.st.Level >= ready[boss] {
				f = bossFoe(boss)
			} else {
				f = g.wildFoe()
			}
			fights++
			g.startBattle(f, sTown)
			if !g.runBattle() {
				deaths++
				g.st.restore()
				g.st.Gold /= 2
				if f.Boss >= 0 {
					ready[boss]++
				}
				continue
			}
			if f.Boss >= 0 {
				wins[boss] = g.st.Level
			}
		}
	}
	t.Logf("fights %d, deaths %d, boss levels %v, gold %d, gear %d/%d", fights, deaths, wins, g.st.Gold, g.st.Weapon, g.st.Armour)
}

func (g *Game) shop() {
	for {
		w, a := g.st.Weapon+1, g.st.Armour+1
		switch {
		case w < len(weapons) && g.st.Gold >= weapons[w].Price:
			g.st.Gold -= weapons[w].Price
			g.st.Weapon = w
		case a < len(armours) && g.st.Gold >= armours[a].Price:
			g.st.Gold -= armours[a].Price
			g.st.Armour = a
		case g.st.Items[0] < 4 && g.st.Gold >= itemPrices[0]+100:
			g.st.Gold -= itemPrices[0]
			g.st.Items[0]++
		default:
			return
		}
	}
}

// runBattle drives the current battle to its end and reports a win.
func (g *Game) runBattle() bool {
	b := g.battle
	for range 200000 {
		if g.scene != sBattle {
			return g.scene != sOver
		}
		b.msgT = 0
		if b.phase == bpCommand && len(b.queue) == 0 && b.after == nil {
			g.botTurn(b)
			continue
		}
		b.update(g)
	}
	panic("battle never ended")
}

func (g *Game) botTurn(b *battle) {
	h := &g.st.Heroes[b.actor]
	low := -1
	for i, o := range g.st.Heroes {
		if float64(o.HP) < float64(o.MaxHP)*0.4 && (low < 0 || o.HP < g.st.Heroes[low].HP) {
			low = i
		}
	}
	var best *Skill
	for _, sk := range skills[h.Name] {
		if sk.Level <= g.st.Level && sk.MP <= h.MP && (sk.Kind == skHit || sk.Kind == skMagic) {
			best = &sk
		}
	}
	switch {
	case low >= 0 && h.Name == "LORIE" && h.MP >= 4:
		b.useSkill(g, h, skills["LORIE"][0])
		b.onTgt(low)
	case low >= 0 && g.st.Items[0] > 0:
		b.useItem(g, h, 0)
		b.onTgt(low)
	case best != nil:
		b.useSkill(g, h, *best)
	default:
		b.act(g, func() string { return b.heroHit(g, h, 1, "attacks") })
	}
}
