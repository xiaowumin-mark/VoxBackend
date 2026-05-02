package dsp

import (
	"math"
	"sync/atomic"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

type SaturatorCurve int

const (
	CurveTanh SaturatorCurve = iota
	CurveCubic
	CurveAsymTube
)

type Saturator struct {
	driveBits   atomic.Uint64
	mixBits     atomic.Uint64
	presetDrive float64
	presetMix   float64
	curve       SaturatorCurve
}

func NewSaturator(drive, mix float64, curve SaturatorCurve) *Saturator {
	s := &Saturator{
		presetDrive: drive,
		presetMix:   clamp(mix, 0, 1),
		curve:       curve,
	}
	s.SetDrive(drive)
	s.SetMix(mix)
	return s
}

func (s *Saturator) SetDrive(drive float64) {
	s.driveBits.Store(math.Float64bits(clamp(drive, 0, 10)))
}

func (s *Saturator) SetMix(x float64) {
	s.mixBits.Store(math.Float64bits(clamp(x, 0, 1)))
}

func (s *Saturator) drive() float64 {
	return math.Float64frombits(s.driveBits.Load())
}

func (s *Saturator) mix() float64 {
	return math.Float64frombits(s.mixBits.Load())
}

func (s *Saturator) SetIntensity(intensity float64) {
	intensity = clamp(intensity, 0, 1)
	s.SetDrive(intensity * s.presetDrive)
	s.SetMix(intensity * s.presetMix)
}

func (s *Saturator) Process(samples []audio.Sample) {
	m := s.mix()
	if m <= 0 {
		return
	}
	d := s.drive()
	for i := range samples {
		wetL := s.applyCurve(samples[i][0] * d)
		wetR := s.applyCurve(samples[i][1] * d)
		samples[i][0] = samples[i][0]*(1-m) + wetL*m
		samples[i][1] = samples[i][1]*(1-m) + wetR*m
	}
}

func (s *Saturator) applyCurve(x float64) float64 {
	switch s.curve {
	case CurveTanh:
		return math.Tanh(x)
	case CurveCubic:
		x2 := x * x
		if x > 1.5 {
			return 1.0
		}
		if x < -1.5 {
			return -1.0
		}
		return x - x2*x/3.0
	case CurveAsymTube:
		if x > 0 {
			y := math.Tanh(x)
			return y*0.85 + math.Tanh(x*0.5)*0.15
		}
		y := math.Tanh(x * 1.3)
		return y*0.85 + math.Tanh(x*0.4)*0.15
	default:
		return x
	}
}

func (s *Saturator) Reset() {}

func (s *Saturator) LatencySamples() int {
	return 0
}
