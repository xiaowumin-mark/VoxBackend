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
	DefaultVocalRamp      = 1000 * time.Millisecond
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
	if isZeroConfig(cfg) {
		return def
	}
	if onlyTracksOrCallbacks(cfg) {
		def.Tracks = cloneTracks(cfg.Tracks)
		def.Callbacks = cfg.Callbacks
		return def
	}

	preserveZeroValues := resemblesDefaultConfig(cfg, def)
	if cfg.SeparatorMode == "" {
		cfg.SeparatorMode = def.SeparatorMode
	}
	if !preserveZeroValues {
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
		if cfg.Crossfade == 0 {
			cfg.Crossfade = def.Crossfade
		}
		if cfg.Prewarm == 0 {
			cfg.Prewarm = def.Prewarm
		}
	}
	if cfg.ONNX.Profile == "" {
		cfg.ONNX.Profile = def.ONNX.Profile
	}
	if cfg.ONNX.ModelPath == "" {
		cfg.ONNX.ModelPath = def.ONNX.ModelPath
	}
	if cfg.SpeakerBufferSize <= 0 {
		cfg.SpeakerBufferSize = def.SpeakerBufferSize
	}
	if cfg.StateFrameRate <= 0 {
		cfg.StateFrameRate = def.StateFrameRate
	}
	if cfg.DSP.Mode == "" {
		cfg.DSP.Mode = def.DSP.Mode
	} else if !isValidDSPMode(cfg.DSP.Mode) {
		cfg.DSP.Mode = def.DSP.Mode
	}
	cfg.Tracks = cloneTracks(cfg.Tracks)
	return cfg
}

func isValidDSPMode(mode DSPMode) bool {
	return mode == DSPModeOff || mode == DSPModeOn || mode == DSPModeAuto
}

func onlyTracksOrCallbacks(cfg Config) bool {
	return cfg.SeparatorMode == "" &&
		cfg.VocalGain == 0 &&
		cfg.VocalGainRamp == 0 &&
		cfg.MasterVolume == 0 &&
		cfg.FakeVocalAmount == 0 &&
		cfg.FakeAccompBleed == 0 &&
		cfg.AILatency == 0 &&
		cfg.ONNX == (separator.ONNXConfig{}) &&
		cfg.Crossfade == 0 &&
		cfg.Prewarm == 0 &&
		cfg.DSP == (DSPConfig{}) &&
		cfg.SpeakerBufferSize == 0 &&
		cfg.StateFrameRate == 0
}

func resemblesDefaultConfig(cfg, def Config) bool {
	score := 0
	if cfg.SeparatorMode == def.SeparatorMode {
		score++
	}
	if cfg.VocalGain == def.VocalGain {
		score++
	}
	if cfg.VocalGainRamp == def.VocalGainRamp {
		score++
	}
	if cfg.MasterVolume == def.MasterVolume {
		score++
	}
	if cfg.FakeVocalAmount == def.FakeVocalAmount {
		score++
	}
	if cfg.FakeAccompBleed == def.FakeAccompBleed {
		score++
	}
	if cfg.AILatency == def.AILatency {
		score++
	}
	if cfg.ONNX == def.ONNX {
		score++
	}
	if cfg.Crossfade == def.Crossfade {
		score++
	}
	if cfg.Prewarm == def.Prewarm {
		score++
	}
	if cfg.DSP == def.DSP {
		score++
	}
	if cfg.SpeakerBufferSize == def.SpeakerBufferSize {
		score++
	}
	if cfg.StateFrameRate == def.StateFrameRate {
		score++
	}
	return score >= 4
}

func isZeroConfig(cfg Config) bool {
	return cfg.SeparatorMode == "" &&
		len(cfg.Tracks) == 0 &&
		cfg.VocalGain == 0 &&
		cfg.VocalGainRamp == 0 &&
		cfg.MasterVolume == 0 &&
		cfg.FakeVocalAmount == 0 &&
		cfg.FakeAccompBleed == 0 &&
		cfg.AILatency == 0 &&
		cfg.ONNX == (separator.ONNXConfig{}) &&
		cfg.Crossfade == 0 &&
		cfg.Prewarm == 0 &&
		cfg.DSP == (DSPConfig{}) &&
		cfg.SpeakerBufferSize == 0 &&
		cfg.StateFrameRate == 0 &&
		isZeroCallbacks(cfg.Callbacks)
}

func isZeroCallbacks(cb Callbacks) bool {
	return cb.OnEvent == nil &&
		cb.OnState == nil &&
		cb.OnPausedChanged == nil &&
		cb.OnVolumeChanged == nil &&
		cb.OnTrackChanged == nil &&
		cb.OnVocalChanged == nil &&
		cb.OnPlaylistChanged == nil
}
