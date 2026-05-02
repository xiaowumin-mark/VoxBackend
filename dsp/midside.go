package dsp

import (
	"math"
	"sync/atomic"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

type MidSide struct {
	midGain        float64
	sideGain       float64
	presetSideGain float64
	monoBass       bool
	bassFreqHz     float64
	sampleRate     int
	lpfL           *Biquad
	lpfR           *Biquad
	midGainBits    atomic.Uint64
	sideGainBits   atomic.Uint64
}

func NewMidSide(midGain, sideGain float64, sampleRate int) *MidSide {
	ms := &MidSide{
		midGain:        clamp(midGain, 0, 4),
		sideGain:       clamp(sideGain, 0, 4),
		presetSideGain: clamp(sideGain, 0, 4),
		monoBass:       false,
		bassFreqHz:     120,
		sampleRate:     sampleRate,
		lpfL:           NewLowpass(120, 0.707, sampleRate),
		lpfR:           NewLowpass(120, 0.707, sampleRate),
	}
	ms.midGainBits.Store(math.Float64bits(ms.midGain))
	ms.sideGainBits.Store(math.Float64bits(ms.sideGain))
	return ms
}

func (m *MidSide) SetMidGain(g float64) {
	m.midGainBits.Store(math.Float64bits(clamp(g, 0, 4)))
}

func (m *MidSide) SetSideGain(g float64) {
	m.sideGainBits.Store(math.Float64bits(clamp(g, 0, 4)))
}

func (m *MidSide) SetIntensity(intensity float64) {
	intensity = clamp(intensity, 0, 1)
	side := 1.0 + intensity*(m.presetSideGain-1.0)
	m.SetSideGain(side)
}

func (m *MidSide) SetMonoBass(enabled bool) {
	m.monoBass = enabled
}

func (m *MidSide) Process(samples []audio.Sample) {
	midGain := math.Float64frombits(m.midGainBits.Load())
	sideGain := math.Float64frombits(m.sideGainBits.Load())

	if m.monoBass {
		lowL := make([]audio.Sample, len(samples))
		lowR := make([]audio.Sample, len(samples))
		copy(lowL, samples)
		copy(lowR, samples)
		m.lpfL.Process(lowL)
		m.lpfR.Process(lowR)

		for i := range samples {
			midL := lowL[i][0]
			midR := lowR[i][1]
			mono := (midL + midR) * 0.5
			lowL[i][0] = mono
			lowL[i][1] = mono
		}

		midOnly := make([]audio.Sample, len(samples))
		copy(midOnly, samples)
		for i := range midOnly {
			midOnly[i][0] -= lowL[i][0]
			midOnly[i][1] -= lowL[i][1]
		}
		copy(samples, midOnly)
		for i := range samples {
			samples[i][0] += lowL[i][0]
			samples[i][1] += lowL[i][1]
		}
	}

	rsqrt2 := 1.0 / math.Sqrt2
	for i := range samples {
		mid := (samples[i][0] + samples[i][1]) * rsqrt2
		side := (samples[i][0] - samples[i][1]) * rsqrt2
		mid *= midGain
		side *= sideGain
		samples[i][0] = (mid + side) * rsqrt2
		samples[i][1] = (mid - side) * rsqrt2
	}
}

func (m *MidSide) Reset() {
	m.lpfL.Reset()
	m.lpfR.Reset()
}

func (m *MidSide) LatencySamples() int {
	return 0
}
