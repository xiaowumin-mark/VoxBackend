package player

import (
	"time"

	"github.com/xiaowumin-mark/VoxBackend/separator"
)

const (
	ChunkSize             = 2048
	BufferChunks          = 128
	SpeakerBufferSize     = 4096
	DefaultAILatency      = ChunkSize * 2
	DefaultONNXModel      = "umxl_vocals.onnx"
	DefaultSeparator      = "fake"
	DefaultCrossfade      = 12 * time.Second
	DefaultPrewarm        = 6 * time.Second
	DefaultMasterVolume   = 1.0
	DefaultVocalGain      = 1.0
	DefaultVocalRamp      = 20 * time.Millisecond
	DefaultStateFrameRate = 60
)

type SeparatorMode string

const (
	SeparatorFake SeparatorMode = "fake"
	SeparatorONNX SeparatorMode = "onnx"
)

type Config struct {
	SeparatorMode SeparatorMode
	Tracks        []Track

	VocalGain       float64
	VocalGainRamp   time.Duration
	MasterVolume    float64
	FakeVocalAmount float64
	FakeAccompBleed float64
	AILatency       int

	ONNX separator.ONNXConfig

	Crossfade time.Duration
	Prewarm   time.Duration

	DSP DSPConfig

	SpeakerBufferSize int
	StateFrameRate    int
	Callbacks         Callbacks
}

type DSPMode string

const (
	DSPModeOff  DSPMode = "off"
	DSPModeOn   DSPMode = "on"
	DSPModeAuto DSPMode = "auto"
)

type DSPConfig struct {
	Mode DSPMode
}

func DefaultConfig() Config {
	return Config{
		SeparatorMode:     SeparatorFake,
		VocalGain:         DefaultVocalGain,
		VocalGainRamp:     DefaultVocalRamp,
		MasterVolume:      DefaultMasterVolume,
		FakeVocalAmount:   0.85,
		FakeAccompBleed:   1.0,
		AILatency:         DefaultAILatency,
		ONNX:              separator.ONNXConfig{Profile: separator.DefaultUMXVocalsProfile, ModelPath: DefaultONNXModel},
		Crossfade:         DefaultCrossfade,
		Prewarm:           DefaultPrewarm,
		DSP:               DSPConfig{Mode: DSPModeAuto},
		SpeakerBufferSize: SpeakerBufferSize,
		StateFrameRate:    DefaultStateFrameRate,
	}
}

func normalizeConfig(cfg Config) Config {
	def := DefaultConfig()
	if cfg.SeparatorMode == "" {
		cfg.SeparatorMode = def.SeparatorMode
	}
	if cfg.VocalGain == 0 {
		cfg.VocalGain = def.VocalGain
	}
	if cfg.MasterVolume == 0 {
		cfg.MasterVolume = def.MasterVolume
	}
	if cfg.VocalGainRamp == 0 {
		cfg.VocalGainRamp = def.VocalGainRamp
	}
	if cfg.FakeVocalAmount == 0 {
		cfg.FakeVocalAmount = def.FakeVocalAmount
	}
	if cfg.FakeAccompBleed == 0 {
		cfg.FakeAccompBleed = def.FakeAccompBleed
	}
	if cfg.AILatency == 0 {
		cfg.AILatency = def.AILatency
	}
	if cfg.ONNX.Profile == "" {
		cfg.ONNX.Profile = def.ONNX.Profile
	}
	if cfg.ONNX.ModelPath == "" {
		cfg.ONNX.ModelPath = def.ONNX.ModelPath
	}
	if cfg.Crossfade == 0 {
		cfg.Crossfade = def.Crossfade
	}
	if cfg.Prewarm == 0 {
		cfg.Prewarm = def.Prewarm
	}
	if cfg.SpeakerBufferSize <= 0 {
		cfg.SpeakerBufferSize = def.SpeakerBufferSize
	}
	if cfg.StateFrameRate <= 0 {
		cfg.StateFrameRate = def.StateFrameRate
	}
	if cfg.DSP.Mode == "" {
		cfg.DSP.Mode = def.DSP.Mode
	}
	return cfg
}
