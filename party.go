package main

import (
	"encoding/json"
	"fmt"
)

// Skill kinds.
const (
	skHit     = iota // physical blow, Power is a multiplier on attack
	skMagic          // magic damage, ignores defence
	skHeal           // heal or revive one hero
	skHealAll        // heal every hero still standing
)

type Skill struct {
	Name  string
	MP    int
	Kind  int
	Power float64 // multiplier for skHit, base amount otherwise
	Grow  float64 // amount added per party level
	Level int     // party level that unlocks it
}

type Hero struct {
	Name, Class      string
	HP, MaxHP        int
	MP, MaxMP        int
	Atk, Def         int
	Crit             float64
	GrowHP, GrowMP   int
	GrowAtk, GrowDef int
	Guard            bool `json:"-"`
}

// Gear is shared by the party: one weapon tier and one armour tier.
type Gear struct {
	Name  string
	Bonus int
	Price int
}

var weapons = []Gear{{"BARE STEEL", 0, 0}, {"IRON BLADES", 3, 60}, {"RUNE BLADES", 7, 220}, {"DEMONSBANE", 12, 650}}
var armours = []Gear{{"RAGS", 0, 0}, {"CHAIN MAIL", 2, 50}, {"BLACK PLATE", 5, 200}, {"ASH WARDS", 9, 600}}

// Items: potion heals one, ether restores MP to one, elixir restores everyone.
var itemNames = []string{"POTION", "ETHER", "ELIXIR"}
var itemPrices = []int{12, 30, 150}

const (
	potionHeal = 45
	etherMP    = 15
)

var skills = map[string][]Skill{
	"BASL": {
		{Name: "CLEAVE", MP: 3, Kind: skHit, Power: 1.8, Level: 1},
		{Name: "SUNDER", MP: 7, Kind: skHit, Power: 3.0, Level: 5},
	},
	"MAKAND": {
		{Name: "FIRE", MP: 4, Kind: skMagic, Power: 16, Grow: 3, Level: 1},
		{Name: "BOLT", MP: 9, Kind: skMagic, Power: 32, Grow: 5, Level: 4},
		{Name: "STARFALL", MP: 16, Kind: skMagic, Power: 60, Grow: 8, Level: 8},
	},
	"LORIE": {
		{Name: "MEND", MP: 4, Kind: skHeal, Power: 28, Grow: 4, Level: 1},
		{Name: "BLESS", MP: 10, Kind: skHealAll, Power: 22, Grow: 3, Level: 3},
	},
}

// Boss slots in State.Slain.
const (
	bossPazuzu = iota
	bossAsh
	bossGormoth
	bossVyrelle
	bossKthulgor
	bossLord
)

// State is everything saved to disk.
type State struct {
	Heroes []Hero
	Level  int
	XP     int
	Gold   int
	Weapon int
	Armour int
	Items  [3]int
	Slain  [6]bool
	Day    int
}

func newState() State {
	return State{
		Heroes: []Hero{
			{Name: "BASL", Class: "WARRIOR", HP: 70, MaxHP: 70, MP: 8, MaxMP: 8, Atk: 11, Def: 6, Crit: 0.08, GrowHP: 10, GrowMP: 1, GrowAtk: 2, GrowDef: 1},
			{Name: "MAKAND", Class: "WIZARD", HP: 42, MaxHP: 42, MP: 30, MaxMP: 30, Atk: 4, Def: 2, Crit: 0.05, GrowHP: 6, GrowMP: 4, GrowAtk: 1, GrowDef: 1},
			{Name: "LORIE", Class: "ROGUE", HP: 52, MaxHP: 52, MP: 20, MaxMP: 20, Atk: 8, Def: 4, Crit: 0.25, GrowHP: 8, GrowMP: 3, GrowAtk: 2, GrowDef: 1},
		},
		Level: 1,
		Gold:  40,
		Items: [3]int{3, 1, 0},
		Day:   1,
	}
}

func xpNeed(level int) int { return 10 * level * (level + 1) }

// gainXP adds experience and returns one message per level gained.
func (s *State) gainXP(n int) []string {
	s.XP += n
	var msgs []string
	for s.XP >= xpNeed(s.Level) {
		s.XP -= xpNeed(s.Level)
		s.Level++
		for i := range s.Heroes {
			h := &s.Heroes[i]
			h.MaxHP += h.GrowHP
			h.MaxMP += h.GrowMP
			h.Atk += h.GrowAtk
			h.Def += h.GrowDef
		}
		s.restore()
		msgs = append(msgs, fmt.Sprintf("LEVEL %d! The party grows stronger.", s.Level))
		for name, list := range skills {
			for _, sk := range list {
				if sk.Level == s.Level {
					msgs = append(msgs, fmt.Sprintf("%s learns %s.", name, sk.Name))
				}
			}
		}
	}
	return msgs
}

// restore heals and revives everyone.
func (s *State) restore() {
	for i := range s.Heroes {
		h := &s.Heroes[i]
		h.HP, h.MP = h.MaxHP, h.MaxMP
	}
}

func (s *State) alive() bool {
	for _, h := range s.Heroes {
		if h.HP > 0 {
			return true
		}
	}
	return false
}

func (s *State) bossesDown() int {
	n := 0
	for _, d := range s.Slain[bossAsh : bossKthulgor+1] {
		if d {
			n++
		}
	}
	return n
}

func (s *State) save() error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return writeSave(b)
}

func loadState() (State, bool) {
	var s State
	b, err := readSave()
	if err != nil || json.Unmarshal(b, &s) != nil || len(s.Heroes) != 3 {
		return s, false
	}
	return s, true
}
