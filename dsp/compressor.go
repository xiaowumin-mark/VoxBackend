package dsp

import (
	"math"
	"sync/atomic"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

type Compressor struct {
	thresholdDB        atomic.Uint64
	ratioBits          atomic.Uint64
	makeupGainDB       atomic.Uint64
	presetThresholdDB  float64
	presetRatio        float64
	presetMakeupGainDB float64
	attackS            float64
	releaseS           float64
	envelopeL          float64
	envelopeR          float64
	attackCoeff        float64
	releaseCoeff       float64
	sampleRate         int
}

func NewCompressor(thresholdDB, ratio, makeupGainDB float64, attackMs, releaseMs float64, sampleRate int) *Compressor {
	c := &Compressor{
		presetThresholdDB:  thresholdDB,
		presetRatio:        clamp(ratio, 1, 20),
		presetMakeupGainDB: makeupGainDB,
		sampleRate:         sampleRate,
	}
	c.SetAttackMs(attackMs)
	c.SetReleaseMs(releaseMs)
	c.SetThresholdDB(thresholdDB)
	c.SetRatio(ratio)
	c.SetMakeupGainDB(makeupGainDB)
	return c
}

func (c *Compressor) SetThresholdDB(db float64) {
	c.thresholdDB.Store(math.Float64bits(db))
}

func (c *Compressor) SetRatio(ratio float64) {
	c.ratioBits.Store(math.Float64bits(clamp(ratio, 1, 20)))
}

func (c *Compressor) SetMakeupGainDB(db float64) {
	c.makeupGainDB.Store(math.Float64bits(db))
}

func (c *Compressor) ThresholdDB() float64 {
	return math.Float64frombits(c.thresholdDB.Load())
}

func (c *Compressor) ratio() float64 {
	return math.Float64frombits(c.ratioBits.Load())
}

func (c *Compressor) SetIntensity(intensity float64) {
	intensity = clamp(intensity, 0, 1.5)
	c.SetThresholdDB(intensity * c.presetThresholdDB)
	c.SetRatio(1 + intensity*(c.presetRatio-1))
	c.SetMakeupGainDB(intensity * c.presetMakeupGainDB)
}

func (c *Compressor) SetAttackMs(ms float64) {
	if ms <= 0 {
		ms = 0.1
	}
	c.attackS = ms / 1000.0
	c.attackCoeff = math.Exp(-1 / (float64(c.sampleRate) * c.attackS))
}

func (c *Compressor) SetReleaseMs(ms float64) {
	if ms <= 0 {
		ms = 10
	}
	c.releaseS = ms / 1000.0
	c.releaseCoeff = math.Exp(-1 / (float64(c.sampleRate) * c.releaseS))
}

func (c *Compressor) thresholdLin() float64 {
	return math.Pow(10, math.Float64frombits(c.thresholdDB.Load())/20)
}

func (c *Compressor) makeupLin() float64 {
	return math.Pow(10, math.Float64frombits(c.makeupGainDB.Load())/20)
}

func (c *Compressor) Process(samples []audio.Sample) {
	thresh := c.thresholdLin()
	makeup := c.makeupLin()
	for i := range samples {
		envL := c.updateEnvelope(&c.envelopeL, math.Abs(samples[i][0]))
		envR := c.updateEnvelope(&c.envelopeR, math.Abs(samples[i][1]))

		gainL := c.computeGain(envL, thresh)
		gainR := c.computeGain(envR, thresh)

		samples[i][0] *= gainL * makeup
		samples[i][1] *= gainR * makeup
	}
}

func (c *Compressor) updateEnvelope(env *float64, sampleAbs float64) float64 {
	var coeff float64
	if sampleAbs > *env {
		coeff = c.attackCoeff
	} else {
		coeff = c.releaseCoeff
	}
	*env = coeff*(*env) + (1-coeff)*sampleAbs
	return *env
}

func (c *Compressor) computeGain(env, thresh float64) float64 {
	if env <= thresh {
		return 1
	}
	overshoot := env / thresh
	desired := thresh * math.Pow(overshoot, 1/c.ratio())
	return clamp(desired/env, 0, 1)
}

func (c *Compressor) Reset() {
	c.envelopeL = 0
	c.envelopeR = 0
}

func (c *Compressor) LatencySamples() int {
	return 0
}
