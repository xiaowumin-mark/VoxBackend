package dsp

import (
	"math"
	"sync/atomic"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

type Gate struct {
	thresholdDB       atomic.Uint64
	presetThresholdDB float64
	ratio             float64
	attackS           float64
	releaseS          float64
	envelopeL         float64
	envelopeR         float64
	attackCoeff       float64
	releaseCoeff      float64
	sampleRate        int
}

func NewGate(thresholdDB, ratio float64, attackMs, releaseMs float64, sampleRate int) *Gate {
	g := &Gate{
		presetThresholdDB: thresholdDB,
		ratio:             clamp(ratio, 1, 20),
		sampleRate:        sampleRate,
	}
	g.SetAttackMs(attackMs)
	g.SetReleaseMs(releaseMs)
	g.SetThresholdDB(thresholdDB)
	return g
}

func (g *Gate) SetThresholdDB(db float64) {
	g.thresholdDB.Store(math.Float64bits(db))
}

func (g *Gate) ThresholdDB() float64 {
	return math.Float64frombits(g.thresholdDB.Load())
}

func (g *Gate) SetIntensity(intensity float64) {
	intensity = clamp(intensity, 0, 1.5)
	off := -60.0
	db := off + intensity*(g.presetThresholdDB-off)
	g.SetThresholdDB(db)
}

func (g *Gate) SetAttackMs(ms float64) {
	if ms <= 0 {
		ms = 0.1
	}
	g.attackS = ms / 1000.0
	g.attackCoeff = math.Exp(-1 / (float64(g.sampleRate) * g.attackS))
}

func (g *Gate) SetReleaseMs(ms float64) {
	if ms <= 0 {
		ms = 10
	}
	g.releaseS = ms / 1000.0
	g.releaseCoeff = math.Exp(-1 / (float64(g.sampleRate) * g.releaseS))
}

func (g *Gate) thresholdLin() float64 {
	return math.Pow(10, math.Float64frombits(g.thresholdDB.Load())/20)
}

func (g *Gate) Process(samples []audio.Sample) {
	thresh := g.thresholdLin()
	for i := range samples {
		envL := g.updateEnvelope(&g.envelopeL, math.Abs(samples[i][0]))
		envR := g.updateEnvelope(&g.envelopeR, math.Abs(samples[i][1]))

		gainL := g.computeGain(envL, thresh)
		gainR := g.computeGain(envR, thresh)

		samples[i][0] *= gainL
		samples[i][1] *= gainR
	}
}

func (g *Gate) updateEnvelope(env *float64, sampleAbs float64) float64 {
	var coeff float64
	if sampleAbs > *env {
		coeff = g.attackCoeff
	} else {
		coeff = g.releaseCoeff
	}
	*env = coeff*(*env) + (1-coeff)*sampleAbs
	return *env
}

func (g *Gate) computeGain(env, thresh float64) float64 {
	if env >= thresh {
		return 1
	}
	if env < 1e-12 {
		env = 1e-12
	}
	desired := env * math.Pow(env/thresh, g.ratio-1)
	return clamp(desired/env, 0, 1)
}

func (g *Gate) Reset() {
	g.envelopeL = 0
	g.envelopeR = 0
}

func (g *Gate) LatencySamples() int {
	return 0
}
