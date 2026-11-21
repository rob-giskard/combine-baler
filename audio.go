package main

// Tiny software synth: cosy generative music, engine hum, reaper swish and
// one-shot effects. Platform independent; the platform layer pulls samples
// with Fill (mono, 16 bit).

import (
	"math"
	"sync"
)

const SampleRate = 22050

const (
	SMenu = iota
	SSelect
	SReapOn
	SReapOff
	SBale
	SPick
	SDrop
	SDoor
	SBump
	SStep
	SDayEnd
	SCoin
	SSip
	SFizz
	SBoing
	SNope
	SYawn
	SBoost
	SSpray
	SSnap
	SSparkle
	SHornCombine
	SHornTruck
)

type voice struct {
	f0, f1, dur, t, vol, noise, ph float64
	horn                           bool // held, reedy tone instead of a ping
}

type pluck struct {
	ph, inc, amp, dec, h2 float64
}

type chord struct {
	bass int
	arp  [4]int
}

var prog = [8]chord{
	{36, [4]int{60, 64, 67, 72}}, // C
	{33, [4]int{57, 60, 64, 69}}, // Am
	{41, [4]int{57, 60, 65, 69}}, // F
	{43, [4]int{55, 59, 62, 67}}, // G
	{36, [4]int{60, 64, 67, 72}}, // C
	{40, [4]int{55, 59, 64, 67}}, // Em
	{41, [4]int{57, 60, 65, 69}}, // F
	{43, [4]int{55, 59, 62, 67}}, // G
}
var arpPat = [8]int{0, 2, 1, 2, 3, 2, 1, 2}
var penta = [6]int{72, 74, 76, 79, 81, 84}

type Audio struct {
	mu      sync.Mutex
	MusicOn bool
	voices  []voice
	plucks  [24]pluck
	np      int

	step    int
	stepPos float64
	padInc  [3]float64
	padPh   [3]float64
	melIdx  int

	cOn, cVol, cSpdT, cSpd, cPh float64
	tOn, tVol, tSpdT, tSpd, tPh float64
	reapT, reapVol, lp1, lp2    float64
	lfo                         float64
	nz                          uint32
}

func NewAudio() *Audio { return &Audio{MusicOn: true, nz: 0x1234567, melIdx: 2} }

func midiHz(m int) float64 { return 440 * math.Pow(2, float64(m-69)/12) }

func (a *Audio) noiseS() float64 {
	a.nz ^= a.nz << 13
	a.nz ^= a.nz >> 17
	a.nz ^= a.nz << 5
	return float64(a.nz&0xFFFF)/32768 - 1
}

func (a *Audio) addPluck(m int, amp, tau, h2 float64) {
	a.plucks[a.np] = pluck{inc: 2 * math.Pi * midiHz(m) / SampleRate, amp: amp,
		dec: math.Exp(-1 / (SampleRate * tau)), h2: h2}
	a.np = (a.np + 1) % len(a.plucks)
}

// SetState is called once per game frame with what the machines are doing.
func (a *Audio) SetState(combineOn bool, combineSpd float64, truckOn bool, truckSpd float64, reaping bool) {
	a.mu.Lock()
	b := func(v bool) float64 {
		if v {
			return 1
		}
		return 0
	}
	a.cOn, a.cSpdT, a.tOn, a.tSpdT, a.reapT = b(combineOn), combineSpd, b(truckOn), truckSpd, b(reaping)
	a.mu.Unlock()
}

func (a *Audio) ToggleMusic() {
	a.mu.Lock()
	a.MusicOn = !a.MusicOn
	a.mu.Unlock()
}

func (a *Audio) Play(id int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.voices) > 40 {
		return
	}
	v := func(f0, f1, dur, vol, noise, delay float64) {
		a.voices = append(a.voices, voice{f0: f0, f1: f1, dur: dur, vol: vol, noise: noise, t: -delay})
	}
	switch id {
	case SMenu:
		v(660, 660, 0.06, 0.10, 0, 0)
	case SSelect:
		v(523, 523, 0.10, 0.12, 0, 0)
		v(784, 784, 0.18, 0.12, 0, 0.09)
	case SReapOn:
		v(170, 400, 0.28, 0.09, 0.1, 0)
	case SReapOff:
		v(400, 150, 0.28, 0.09, 0.1, 0)
	case SBale:
		v(120, 55, 0.20, 0.26, 0.15, 0)
	case SPick:
		v(520, 830, 0.09, 0.12, 0, 0)
	case SDrop:
		v(300, 210, 0.08, 0.16, 0.3, 0)
	case SDoor:
		v(900, 900, 0.03, 0.08, 1, 0)
		v(150, 85, 0.09, 0.16, 0.1, 0.03)
	case SBump:
		v(95, 45, 0.22, 0.28, 0.3, 0)
	case SStep:
		v(200, 200, 0.03, 0.03, 1, 0)
	case SCoin:
		for i, f := range []float64{1318.5, 1568, 2093} {
			v(f, f, 0.25, 0.09, 0, float64(i)*0.07)
		}
	case SSip:
		v(260, 200, 0.09, 0.10, 0.2, 0)
		v(240, 180, 0.09, 0.10, 0.2, 0.16)
	case SFizz:
		v(3000, 3000, 0.5, 0.05, 1, 0)
		v(900, 1400, 0.12, 0.06, 0, 0.05)
	case SBoing:
		v(220, 440, 0.12, 0.12, 0, 0)
	case SNope:
		v(160, 120, 0.14, 0.12, 0.1, 0)
	case SYawn:
		v(420, 240, 0.5, 0.035, 0, 0)
	case SBoost: // whoosh and a kick of the engine
		v(2400, 500, 0.45, 0.07, 1, 0)
		v(110, 220, 0.35, 0.16, 0.2, 0)
	case SSpray:
		v(5000, 4200, 0.3, 0.03, 1, 0)
	case SSnap:
		v(900, 200, 0.12, 0.10, 0.4, 0)
		v(300, 150, 0.1, 0.08, 0.1, 0.1)
	case SSparkle:
		for i, f := range []float64{1568, 2093, 2637, 3136} {
			v(f, f, 0.2, 0.05, 0, float64(i)*0.05)
		}
	case SHornCombine: // a low two-tone honk
		hv := func(f, vol, delay float64) {
			a.voices = append(a.voices, voice{f0: f, f1: f * 0.98, dur: 0.42, vol: vol, t: -delay, horn: true})
		}
		hv(196, 0.13, 0)
		hv(247, 0.10, 0)
	case SHornTruck: // a higher beep-beep
		for _, d := range []float64{0, 0.17} {
			a.voices = append(a.voices, voice{f0: 392, f1: 392, dur: 0.13, vol: 0.11, t: -d, horn: true},
				voice{f0: 494, f1: 494, dur: 0.13, vol: 0.08, t: -d, horn: true})
		}
	case SDayEnd:
		for i, f := range []float64{523.25, 659.25, 783.99, 1046.5} {
			v(f, f, 1.0, 0.13, 0, float64(i)*0.16)
		}
	}
}

func (a *Audio) musicStep() {
	c := prog[(a.step/8)%8]
	k := a.step % 8
	acc := 0.13
	if k == 0 {
		acc = 0.17
	}
	a.addPluck(c.arp[arpPat[k]], acc, 0.42, 0.25)
	if k == 0 {
		a.addPluck(c.bass, 0.24, 0.9, 0.12)
		for i := 0; i < 3; i++ {
			a.padInc[i] = 2 * math.Pi * midiHz(c.arp[i]-12) / SampleRate
		}
	} else if k == 4 {
		a.addPluck(c.bass+7, 0.13, 0.6, 0.12)
	}
	h := hash(a.step, 91)
	if k%2 == 0 && h%100 < 40 {
		a.melIdx += [6]int{-1, 1, 1, -1, 2, -2}[(h>>8)%6]
		if a.melIdx < 0 {
			a.melIdx = 1
		}
		if a.melIdx > 5 {
			a.melIdx = 4
		}
		a.addPluck(penta[a.melIdx], 0.085, 1.0, 0.08)
	}
	a.step++
}

const stepLen = SampleRate * 60.0 / 84 / 2 // one eighth note at 84 bpm

func (a *Audio) Fill(out []int16) {
	a.mu.Lock()
	defer a.mu.Unlock()
	const sm = 0.0006
	for i := range out {
		s := 0.0
		if a.MusicOn {
			if a.stepPos <= 0 {
				a.musicStep()
				a.stepPos += stepLen
			}
			a.stepPos--
			p := (float64((a.step-1)%8) + 1 - a.stepPos/stepLen) / 8
			pe := 0.03 * math.Sin(math.Pi*p)
			for j := range a.padPh {
				a.padPh[j] += a.padInc[j]
				s += pe * math.Sin(a.padPh[j])
			}
		}
		for j := range a.plucks {
			pl := &a.plucks[j]
			if pl.amp < 0.0004 {
				continue
			}
			pl.ph += pl.inc
			s += pl.amp * (math.Sin(pl.ph) + pl.h2*math.Sin(2*pl.ph))
			pl.amp *= pl.dec
		}

		// engines
		a.lfo += 2 * math.Pi * 9 / SampleRate
		a.cVol += (a.cOn - a.cVol) * sm
		a.cSpd += (a.cSpdT - a.cSpd) * sm
		a.tVol += (a.tOn - a.tVol) * sm
		a.tSpd += (a.tSpdT - a.tSpd) * sm
		if a.cVol > 0.001 {
			a.cPh += 2 * math.Pi * (46 + 26*a.cSpd) / SampleRate
			s += a.cVol * (0.035 + 0.02*a.cSpd) * (0.85 + 0.15*math.Sin(a.lfo*1.3)) *
				(math.Sin(a.cPh) + 0.5*math.Sin(2*a.cPh) + 0.22*math.Sin(3*a.cPh))
		}
		if a.tVol > 0.001 {
			a.tPh += 2 * math.Pi * (68 + 52*a.tSpd) / SampleRate
			s += a.tVol * (0.03 + 0.02*a.tSpd) * (math.Sin(a.tPh) + 0.4*math.Sin(2*a.tPh))
		}
		// reaper: band limited noise pulsing with the reel
		a.reapVol += (a.reapT - a.reapVol) * sm
		if a.reapVol > 0.001 {
			n := a.noiseS()
			a.lp1 += (n - a.lp1) * 0.35
			a.lp2 += (n - a.lp2) * 0.04
			s += (a.lp1 - a.lp2) * a.reapVol * 0.09 * (0.7 + 0.3*math.Sin(a.lfo))
		}

		// one-shots
		n := 0
		for _, v := range a.voices {
			v.t += 1.0 / SampleRate
			if v.t >= v.dur {
				continue
			}
			if v.t >= 0 {
				fr := v.t / v.dur
				v.ph += 2 * math.Pi * (v.f0 + (v.f1-v.f0)*fr) / SampleRate
				env := (1 - fr) * (1 - fr) * math.Min(1, v.t/0.004)
				tone := math.Sin(v.ph)
				if v.horn {
					env = math.Min(1, v.t/0.015) * math.Min(1, (1-fr)/0.2)
					tone = 0.6*math.Sin(v.ph) + 0.3*math.Sin(2*v.ph) + 0.18*math.Sin(3*v.ph) + 0.1*math.Sin(4*v.ph)
				}
				s += env * v.vol * ((1-v.noise)*tone + v.noise*a.noiseS())
			}
			a.voices[n] = v
			n++
		}
		a.voices = a.voices[:n]

		s = s / (1 + math.Abs(s)) // soft limiter
		out[i] = int16(s * 30000)
	}
}
