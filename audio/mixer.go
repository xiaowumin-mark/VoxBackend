package audio

import (
	"math"
	"sync/atomic"
)

// Mixer 将分离得到的人声与伴奏重新混合，并支持运行时热更新参数。
type Mixer struct {
	targetGainBits   atomic.Uint64
	currentGainBits  atomic.Uint64
	masterVolumeBits atomic.Uint64
	rampMsBits       atomic.Uint64
	sampleRate       int
}

func NewMixer(vocalGain, masterVolume float64, sampleRate int, rampDurationMs float64) *Mixer {
	m := &Mixer{sampleRate: sampleRate}
	m.SetRampDurationMs(rampDurationMs)
	m.targetGainBits.Store(math.Float64bits(vocalGain))
	m.currentGainBits.Store(math.Float64bits(vocalGain))
	m.masterVolumeBits.Store(math.Float64bits(clamp(masterVolume, 0, 5.0)))
	return m
}

func (m *Mixer) SetMasterVolume(v float64) {
	m.masterVolumeBits.Store(math.Float64bits(clamp(v, 0, 5.0)))
}

func (m *Mixer) MasterVolume() float64 {
	return math.Float64frombits(m.masterVolumeBits.Load())
}

func (m *Mixer) SetVocalGain(v float64) {
	m.targetGainBits.Store(math.Float64bits(v))
}

func (m *Mixer) VocalGain() float64 {
	return math.Float64frombits(m.targetGainBits.Load())
}

func (m *Mixer) SetRampDurationMs(rampDurationMs float64) {
	if rampDurationMs < 0 {
		rampDurationMs = 0
	}
	m.rampMsBits.Store(math.Float64bits(rampDurationMs))
}

func (m *Mixer) RampDurationMs() float64 {
	return math.Float64frombits(m.rampMsBits.Load())
}

func (m *Mixer) MixSingle(vocals, accomp Sample) Sample {
	currentGain := math.Float64frombits(m.currentGainBits.Load())
	targetGain := math.Float64frombits(m.targetGainBits.Load())
	masterVol := math.Float64frombits(m.masterVolumeBits.Load())

	currentGain = slewTowards(currentGain, targetGain, m.slewPerSample())
	m.currentGainBits.Store(math.Float64bits(currentGain))

	left := (accomp[0] + currentGain*vocals[0]) * 0.8 * masterVol
	right := (accomp[1] + currentGain*vocals[1]) * 0.8 * masterVol
	return Sample{softClip(left), softClip(right)}
}

func (m *Mixer) slewPerSample() float64 {
	if m.sampleRate <= 0 {
		return 1
	}
	rampDurationMs := m.RampDurationMs()
	if rampDurationMs <= 0 {
		return math.Inf(1)
	}
	rampSamples := float64(m.sampleRate) * rampDurationMs / 1000.0
	if rampSamples < 1 {
		rampSamples = 1
	}
	return 1.0 / rampSamples
}

func softClip(v float64) float64 {
	return math.Tanh(v)
}

func slewTowards(current, target, maxStep float64) float64 {
	if current == target {
		return current
	}
	if math.IsInf(maxStep, 1) {
		return target
	}
	delta := target - current
	if math.Abs(delta) <= maxStep {
		return target
	}
	if delta > 0 {
		return current + maxStep
	}
	return current - maxStep
}
