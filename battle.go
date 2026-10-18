package main

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

// Foe move kinds.
const (
	mvHit   = iota // one hero
	mvAll          // every hero
	mvTwice        // two blows on random heroes
	mvDrain        // one hero, heals the foe by the damage
	mvHeal         // foe heals Power x its max HP
)

type move struct {
	Name   string
	Kind   int
	Power  float64
	Weight int
}

type Foe struct {
	Name      string
	Art, BG   *ebiten.Image
	Tint      [3]float32
	HP, MaxHP int
	Atk, Def  int
	XP, Gold  int
	Boss      int // index into State.Slain, or -1
	Moves     []move
	Theme     theme
}

// Phases of the player's turn.
const (
	bpCommand = iota
	bpSkill
	bpItem
	bpTarget
	bpBusy
)

type battle struct {
	foe   *Foe
	th    theme
	phase int
	actor int
	cmd   menu
	sub   menu
	tgt   menu
	onTgt func(target int) // runs once a target hero is chosen

	queue []func() string // each event applies its effect and returns its line
	msgT  int
	after func() // runs when the queue is empty

	hitT, shake, dmgT int
	dmgText           string
	heroFlash         [3]int
	dead              bool
	deadT             int
}

func (g *Game) startBattle(f *Foe, back scene) {
	b := &battle{foe: f, th: f.Theme, phase: bpBusy, cmd: newMenu("ATTACK", "SKILL", "ITEM", "GUARD", "FLEE")}
	g.battle, g.back, g.scene = b, back, sBattle
	if f.Boss >= 0 {
		play("boss")
		b.cmd.off[4] = true
	}
	b.push(func() string { return fmt.Sprintf("%s appears!", f.Name) })
	b.after = func() { b.nextHero(g, -1) }
}

func (b *battle) push(ev ...func() string) { b.queue = append(b.queue, ev...) }

func (b *battle) update(g *Game) {
	for i := range b.heroFlash {
		b.heroFlash[i] = max(0, b.heroFlash[i]-1)
	}
	b.hitT, b.shake, b.dmgT = max(0, b.hitT-1), max(0, b.shake-1), max(0, b.dmgT-1)
	if b.dead {
		b.deadT++
	}

	// Play out queued events one line at a time.
	if b.msgT > 0 {
		b.msgT--
		if confirm() {
			b.msgT = 0
		}
		return
	}
	if len(b.queue) > 0 {
		ev := b.queue[0]
		b.queue = b.queue[1:]
		if msg := ev(); msg != "" {
			g.say(msg)
			b.msgT = 55
		}
		return
	}
	if b.after != nil {
		f := b.after
		b.after = nil
		f()
		return
	}

	h := &g.st.Heroes[b.actor]
	switch b.phase {
	case bpCommand:
		switch b.cmd.update() {
		case 0:
			b.act(g, func() string { return b.heroHit(g, h, 1, "attacks") })
		case 1:
			b.sub = newMenu()
			for _, sk := range skills[h.Name] {
				if sk.Level <= g.st.Level {
					b.sub.items = append(b.sub.items, fmt.Sprintf("%-8s %dMP", sk.Name, sk.MP))
					b.sub.off = append(b.sub.off, sk.MP > h.MP)
				}
			}
			b.phase = bpSkill
		case 2:
			b.sub = newMenu()
			for i, n := range itemNames {
				b.sub.items = append(b.sub.items, fmt.Sprintf("%-8s x%d", n, g.st.Items[i]))
				b.sub.off = append(b.sub.off, g.st.Items[i] == 0)
			}
			b.phase = bpItem
		case 3:
			b.act(g, func() string {
				h.Guard = true
				return h.Name + " raises a guard."
			})
		case 4:
			b.act(g, func() string {
				if chance(0.55) {
					b.queue = nil
					b.after = func() { g.endBattle() }
					return "The party flees into the dark!"
				}
				return "No escape!"
			})
		}
	case bpSkill:
		switch i := b.sub.update(); {
		case i == -2:
			b.phase = bpCommand
		case i >= 0:
			b.useSkill(g, h, skills[h.Name][i])
		}
	case bpItem:
		switch i := b.sub.update(); {
		case i == -2:
			b.phase = bpCommand
		case i >= 0:
			b.useItem(g, h, i)
		}
	case bpTarget:
		switch i := b.tgt.update(); {
		case i == -2:
			b.phase = bpCommand
		case i >= 0:
			b.onTgt(i)
		}
	}
}

// act queues the hero's action, then moves on to whoever is next.
func (b *battle) act(g *Game, ev func() string) {
	b.phase = bpBusy
	b.push(ev)
	b.after = func() { b.nextHero(g, b.actor) }
}

func (b *battle) chooseTarget(g *Game, then func(t int)) {
	b.tgt = newMenu()
	for _, h := range g.st.Heroes {
		b.tgt.items = append(b.tgt.items, h.Name)
		b.tgt.off = append(b.tgt.off, false)
	}
	b.tgt.sel = b.actor
	b.onTgt = then
	b.phase = bpTarget
}

func (b *battle) useSkill(g *Game, h *Hero, sk Skill) {
	lv := float64(g.st.Level)
	amount := func() int { return int((sk.Power + sk.Grow*lv) * (0.9 + rand.Float64()*0.2)) }
	switch sk.Kind {
	case skHit:
		b.act(g, func() string {
			h.MP -= sk.MP
			return b.heroHit(g, h, sk.Power, "uses "+sk.Name)
		})
	case skMagic:
		b.act(g, func() string {
			h.MP -= sk.MP
			play("magic")
			n := amount() + weapons[g.st.Weapon].Bonus/2
			b.damageFoe(n)
			return fmt.Sprintf("%s casts %s! %d damage.", h.Name, sk.Name, n)
		})
	case skHeal:
		b.chooseTarget(g, func(t int) {
			b.act(g, func() string {
				h.MP -= sk.MP
				return b.heal(g, t, amount(), h.Name+" casts "+sk.Name+".")
			})
		})
	case skHealAll:
		b.act(g, func() string {
			h.MP -= sk.MP
			play("heal")
			n := amount()
			for i := range g.st.Heroes {
				o := &g.st.Heroes[i]
				if o.HP > 0 {
					o.HP = min(o.MaxHP, o.HP+n)
				}
			}
			return fmt.Sprintf("%s casts %s. All heal %d.", h.Name, sk.Name, n)
		})
	}
}

func (b *battle) useItem(g *Game, h *Hero, i int) {
	switch i {
	case 0:
		b.chooseTarget(g, func(t int) {
			b.act(g, func() string {
				g.st.Items[0]--
				return b.heal(g, t, potionHeal, h.Name+" uses a POTION.")
			})
		})
	case 1:
		b.chooseTarget(g, func(t int) {
			b.act(g, func() string {
				g.st.Items[1]--
				o := &g.st.Heroes[t]
				o.MP = min(o.MaxMP, o.MP+etherMP)
				play("heal")
				return fmt.Sprintf("%s uses an ETHER. %s regains MP.", h.Name, o.Name)
			})
		})
	case 2:
		b.act(g, func() string {
			g.st.Items[2]--
			g.st.restore()
			play("heal")
			return h.Name + " opens an ELIXIR. All are restored!"
		})
	}
}

func (b *battle) heal(g *Game, t, n int, prefix string) string {
	o := &g.st.Heroes[t]
	play("heal")
	if o.HP <= 0 {
		o.HP = min(o.MaxHP, n)
		return fmt.Sprintf("%s %s rises again!", prefix, o.Name)
	}
	o.HP = min(o.MaxHP, o.HP+n)
	return fmt.Sprintf("%s %s heals %d.", prefix, o.Name, n)
}

// heroHit resolves a physical blow on the foe.
func (b *battle) heroHit(g *Game, h *Hero, mult float64, verb string) string {
	atk := float64(h.Atk+weapons[g.st.Weapon].Bonus) * mult
	n := max(1, int(atk*(0.85+rand.Float64()*0.3))-b.foe.Def/2)
	crit := chance(h.Crit)
	if crit {
		n = n * 3 / 2
	}
	b.damageFoe(n)
	play("hit")
	if crit {
		return fmt.Sprintf("%s %s! CRITICAL! %d damage.", h.Name, verb, n)
	}
	return fmt.Sprintf("%s %s. %d damage.", h.Name, verb, n)
}

func (b *battle) damageFoe(n int) {
	b.foe.HP = max(0, b.foe.HP-n)
	b.hitT, b.dmgT, b.dmgText = 14, 40, fmt.Sprint(n)
}

// nextHero gives the turn to the next standing hero after from, or to the foe.
func (b *battle) nextHero(g *Game, from int) {
	if b.foe.HP <= 0 {
		b.victory(g)
		return
	}
	for i := from + 1; i < len(g.st.Heroes); i++ {
		if g.st.Heroes[i].HP > 0 {
			b.actor, b.phase = i, bpCommand
			b.cmd.sel = 0
			return
		}
	}
	b.foeTurn(g)
}

func (b *battle) foeTurn(g *Game) {
	b.phase = bpBusy
	f := b.foe
	total := 0
	for _, m := range f.Moves {
		total += m.Weight
	}
	r := rand.IntN(total)
	var m move
	for _, m = range f.Moves {
		if r < m.Weight {
			break
		}
		r -= m.Weight
	}
	if m.Kind == mvHeal && f.HP > f.MaxHP*3/4 {
		m = move{Name: "strikes", Kind: mvHit, Power: 1}
	}

	switch m.Kind {
	case mvHit, mvDrain:
		b.push(func() string {
			t := randomHero(g)
			n := b.hurt(g, t, m.Power)
			if m.Kind == mvDrain {
				f.HP = min(f.MaxHP, f.HP+n)
				return fmt.Sprintf("%s uses %s on %s. %d damage. It drinks the life!", f.Name, m.Name, g.st.Heroes[t].Name, n)
			}
			return fmt.Sprintf("%s %s %s. %d damage.", f.Name, m.Name, g.st.Heroes[t].Name, n)
		})
	case mvTwice:
		b.push(func() string { return fmt.Sprintf("%s uses %s!", f.Name, m.Name) })
		for range 2 {
			b.push(func() string {
				if !g.st.alive() {
					return ""
				}
				t := randomHero(g)
				return fmt.Sprintf("%s takes %d damage.", g.st.Heroes[t].Name, b.hurt(g, t, m.Power))
			})
		}
	case mvAll:
		b.push(func() string {
			total := 0
			for i := range g.st.Heroes {
				if g.st.Heroes[i].HP > 0 {
					total += b.hurt(g, i, m.Power)
				}
			}
			return fmt.Sprintf("%s unleashes %s! %d damage in all.", f.Name, m.Name, total)
		})
	case mvHeal:
		b.push(func() string {
			n := int(float64(f.MaxHP) * m.Power)
			f.HP = min(f.MaxHP, f.HP+n)
			play("heal")
			return fmt.Sprintf("%s uses %s and heals %d.", f.Name, m.Name, n)
		})
	}
	b.after = func() {
		if !g.st.alive() {
			b.push(func() string { return "The party has fallen..." })
			b.after = func() { g.goOver() }
			return
		}
		for i := range g.st.Heroes {
			g.st.Heroes[i].Guard = false
		}
		b.nextHero(g, -1)
	}
}

// hurt deals foe damage to hero t and returns the amount.
func (b *battle) hurt(g *Game, t int, power float64) int {
	h := &g.st.Heroes[t]
	atk := float64(b.foe.Atk) * power * (0.85 + rand.Float64()*0.3)
	n := max(1, int(atk)-(h.Def+armours[g.st.Armour].Bonus)/2)
	if h.Guard {
		n = max(1, n/2)
	}
	h.HP = max(0, h.HP-n)
	b.heroFlash[t], b.shake = 18, 10
	play("hurt")
	return n
}

func randomHero(g *Game) int {
	var up []int
	for i, h := range g.st.Heroes {
		if h.HP > 0 {
			up = append(up, i)
		}
	}
	return up[rand.IntN(len(up))]
}

func (b *battle) victory(g *Game) {
	f := b.foe
	b.dead = true
	play("win")
	b.push(func() string { return fmt.Sprintf("%s is destroyed!", f.Name) })
	b.push(func() string {
		g.st.Gold += f.Gold
		return fmt.Sprintf("Won %d XP and %d gold.", f.XP, f.Gold)
	})
	b.push(func() string {
		for _, m := range g.st.gainXP(f.XP) {
			g.say(m)
		}
		return ""
	})
	b.after = func() {
		if f.Boss >= 0 {
			g.st.Slain[f.Boss] = true
		}
		if f.Boss == bossLord {
			g.goWin()
			return
		}
		g.endBattle()
	}
}

func (g *Game) endBattle() {
	for i := range g.st.Heroes {
		g.st.Heroes[i].Guard = false
	}
	g.enter(g.back)
}

func (b *battle) draw(g *Game, dst *ebiten.Image) {
	f := b.foe
	drawScene(dst, f.BG, 0.3)

	// The foe: scaled to fit, bobbing, flashing when struck.
	w, h := float64(f.Art.Bounds().Dx()), float64(f.Art.Bounds().Dy())
	s := min(300/w, 186/h, 3.0)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(s, s)
	bob := math.Sin(float64(g.tick)/22) * 2
	x := sceneX + (sceneW-w*s)/2
	y := sceneY + 8 + (sceneH-8-h*s)/2 + bob
	if b.hitT > 0 {
		x += float64(rand.IntN(5) - 2)
	}
	op.GeoM.Translate(math.Round(x), math.Round(y))
	op.ColorScale.Scale(f.Tint[0], f.Tint[1], f.Tint[2], 1)
	if b.hitT > 0 && b.hitT/2%2 == 0 {
		op.ColorScale.Scale(3, 3, 3, 1)
	}
	if b.dead {
		op.ColorScale.ScaleAlpha(1 - float32(min(b.deadT, 60))/60*0.9)
	}
	dst.DrawImage(f.Art, op)

	// Foe life bar along the top of the scene.
	fill(dst, sceneX, sceneY, sceneW, 9, black)
	bar(dst, sceneX+4, sceneY+2, 250, 5, float64(f.HP)/float64(f.MaxHP), b.th.accent, b.th.dim)
	drawText(dst, fmt.Sprint(f.HP), sceneX+262, sceneY-3, b.th.hot)

	if b.dmgT > 0 {
		drawCentered(dst, b.dmgText, W/2, sceneY+70-float64(40-b.dmgT)/2, 2, b.th.hot)
	}

	target := -1
	if b.phase == bpTarget {
		target = b.tgt.sel
	}
	active := -1
	if b.phase != bpBusy && len(b.queue) == 0 && b.msgT == 0 {
		active = b.actor
	}
	g.hud(dst, b.th, f.Name, active, target, b.heroFlash)

	switch {
	case active < 0:
		g.bottom(dst, b.th, nil, "")
	case b.phase == bpCommand:
		g.bottom(dst, b.th, &b.cmd, g.st.Heroes[b.actor].Name+":")
	case b.phase == bpTarget:
		g.bottom(dst, b.th, &b.tgt, "ON WHOM?")
	default:
		g.bottom(dst, b.th, &b.sub, g.st.Heroes[b.actor].Name+":")
	}
}
