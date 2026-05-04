package separator

import "github.com/xiaowumin-mark/VoxBackend/audio"

// Chunk 保存一次分离结果。
type Chunk struct {
	Vocals []audio.Sample
	Accomp []audio.Sample
}

// Separator 是音频分离器的统一接口。
type Separator interface {
	Process(dst *Chunk, mix []audio.Sample) error
	LatencySamples() int
	WarmupReady() <-chan struct{}
	Close() error
	Reset()
	ResetOutput()
}

// Drainable 表示分离器可以在输入结束后排出内部延迟样本。
type Drainable interface {
	Drain(dst *Chunk, maxSamples int) (int, error)
}

type ParameterizedFake interface {
	SetFakeParams(vocalAmount, accompBleed float64)
}

func ensureChunkSize(dst *Chunk, n int) {
	if len(dst.Vocals) != n {
		dst.Vocals = make([]audio.Sample, n)
	}
	if len(dst.Accomp) != n {
		dst.Accomp = make([]audio.Sample, n)
	}
}
