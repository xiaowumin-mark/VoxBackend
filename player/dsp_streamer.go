package player

import (
	"github.com/gopxl/beep/v2"
	"github.com/xiaowumin-mark/VoxBackend/audio"
	"github.com/xiaowumin-mark/VoxBackend/dsp"
)

type dspStreamer struct {
	ring        *audio.Ring
	mixer       *audio.Mixer
	vocalChain  *dsp.Chain
	accompChain *dsp.Chain
	err         error
	vocalBuf    []audio.Sample
	accompBuf   []audio.Sample

	mode           DSPMode
	bypassed       bool
	fadingRemain   int
	fadingToBypass bool
	dryVocal       []audio.Sample
	dryAccomp      []audio.Sample
}

const crossfadeDurationSamples = 2205

func newDSPStreamer(ring *audio.Ring, mixer *audio.Mixer, vocalChain, accompChain *dsp.Chain, mode DSPMode) *dspStreamer {
	s := &dspStreamer{
		ring:        ring,
		mixer:       mixer,
		vocalChain:  vocalChain,
		accompChain: accompChain,
		mode:        mode,
	}
	if mode == DSPModeOff {
		s.bypassed = true
	}
	return s
}

func (s *dspStreamer) Stream(samples [][2]float64) (int, bool) {
	n := len(samples)
	s.growBufs(n)

	nPairs, ok := s.ring.ReadPairs(s.vocalBuf[:n], s.accompBuf[:n])
	if !ok {
		s.err = s.ring.Err()
		return 0, false
	}

	vox := s.vocalBuf[:nPairs]
	acc := s.accompBuf[:nPairs]

	if s.bypassed && s.fadingRemain == 0 {
		for i := 0; i < nPairs; i++ {
			samples[i] = s.mixer.MixSingle(vox[i], acc[i])
		}
		return nPairs, true
	}

	if s.fadingRemain > 0 {
		s.growDryBufs(nPairs)
		copy(s.dryVocal, vox)
		copy(s.dryAccomp, acc)
	}

	if s.vocalChain != nil {
		s.vocalChain.Process(vox)
	}
	if s.accompChain != nil {
		s.accompChain.Process(acc)
	}

	if s.fadingRemain > 0 {
		fadingFrom := nPairs - s.fadingRemain
		if fadingFrom < 0 {
			fadingFrom = 0
		}
		for i := fadingFrom; i < nPairs; i++ {
			offset := i - fadingFrom + (crossfadeDurationSamples - s.fadingRemain)
			alpha := clampFloat(1, float64(offset+1)/float64(crossfadeDurationSamples+1))
			if s.fadingToBypass {
				alpha = 1 - alpha
			}
			dryVox := s.dryVocal[i]
			dryAcc := s.dryAccomp[i]
			wetVox := vox[i]
			wetAcc := acc[i]
			vox[i][0] = dryVox[0]*(1-alpha) + wetVox[0]*alpha
			vox[i][1] = dryVox[1]*(1-alpha) + wetVox[1]*alpha
			acc[i][0] = dryAcc[0]*(1-alpha) + wetAcc[0]*alpha
			acc[i][1] = dryAcc[1]*(1-alpha) + wetAcc[1]*alpha
		}
		s.fadingRemain -= nPairs
		if s.fadingRemain <= 0 {
			s.fadingRemain = 0
			s.bypassed = s.fadingToBypass
		}
	}

	for i := 0; i < nPairs; i++ {
		samples[i] = s.mixer.MixSingle(vox[i], acc[i])
	}
	return nPairs, true
}

func (s *dspStreamer) OnVocalGainChanged(gain float64) {
	if s.mode != DSPModeAuto {
		return
	}

	intensity := 1.0 - gain
	if intensity < 0 {
		intensity = 0
	}
	if intensity > 1.5 {
		intensity = 1.5
	}

	targetBypassed := intensity < 0.05

	if targetBypassed != s.bypassed && s.fadingRemain <= 0 {
		s.fadingRemain = crossfadeDurationSamples
		s.fadingToBypass = targetBypassed
	}

	if !targetBypassed {
		dsp.ApplyIntensityToChains(s.vocalChain, s.accompChain, intensity)
	}
}

func (s *dspStreamer) SetMode(mode DSPMode) {
	s.mode = mode
	switch mode {
	case DSPModeOff:
		s.startCrossfadeToBypass(true)
	case DSPModeOn:
		s.startCrossfadeToBypass(false)
	case DSPModeAuto:
	}
}

func (s *dspStreamer) startCrossfadeToBypass(targetBypass bool) {
	if s.bypassed == targetBypass && s.fadingRemain <= 0 {
		return
	}
	s.fadingRemain = crossfadeDurationSamples
	s.fadingToBypass = targetBypass
}

func (s *dspStreamer) Err() error {
	return s.err
}

func (s *dspStreamer) Mixer() *audio.Mixer {
	return s.mixer
}

func (s *dspStreamer) growBufs(n int) {
	if cap(s.vocalBuf) < n {
		s.vocalBuf = make([]audio.Sample, n)
	}
	s.vocalBuf = s.vocalBuf[:n]
	if cap(s.accompBuf) < n {
		s.accompBuf = make([]audio.Sample, n)
	}
	s.accompBuf = s.accompBuf[:n]
}

func (s *dspStreamer) growDryBufs(n int) {
	if cap(s.dryVocal) < n {
		s.dryVocal = make([]audio.Sample, n, n)
	}
	s.dryVocal = s.dryVocal[:n]
	if cap(s.dryAccomp) < n {
		s.dryAccomp = make([]audio.Sample, n, n)
	}
	s.dryAccomp = s.dryAccomp[:n]
}

func clampFloat(v, x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > v {
		return v
	}
	return x
}

var _ beep.Streamer = (*audio.RealtimeStreamer)(nil)
