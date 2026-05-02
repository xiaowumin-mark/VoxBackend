package dsp

func PresetVocalChain(sampleRate int) *Chain {
	return NewChain(
		NewGate(-40, 4, 2, 100, sampleRate),
		NewHighpass(80, 0.707, sampleRate),
		NewLowpass(12000, 0.707, sampleRate),
		NewPeaking(3000, 1.5, 2, sampleRate),
		NewCompressor(-18, 2, 2, 5, 100, sampleRate),
	)
}

func PresetAccompChain(sampleRate int) *Chain {
	return NewChain(
		NewHighpass(30, 0.707, sampleRate),
		NewCompressor(-14, 1.5, 0, 10, 80, sampleRate),
		NewMidSide(1.0, 0.85, sampleRate),
		NewHighShelf(8000, 0.7, 1.5, sampleRate),
		NewSaturator(0.3, 0.15, CurveTanh),
	)
}
