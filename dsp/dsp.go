package dsp

import "github.com/xiaowumin-mark/VoxBackend/audio"

type Processor interface {
	Process(samples []audio.Sample)
	Reset()
	LatencySamples() int
}

type Chain struct {
	processors []Processor
}

func NewChain(processors ...Processor) *Chain {
	return &Chain{processors: append([]Processor{}, processors...)}
}

func (c *Chain) Process(samples []audio.Sample) {
	for _, p := range c.processors {
		p.Process(samples)
	}
}

func (c *Chain) Reset() {
	for _, p := range c.processors {
		p.Reset()
	}
}

func (c *Chain) LatencySamples() int {
	total := 0
	for _, p := range c.processors {
		total += p.LatencySamples()
	}
	return total
}

func (c *Chain) Processors() []Processor {
	return c.processors
}
