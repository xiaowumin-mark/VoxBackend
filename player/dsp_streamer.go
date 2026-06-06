package player

import (
	"sync"

	"github.com/gopxl/beep/v2"
	"github.com/xiaowumin-mark/VoxBackend/audio"
	"github.com/xiaowumin-mark/VoxBackend/dsp"
)

type dspStreamer struct {
	mu          sync.Mutex
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
		s.mu.Lock()
		s.err = s.ring.Err()
		s.mu.Unlock()
		return 0, false
	}

	vox := s.vocalBuf[:nPairs]
	acc := s.accompBuf[:nPairs]

	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mode != DSPModeAuto {
		return
	}

	intensity := dspIntensityFromGain(gain)
	targetBypassed := intensity < 0.05

	if targetBypassed == s.bypassed {
		s.fadingRemain = 0
	} else if s.fadingRemain <= 0 || s.fadingToBypass != targetBypassed {
		s.fadingRemain = crossfadeDurationSamples
		s.fadingToBypass = targetBypassed
	}

	if !targetBypassed {
		dsp.ApplyIntensityToChains(s.vocalChain, s.accompChain, intensity)
	}
}

func (s *dspStreamer) SetInitialVocalGain(gain float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mode != DSPModeAuto {
		return
	}

	intensity := dspIntensityFromGain(gain)
	s.fadingRemain = 0
	s.fadingToBypass = false
	s.bypassed = intensity < 0.05
	if !s.bypassed {
		dsp.ApplyIntensityToChains(s.vocalChain, s.accompChain, intensity)
	}
}

func (s *dspStreamer) SetInitialBypassed(bypassed bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if bypassed {
		s.mode = DSPModeOff
	}
	s.bypassed = bypassed
	s.fadingRemain = 0
	s.fadingToBypass = bypassed
}

func (s *dspStreamer) SetModeImmediate(mode DSPMode, gain float64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.mode = mode
	s.fadingRemain = 0
	switch mode {
	case DSPModeOff:
		s.bypassed = true
		s.fadingToBypass = true
	case DSPModeOn:
		s.bypassed = false
		s.fadingToBypass = false
	case DSPModeAuto:
		intensity := dspIntensityFromGain(gain)
		s.bypassed = intensity < 0.05
		s.fadingToBypass = s.bypassed
		if !s.bypassed {
			dsp.ApplyIntensityToChains(s.vocalChain, s.accompChain, intensity)
		}
	}
}

func (s *dspStreamer) SetMode(mode DSPMode) {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()
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

func dspIntensityFromGain(gain float64) float64 {
	intensity := 1.0 - gain
	if intensity < 0 {
		return 0
	}
	if intensity > 1.5 {
		return 1.5
	}
	return intensity
}

var _ beep.Streamer = (*audio.RealtimeStreamer)(nil)
