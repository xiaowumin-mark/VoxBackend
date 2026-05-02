package dsp

func ApplyIntensityToChains(vocal, accomp *Chain, intensity float64) {
	if vocal != nil {
		for _, p := range vocal.Processors() {
			applyIntensity(p, intensity)
		}
	}
	if accomp != nil {
		for _, p := range accomp.Processors() {
			applyIntensity(p, intensity)
		}
	}
}

func applyIntensity(p Processor, intensity float64) {
	switch v := p.(type) {
	case *Gate:
		v.SetIntensity(intensity)
	case *Compressor:
		v.SetIntensity(intensity)
	case *MidSide:
		v.SetIntensity(intensity)
	case *Saturator:
		v.SetIntensity(intensity)
	}
}
