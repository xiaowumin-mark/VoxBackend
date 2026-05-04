package separator

import (
	"sync"

	"github.com/xiaowumin-mark/VoxBackend/audio"
)

// Fake 是一个轻量的中置人声近似分离器，适合开发和无 ONNX 模型时兜底。
type Fake struct {
	mu          sync.RWMutex
	vocalAmount float64
	accompBleed float64
	latency     int
	pending     []audio.Sample
	warmupReady chan struct{}
	lastRealOutputSamples int
}

func NewFake(vocalAmount, accompBleed float64, latency int) *Fake {
	ch := make(chan struct{})
	close(ch)
	return &Fake{
		vocalAmount: clamp(vocalAmount, 0, 1.5),
		accompBleed: clamp(accompBleed, 0, 1.5),
		latency:     maxInt(latency, 0),
		warmupReady: ch,
	}
}

func (s *Fake) SetFakeParams(vocalAmount, accompBleed float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.vocalAmount = clamp(vocalAmount, 0, 1.5)
	s.accompBleed = clamp(accompBleed, 0, 1.5)
}

func (s *Fake) WarmupReady() <-chan struct{} {
	return s.warmupReady
}

func (s *Fake) Process(dst *Chunk, mix []audio.Sample) error {
	ensureChunkSize(dst, len(mix))
	s.lastRealOutputSamples = 0

	if s.latency == 0 {
		s.split(dst, mix)
		s.lastRealOutputSamples = len(mix)
		return nil
	}

	s.pending = append(s.pending, mix...)
	available := len(s.pending) - s.latency
	if available < 0 {
		available = 0
	}

	emit := minInt(len(mix), available)
	if emit > 0 {
		s.lastRealOutputSamples = emit
		s.split(dst, s.pending[:emit])
		copy(s.pending, s.pending[emit:])
		s.pending = s.pending[:len(s.pending)-emit]
	}

	for i := emit; i < len(mix); i++ {
		dst.Vocals[i] = audio.Sample{}
		dst.Accomp[i] = audio.Sample{}
	}
	return nil
}

func (s *Fake) LatencySamples() int {
	return s.latency
}

func (s *Fake) Close() error {
	return nil
}

func (s *Fake) Reset() {
	s.pending = nil
	s.lastRealOutputSamples = 0
}

func (s *Fake) ResetOutput() {
	s.pending = nil
	s.lastRealOutputSamples = 0
}

func (s *Fake) LastRealOutputSamples() int {
	return s.lastRealOutputSamples
}

func (s *Fake) Drain(dst *Chunk, maxSamples int) (int, error) {
	if maxSamples <= 0 || len(s.pending) == 0 {
		return 0, nil
	}
	emit := minInt(maxSamples, len(s.pending))
	ensureChunkSize(dst, maxSamples)
	s.split(dst, s.pending[:emit])
	copy(s.pending, s.pending[emit:])
	s.pending = s.pending[:len(s.pending)-emit]
	for i := emit; i < maxSamples; i++ {
		dst.Vocals[i] = audio.Sample{}
		dst.Accomp[i] = audio.Sample{}
	}
	return emit, nil
}

func (s *Fake) split(dst *Chunk, mix []audio.Sample) {
	s.mu.RLock()
	vocalAmount := s.vocalAmount
	accompBleed := s.accompBleed
	s.mu.RUnlock()

	for i := range mix {
		left := mix[i][0]
		right := mix[i][1]
		center := 0.5 * (left + right) * vocalAmount

		vocal := audio.Sample{center, center}
		accomp := audio.Sample{
			left - accompBleed*vocal[0],
			right - accompBleed*vocal[1],
		}

		dst.Vocals[i] = vocal
		dst.Accomp[i] = accomp
	}
}
