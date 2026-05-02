package dsp

import (
	"math"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

type BiquadType int

const (
	BiquadLowpass BiquadType = iota
	BiquadHighpass
	BiquadBandpass
	BiquadNotch
	BiquadPeaking
	BiquadLowShelf
	BiquadHighShelf
)

type Biquad struct {
	b0, b1, b2, a1, a2 float64
	x1L, x2L, y1L, y2L float64
	x1R, x2R, y1R, y2R float64
}

func NewLowpass(freqHz, q float64, sampleRate int) *Biquad {
	return newBiquad(BiquadLowpass, freqHz, q, 0, sampleRate)
}

func NewHighpass(freqHz, q float64, sampleRate int) *Biquad {
	return newBiquad(BiquadHighpass, freqHz, q, 0, sampleRate)
}

func NewBandpass(freqHz, q float64, sampleRate int) *Biquad {
	return newBiquad(BiquadBandpass, freqHz, q, 0, sampleRate)
}

func NewNotch(freqHz, q float64, sampleRate int) *Biquad {
	return newBiquad(BiquadNotch, freqHz, q, 0, sampleRate)
}

func NewPeaking(freqHz, q, dBGain float64, sampleRate int) *Biquad {
	return newBiquad(BiquadPeaking, freqHz, q, dBGain, sampleRate)
}

func NewLowShelf(freqHz, q, dBGain float64, sampleRate int) *Biquad {
	return newBiquad(BiquadLowShelf, freqHz, q, dBGain, sampleRate)
}

func NewHighShelf(freqHz, q, dBGain float64, sampleRate int) *Biquad {
	return newBiquad(BiquadHighShelf, freqHz, q, dBGain, sampleRate)
}

func newBiquad(typ BiquadType, freqHz, q, dBGain float64, sampleRate int) *Biquad {
	omega := 2 * math.Pi * freqHz / float64(sampleRate)
	sin := math.Sin(omega)
	cos := math.Cos(omega)

	switch typ {
	case BiquadLowpass:
		alpha := sin / (2 * q)
		b0 := (1 - cos) / 2
		b1 := 1 - cos
		b2 := (1 - cos) / 2
		a0 := 1 + alpha
		a1 := -2 * cos
		a2 := 1 - alpha
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}

	case BiquadHighpass:
		alpha := sin / (2 * q)
		b0 := (1 + cos) / 2
		b1 := -(1 + cos)
		b2 := (1 + cos) / 2
		a0 := 1 + alpha
		a1 := -2 * cos
		a2 := 1 - alpha
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}

	case BiquadBandpass:
		alpha := sin / (2 * q)
		b0 := alpha
		b1 := float64(0)
		b2 := -alpha
		a0 := 1 + alpha
		a1 := -2 * cos
		a2 := 1 - alpha
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}

	case BiquadNotch:
		alpha := sin / (2 * q)
		b0 := float64(1)
		b1 := -2 * cos
		b2 := float64(1)
		a0 := 1 + alpha
		a1 := -2 * cos
		a2 := 1 - alpha
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}

	case BiquadPeaking:
		A := math.Pow(10, dBGain/40)
		alpha := sin / (2 * q)
		b0 := 1 + alpha*A
		b1 := -2 * cos
		b2 := 1 - alpha*A
		a0 := 1 + alpha/A
		a1 := -2 * cos
		a2 := 1 - alpha/A
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}

	case BiquadLowShelf:
		A := math.Pow(10, dBGain/40)
		alpha := sin / (2 * q) * math.Sqrt((A+1/A)*(1/1.0-1)+2)
		sqrtA := math.Sqrt(A)
		b0 := A * ((A + 1) - (A-1)*cos + 2*sqrtA*alpha)
		b1 := 2 * A * ((A - 1) - (A+1)*cos)
		b2 := A * ((A + 1) - (A-1)*cos - 2*sqrtA*alpha)
		a0 := (A + 1) + (A-1)*cos + 2*sqrtA*alpha
		a1 := -2 * ((A - 1) + (A+1)*cos)
		a2 := (A + 1) + (A-1)*cos - 2*sqrtA*alpha
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}

	case BiquadHighShelf:
		A := math.Pow(10, dBGain/40)
		alpha := sin / (2 * q) * math.Sqrt((A+1/A)*(1/1.0-1)+2)
		sqrtA := math.Sqrt(A)
		b0 := A * ((A + 1) + (A-1)*cos + 2*sqrtA*alpha)
		b1 := -2 * A * ((A - 1) + (A+1)*cos)
		b2 := A * ((A + 1) + (A-1)*cos - 2*sqrtA*alpha)
		a0 := (A + 1) - (A-1)*cos + 2*sqrtA*alpha
		a1 := 2 * ((A - 1) - (A+1)*cos)
		a2 := (A + 1) - (A-1)*cos - 2*sqrtA*alpha
		return &Biquad{b0: b0 / a0, b1: b1 / a0, b2: b2 / a0, a1: a1 / a0, a2: a2 / a0}
	}

	return &Biquad{b0: 1}
}

func (b *Biquad) Process(samples []audio.Sample) {
	for i := range samples {
		l := b.tickL(samples[i][0])
		r := b.tickR(samples[i][1])
		samples[i][0] = l
		samples[i][1] = r
	}
}

func (b *Biquad) tickL(x float64) float64 {
	y := b.b0*x + b.b1*b.x1L + b.b2*b.x2L - b.a1*b.y1L - b.a2*b.y2L
	b.x2L = b.x1L
	b.x1L = x
	b.y2L = b.y1L
	b.y1L = y
	return y
}

func (b *Biquad) tickR(x float64) float64 {
	y := b.b0*x + b.b1*b.x1R + b.b2*b.x2R - b.a1*b.y1R - b.a2*b.y2R
	b.x2R = b.x1R
	b.x1R = x
	b.y2R = b.y1R
	b.y1R = y
	return y
}

func (b *Biquad) Reset() {
	b.x1L, b.x2L, b.y1L, b.y2L = 0, 0, 0, 0
	b.x1R, b.x2R, b.y1R, b.y2R = 0, 0, 0, 0
}

func (b *Biquad) LatencySamples() int {
	return 0
}
