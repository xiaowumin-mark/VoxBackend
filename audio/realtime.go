package audio

// RealtimeStreamer 从 Ring 中读取 vocals/accomp 对，并实时混音给 beep 播放。
type RealtimeStreamer struct {
	ring  *Ring
	mixer *Mixer
	err   error
}

func NewRealtimeStreamer(ring *Ring, mixer *Mixer) *RealtimeStreamer {
	return &RealtimeStreamer{ring: ring, mixer: mixer}
}

func (s *RealtimeStreamer) Stream(samples [][2]float64) (int, bool) {
	n, ok := s.ring.MixForPlayback(samples, s.mixer)
	if !ok {
		s.err = s.ring.Err()
		return 0, false
	}
	return n, true
}

func (s *RealtimeStreamer) Err() error {
	return s.err
}

func (s *RealtimeStreamer) Mixer() *Mixer {
	return s.mixer
}
