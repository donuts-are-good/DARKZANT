package main

import (
	"encoding/binary"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const rate = 44100

var (
	actx *audio.Context
	sfx  = map[string][]byte{}
)

// note is one step of a beeper sound: a square wave or noise burst.
type note struct {
	hz    float64 // 0 means noise
	sec   float64
	slide float64 // hz change per second
}

func initSound() {
	actx = audio.NewContext(rate)
	sfx["move"] = synth(0.10, note{hz: 990, sec: 0.025})
	sfx["ok"] = synth(0.12, note{hz: 660, sec: 0.04}, note{hz: 990, sec: 0.06})
	sfx["no"] = synth(0.12, note{hz: 180, sec: 0.12})
	sfx["hit"] = synth(0.18, note{sec: 0.09}, note{hz: 140, sec: 0.05, slide: -600})
	sfx["hurt"] = synth(0.20, note{hz: 220, sec: 0.16, slide: -900}, note{sec: 0.06})
	sfx["magic"] = synth(0.12, note{hz: 300, sec: 0.28, slide: 3200})
	sfx["heal"] = synth(0.12, note{hz: 523, sec: 0.06}, note{hz: 659, sec: 0.06}, note{hz: 784, sec: 0.06}, note{hz: 1046, sec: 0.12})
	sfx["coin"] = synth(0.10, note{hz: 988, sec: 0.05}, note{hz: 1319, sec: 0.14})
	sfx["win"] = synth(0.12, note{hz: 392, sec: 0.1}, note{hz: 523, sec: 0.1}, note{hz: 659, sec: 0.1}, note{hz: 784, sec: 0.3})
	sfx["lose"] = synth(0.14, note{hz: 392, sec: 0.2}, note{hz: 311, sec: 0.2}, note{hz: 262, sec: 0.2}, note{hz: 196, sec: 0.6, slide: -60})
	sfx["boss"] = synth(0.16, note{hz: 110, sec: 0.3, slide: -40}, note{hz: 104, sec: 0.3, slide: -40}, note{hz: 98, sec: 0.6, slide: -30})
}

func play(name string) {
	if b, ok := sfx[name]; ok && actx != nil {
		actx.NewPlayerFromBytes(b).Play()
	}
}

// synth renders notes to 16-bit stereo PCM, the format the audio context wants.
func synth(vol float64, notes ...note) []byte {
	var out []byte
	buf := make([]byte, 4)
	for _, n := range notes {
		count := int(n.sec * rate)
		phase := 0.0
		for i := range count {
			t := float64(i) / rate
			var v float64
			if n.hz == 0 {
				v = rand.Float64()*2 - 1
			} else {
				phase += (n.hz + n.slide*t) / rate
				v = 1
				if math.Mod(phase, 1) > 0.5 {
					v = -1
				}
			}
			env := 1 - float64(i)/float64(count)*0.7
			s := int16(v * vol * env * math.MaxInt16)
			binary.LittleEndian.PutUint16(buf[0:], uint16(s))
			binary.LittleEndian.PutUint16(buf[2:], uint16(s))
			out = append(out, buf...)
		}
	}
	return out
}
