package separator

import (
	"errors"
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/xiaowumin-mark/VoxBackend/audio"
	ort "github.com/yalue/onnxruntime_go"
)

const ONNXSampleRate = 44100
const defaultChunkSize = 2048

const (
	DefaultUMXVocalsProfile = "umx-vocals"
	MDXInstHQ3Profile       = "mdx-inst-hq3"
	MDXVocFTProfile         = "mdx-voc-ft"
	MDX23CVocalsProfile     = "mdx23c-vocals"
	MDXKaraProfile          = "mdx-kara"
	MDXKara2Profile         = "mdx-kara2"
)

type StemTarget int

const (
	StemVocals StemTarget = iota
	StemAccompaniment
)

type ONNXModelProfile struct {
	Name              string
	SampleRate        int
	FFTSize           int
	HopSize           int
	FreqBins          int
	Frames            int
	Channels          int
	OutputStemCount   int
	DefaultStepFrames int
	UseComplex4       bool
	PrimaryStem       StemTarget
	PrimaryStemIndex  int
	Compensation      float64
	BoundaryFade      int
	LowLatencyPrefill bool
}

var (
	umxVocalsProfile = ONNXModelProfile{
		Name:              DefaultUMXVocalsProfile,
		SampleRate:        ONNXSampleRate,
		FFTSize:           4096,
		HopSize:           1024,
		FreqBins:          2049,
		Frames:            100,
		Channels:          2,
		OutputStemCount:   1,
		DefaultStepFrames: 25,
		UseComplex4:       false,
		PrimaryStem:       StemVocals,
		PrimaryStemIndex:  0,
		Compensation:      1.0,
		BoundaryFade:      0,
		LowLatencyPrefill: false,
	}
	mdxInstHQ3 = ONNXModelProfile{
		Name:              MDXInstHQ3Profile,
		SampleRate:        ONNXSampleRate,
		FFTSize:           6144,
		HopSize:           1024,
		FreqBins:          3072,
		Frames:            256,
		Channels:          4,
		OutputStemCount:   1,
		DefaultStepFrames: 210,
		UseComplex4:       true,
		PrimaryStem:       StemAccompaniment,
		PrimaryStemIndex:  0,
		Compensation:      1.0,
		BoundaryFade:      4096,
		LowLatencyPrefill: false,
	}
	mdx23cVocals = ONNXModelProfile{
		Name:              MDX23CVocalsProfile,
		SampleRate:        ONNXSampleRate,
		FFTSize:           8192,
		HopSize:           1024,
		FreqBins:          4096,
		Frames:            256,
		Channels:          4,
		OutputStemCount:   2,
		DefaultStepFrames: 256,
		UseComplex4:       true,
		PrimaryStem:       StemVocals,
		PrimaryStemIndex:  0,
		Compensation:      1.0,
		BoundaryFade:      4096,
		LowLatencyPrefill: false,
	}
	mdxVocFT = ONNXModelProfile{
		Name:              MDXVocFTProfile,
		SampleRate:        ONNXSampleRate,
		FFTSize:           6144,
		HopSize:           1024,
		FreqBins:          3072,
		Frames:            256,
		Channels:          4,
		OutputStemCount:   1,
		DefaultStepFrames: 210,
		UseComplex4:       true,
		PrimaryStem:       StemVocals,
		PrimaryStemIndex:  0,
		Compensation:      1.0,
		BoundaryFade:      4096,
		LowLatencyPrefill: false,
	}
	mdxKara = ONNXModelProfile{
		Name:              MDXKaraProfile,
		SampleRate:        ONNXSampleRate,
		FFTSize:           4096,
		HopSize:           1024,
		FreqBins:          2048,
		Frames:            256,
		Channels:          4,
		OutputStemCount:   1,
		DefaultStepFrames: 210,
		UseComplex4:       true,
		PrimaryStem:       StemVocals,
		PrimaryStemIndex:  0,
		Compensation:      1.0,
		BoundaryFade:      4096,
		LowLatencyPrefill: false,
	}
	mdxKara2 = ONNXModelProfile{
		Name:              MDXKara2Profile,
		SampleRate:        ONNXSampleRate,
		FFTSize:           4096,
		HopSize:           1024,
		FreqBins:          2048,
		Frames:            256,
		Channels:          4,
		OutputStemCount:   1,
		DefaultStepFrames: 210,
		UseComplex4:       true,
		PrimaryStem:       StemAccompaniment,
		PrimaryStemIndex:  0,
		Compensation:      1.065,
		BoundaryFade:      4096,
		LowLatencyPrefill: false,
	}
)

func ParseONNXProfile(name string) (ONNXModelProfile, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	switch key {
	case "", "umx", "umx-vocals", "open-unmix":
		return umxVocalsProfile, nil
	case "mdx-voc-ft", "voc_ft", "uvr-mdx-net-voc-ft":
		return mdxVocFT, nil
	case "mdx-inst", "mdx-inst-hq3", "zfturbo-mdx-inst-hq3":
		return mdxInstHQ3, nil
	case "mdx23c", "mdx23c-vocals", "zfturbo-mdx23c-vocals":
		return mdx23cVocals, nil
	case "mdx-kara", "uvr-mdxnet-kara":
		return mdxKara, nil
	case "mdx-kara2", "uvr-mdxnet-kara2":
		return mdxKara2, nil
	default:
		return ONNXModelProfile{}, fmt.Errorf(
			"未知 ONNX 模型配置 %q，可选：umx-vocals、mdx-inst-hq3、mdx23c-vocals、mdx-voc-ft、mdx-kara、mdx-kara2", name,
		)
	}
}

var (
	ortInitOnce sync.Once
	ortInitErr  error
)

type ONNXConfig struct {
	Profile            string
	ModelPath          string
	OtherModelPath     string
	RuntimeLibraryPath string
	StepFrames         int
	Compensation       float64
}

type onnxModel struct {
	path         string
	inputName    string
	outputName   string
	shared       *SharedONNXSession
	inputTensor  *ort.Tensor[float32]
	outputTensor *ort.Tensor[float32]
}

type SharedONNXSession struct {
	session     *ort.DynamicAdvancedSession
	inputShape  ort.Shape
	outputShape ort.Shape
	inputName   string
	outputName  string
	profile     ONNXModelProfile
	modelPath   string
}

type ONNX struct {
	profile    ONNXModelProfile
	vocalModel *onnxModel
	otherModel *onnxModel
	stft       *STFTProcessor

	ownsSharedSessions bool

	pending     []audio.Sample
	vocalsQueue []audio.Sample
	accompQueue []audio.Sample
	accumVocals []audio.Sample
	accumAccomp []audio.Sample
	accumNorm   []float64

	windowSize  int
	stepSamples int
	underflows  int

	compensationBits atomic.Uint64

	fallbackBlendSamples int
	pendingFallbackBlend bool
	fallbackBlendVocals  audio.Sample
	fallbackBlendAccomp  audio.Sample

	warmupOnce  sync.Once
	warmupReady chan struct{}

	outputSamples    int64
	queueStartSample int64
	queueEndSample   int64
	timelineAligned  bool
}

func NewONNX(cfg ONNXConfig) (*ONNX, error) {
	profile, err := ParseONNXProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}
	if cfg.ModelPath == "" {
		return nil, errors.New("ONNX 模型路径不能为空")
	}
	if cfg.RuntimeLibraryPath == "" {
		return nil, errors.New("ONNX Runtime 动态库路径不能为空")
	}

	stepFrames := cfg.StepFrames
	if stepFrames <= 0 {
		stepFrames = profile.DefaultStepFrames
	}
	if stepFrames > profile.Frames {
		return nil, fmt.Errorf("ONNX 步进帧数 %d 超过窗口帧数 %d", stepFrames, profile.Frames)
	}
	if err := InitONNXRuntime(cfg.RuntimeLibraryPath); err != nil {
		return nil, err
	}

	mainModel, err := loadONNXModelFull(cfg.ModelPath, profile)
	if err != nil {
		return nil, err
	}

	var otherModel *onnxModel
	if !profile.UseComplex4 && cfg.OtherModelPath != "" {
		if _, statErr := os.Stat(cfg.OtherModelPath); statErr == nil {
			otherModel, err = loadONNXModelFull(cfg.OtherModelPath, profile)
			if err != nil {
				_ = mainModel.Destroy()
				return nil, err
			}
		} else {
			log.Printf("提示：未找到 other 模型 %q，将使用 mix-vocals 计算伴奏", cfg.OtherModelPath)
		}
	} else if profile.UseComplex4 && cfg.OtherModelPath != "" {
		log.Printf("提示：当前配置为复频谱模型，忽略 -onnx-other-model 参数")
	}

	compensation := profile.Compensation
	if cfg.Compensation > 0 {
		compensation = cfg.Compensation
	}

	stft := NewSTFTProcessor(profile.FFTSize, profile.HopSize)
	s := &ONNX{
		profile:              profile,
		vocalModel:           mainModel,
		otherModel:           otherModel,
		stft:                 stft,
		ownsSharedSessions:   true,
		windowSize:           stft.WindowSamples(profile.Frames),
		stepSamples:          stepFrames * profile.HopSize,
		fallbackBlendSamples: maxInt(profile.HopSize, 512),
		warmupReady:          make(chan struct{}),
	}
	s.SetCompensation(compensation)
	return s, nil
}

func NewONNXWithShared(cfg ONNXConfig, sharedVocal, sharedOther *SharedONNXSession) (*ONNX, error) {
	profile, err := ParseONNXProfile(cfg.Profile)
	if err != nil {
		return nil, err
	}
	if cfg.RuntimeLibraryPath != "" {
		if err := InitONNXRuntime(cfg.RuntimeLibraryPath); err != nil {
			return nil, err
		}
	}
	if sharedVocal == nil {
		return nil, errors.New("共享人声 session 不能为空")
	}

	stepFrames := cfg.StepFrames
	if stepFrames <= 0 {
		stepFrames = profile.DefaultStepFrames
	}

	mainModel, err := newONNXModelFromShared(sharedVocal)
	if err != nil {
		return nil, err
	}

	var otherModel *onnxModel
	if sharedOther != nil {
		otherModel, err = newONNXModelFromShared(sharedOther)
		if err != nil {
			_ = mainModel.Destroy()
			return nil, err
		}
	}

	compensation := profile.Compensation
	if cfg.Compensation > 0 {
		compensation = cfg.Compensation
	}

	stft := NewSTFTProcessor(profile.FFTSize, profile.HopSize)
	s := &ONNX{
		profile:              profile,
		vocalModel:           mainModel,
		otherModel:           otherModel,
		stft:                 stft,
		windowSize:           stft.WindowSamples(profile.Frames),
		stepSamples:          stepFrames * profile.HopSize,
		fallbackBlendSamples: maxInt(profile.HopSize, 512),
		warmupReady:          make(chan struct{}),
	}
	s.SetCompensation(compensation)
	return s, nil
}

func (s *ONNX) SetCompensation(v float64) {
	if v <= 0 {
		v = s.profile.Compensation
	}
	s.compensationBits.Store(math.Float64bits(v))
}

func (s *ONNX) compensation() float64 {
	return math.Float64frombits(s.compensationBits.Load())
}

func (s *ONNX) Reset() {
	s.ResetOutput()
	s.warmupOnce = sync.Once{}
	s.warmupReady = make(chan struct{})
	s.fallbackBlendVocals = audio.Sample{}
	s.fallbackBlendAccomp = audio.Sample{}
}

func (s *ONNX) ResetOutput() {
	s.pending = nil
	s.vocalsQueue = nil
	s.accompQueue = nil
	s.accumVocals = nil
	s.accumAccomp = nil
	s.accumNorm = nil
	s.underflows = 0
	s.pendingFallbackBlend = false
	s.queueStartSample = 0
	s.queueEndSample = 0
	s.outputSamples = 0
	s.timelineAligned = false
}

func (s *ONNX) Process(dst *Chunk, mix []audio.Sample) error {
	ensureChunkSize(dst, len(mix))
	s.pending = append(s.pending, mix...)

	for len(s.pending) >= s.windowSize {
		segment := cloneSamples(s.pending[:s.windowSize])
		vocals, accomp, norm, err := s.separateWindow(segment)
		if err != nil {
			return err
		}
		s.accumulateWindow(vocals, accomp, norm)
		s.emitStepToQueues()
		s.pending = s.pending[s.stepSamples:]
	}

	s.alignQueueToOutputTimeline()

	emit := minInt(len(mix), len(s.vocalsQueue))
	if emit > 0 {
		copy(dst.Vocals[:emit], s.vocalsQueue[:emit])
		copy(dst.Accomp[:emit], s.accompQueue[:emit])
		s.vocalsQueue = s.vocalsQueue[emit:]
		s.accompQueue = s.accompQueue[emit:]
		s.queueStartSample += int64(emit)

		s.warmupOnce.Do(func() { close(s.warmupReady) })
		if s.pendingFallbackBlend {
			s.pendingFallbackBlend = false
			n := minInt(emit, s.fallbackBlendSamples)
			for i := 0; i < n; i++ {
				alpha := float64(i+1) / float64(n+1)
				dst.Vocals[i] = lerpSample(s.fallbackBlendVocals, dst.Vocals[i], alpha)
				dst.Accomp[i] = lerpSample(s.fallbackBlendAccomp, dst.Accomp[i], alpha)
			}
		}
	}

	warmedUp := s.isWarmupDone()
	for i := emit; i < len(mix); i++ {
		if !warmedUp {
			dst.Vocals[i] = audio.Sample{}
			dst.Accomp[i] = audio.Sample{}
			continue
		}
		vocals, accomp := s.makeFallbackSplit(mix[i])
		dst.Vocals[i] = vocals
		dst.Accomp[i] = accomp
		s.fallbackBlendVocals = vocals
		s.fallbackBlendAccomp = accomp
	}

	if emit < len(mix) && warmedUp {
		s.pendingFallbackBlend = true
		s.underflows++
		if s.underflows == 1 || s.underflows%200 == 0 {
			log.Printf("警告：ONNX 输出供应不足，累计 %d 次，已使用平滑回退", s.underflows)
		}
	}

	s.outputSamples += int64(len(mix))
	return nil
}

func (s *ONNX) isWarmupDone() bool {
	select {
	case <-s.warmupReady:
		return true
	default:
		return false
	}
}

func (s *ONNX) makeFallbackSplit(mix audio.Sample) (audio.Sample, audio.Sample) {
	if s.profile.UseComplex4 {
		center := 0.5 * (mix[0] + mix[1])
		vocals := audio.Sample{center, center}
		accomp := audio.Sample{mix[0] - center, mix[1] - center}
		return vocals, accomp
	}
	return audio.Sample{}, mix
}

func (s *ONNX) LatencySamples() int {
	return s.windowSize
}

func (s *ONNX) WarmupReady() <-chan struct{} {
	return s.warmupReady
}

func (s *ONNX) PrefillTargetSamples() int {
	if s.profile.LowLatencyPrefill {
		return 2 * defaultChunkSize
	}
	return s.LatencySamples() + 2*defaultChunkSize
}

func (s *ONNX) Drain(dst *Chunk, maxSamples int) (int, error) {
	if len(s.pending) > 0 {
		for len(s.pending) > 0 {
			padded := make([]audio.Sample, s.windowSize)
			copy(padded, s.pending)
			vocals, accomp, norm, err := s.separateWindow(padded)
			if err != nil {
				return 0, err
			}
			s.accumulateWindow(vocals, accomp, norm)
			if len(s.pending) > s.stepSamples {
				s.emitStepToQueues()
				s.pending = s.pending[s.stepSamples:]
				continue
			}
			s.pending = nil
			break
		}
		s.flushAccumToQueues()
	}
	return s.drainQueued(dst, maxSamples), nil
}

func (s *ONNX) Close() error {
	var firstErr error
	if s.vocalModel != nil {
		if s.ownsSharedSessions && s.vocalModel.shared != nil {
			if err := s.vocalModel.shared.Destroy(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if err := s.vocalModel.Destroy(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if s.otherModel != nil {
		if s.ownsSharedSessions && s.otherModel.shared != nil {
			if err := s.otherModel.shared.Destroy(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		if err := s.otherModel.Destroy(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *ONNX) separateWindow(segment []audio.Sample) ([]audio.Sample, []audio.Sample, []float64, error) {
	if s.profile.UseComplex4 {
		return s.separateWindowComplex(segment)
	}
	return s.separateWindowMagnitude(segment)
}

func (s *ONNX) separateWindowMagnitude(segment []audio.Sample) ([]audio.Sample, []audio.Sample, []float64, error) {
	coeffs, magnitudes, err := s.stft.EncodeStereo(segment, s.profile.Frames)
	if err != nil {
		return nil, nil, nil, err
	}

	vocalMasked, err := s.vocalModel.Run(magnitudes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("运行人声 ONNX 模型失败: %w", err)
	}
	vocals, norm, err := s.stft.DecodeStereo(vocalMasked, coeffs, s.profile.Frames)
	if err != nil {
		return nil, nil, nil, err
	}

	if s.otherModel == nil {
		accompMasked := computeResidualMask(magnitudes, vocalMasked)
		accomp, _, err := s.stft.DecodeStereo(accompMasked, coeffs, s.profile.Frames)
		if err != nil {
			return nil, nil, nil, err
		}
		return vocals, accomp, norm, nil
	}

	otherMasked, err := s.otherModel.Run(magnitudes)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("运行 other ONNX 模型失败: %w", err)
	}
	accomp, _, err := s.stft.DecodeStereo(otherMasked, coeffs, s.profile.Frames)
	if err != nil {
		return nil, nil, nil, err
	}
	return vocals, accomp, norm, nil
}

func (s *ONNX) separateWindowComplex(segment []audio.Sample) ([]audio.Sample, []audio.Sample, []float64, error) {
	features, err := s.stft.EncodeStereoComplex4(segment, s.profile.Frames, s.profile.FreqBins)
	if err != nil {
		return nil, nil, nil, err
	}

	estimated, err := s.vocalModel.Run(features)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("运行 MDX ONNX 模型失败: %w", err)
	}
	rawEstimated := make([]float32, len(estimated))
	copy(rawEstimated, estimated)
	if comp := s.compensation(); comp != 1.0 {
		scale := float32(comp)
		for i := range estimated {
			estimated[i] *= scale
		}
	}

	primaryComplex := estimated
	if s.profile.OutputStemCount > 1 {
		primaryComplex, err = selectComplexStem(
			estimated,
			s.profile.OutputStemCount,
			s.profile.PrimaryStemIndex,
			s.profile.Channels,
			s.profile.FreqBins,
			s.profile.Frames,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		secondaryIdx := 1 - s.profile.PrimaryStemIndex
		secondaryComplex, err := selectComplexStem(
			estimated,
			s.profile.OutputStemCount,
			secondaryIdx,
			s.profile.Channels,
			s.profile.FreqBins,
			s.profile.Frames,
		)
		if err != nil {
			return nil, nil, nil, err
		}

		if s.profile.PrimaryStem == StemVocals {
			vocals, norm, err := s.stft.DecodeStereoComplex4(primaryComplex, s.profile.Frames, s.profile.FreqBins)
			if err != nil {
				return nil, nil, nil, err
			}
			accomp, _, err := s.stft.DecodeStereoComplex4(secondaryComplex, s.profile.Frames, s.profile.FreqBins)
			if err != nil {
				return nil, nil, nil, err
			}
			return vocals, accomp, norm, nil
		}
		accomp, norm, err := s.stft.DecodeStereoComplex4(primaryComplex, s.profile.Frames, s.profile.FreqBins)
		if err != nil {
			return nil, nil, nil, err
		}
		vocals, _, err := s.stft.DecodeStereoComplex4(secondaryComplex, s.profile.Frames, s.profile.FreqBins)
		if err != nil {
			return nil, nil, nil, err
		}
		return vocals, accomp, norm, nil
	}

	primary, norm, err := s.stft.DecodeStereoComplex4(primaryComplex, s.profile.Frames, s.profile.FreqBins)
	if err != nil {
		return nil, nil, nil, err
	}

	residualComplex := make([]float32, len(features))
	for i := range features {
		residualComplex[i] = features[i] - rawEstimated[i]
	}
	secondary, _, err := s.stft.DecodeStereoComplex4(residualComplex, s.profile.Frames, s.profile.FreqBins)
	if err != nil {
		return nil, nil, nil, err
	}

	if s.profile.PrimaryStem == StemVocals {
		return primary, secondary, norm, nil
	}
	return secondary, primary, norm, nil
}

func selectComplexStem(features []float32, stemCount, stemIndex, channels, freqBins, frames int) ([]float32, error) {
	if stemCount <= 1 {
		return features, nil
	}
	if stemIndex < 0 || stemIndex >= stemCount {
		return nil, fmt.Errorf("复频谱输出 stem 下标越界：%d（stemCount=%d）", stemIndex, stemCount)
	}

	perStem := channels * freqBins * frames
	expected := stemCount * perStem
	if len(features) != expected {
		return nil, fmt.Errorf(
			"复频谱输出长度不匹配：期望 %d（%d*%d*%d*%d），实际 %d",
			expected, stemCount, channels, freqBins, frames, len(features),
		)
	}

	start := stemIndex * perStem
	out := make([]float32, perStem)
	copy(out, features[start:start+perStem])
	return out, nil
}

func (s *ONNX) drainQueued(dst *Chunk, maxSamples int) int {
	if maxSamples <= 0 {
		return 0
	}
	s.alignQueueToOutputTimeline()

	ensureChunkSize(dst, maxSamples)
	emit := minInt(maxSamples, len(s.vocalsQueue))
	if emit > 0 {
		copy(dst.Vocals[:emit], s.vocalsQueue[:emit])
		copy(dst.Accomp[:emit], s.accompQueue[:emit])
		s.vocalsQueue = s.vocalsQueue[emit:]
		s.accompQueue = s.accompQueue[emit:]
		s.queueStartSample += int64(emit)
		s.outputSamples += int64(emit)
	}
	for i := emit; i < maxSamples; i++ {
		dst.Vocals[i] = audio.Sample{}
		dst.Accomp[i] = audio.Sample{}
	}
	return emit
}

func (s *ONNX) alignQueueToOutputTimeline() {
	if s.queueStartSample >= s.outputSamples {
		return
	}
	if len(s.vocalsQueue) == 0 {
		s.queueStartSample = s.outputSamples
		if s.queueEndSample < s.queueStartSample {
			s.queueEndSample = s.queueStartSample
		}
		return
	}

	if s.timelineAligned {
		s.queueStartSample = s.outputSamples
		return
	}

	stale := s.outputSamples - s.queueStartSample
	if stale <= 0 {
		return
	}
	drop := minInt(len(s.vocalsQueue), int(minInt64(stale, int64(len(s.vocalsQueue)))))
	if drop <= 0 {
		return
	}
	s.vocalsQueue = s.vocalsQueue[drop:]
	s.accompQueue = s.accompQueue[drop:]
	s.queueStartSample += int64(drop)
	s.timelineAligned = true
}

func InitONNXRuntime(runtimeLibraryPath string) error {
	ortInitOnce.Do(func() {
		ort.SetSharedLibraryPath(runtimeLibraryPath)
		ortInitErr = ort.InitializeEnvironment()
	})
	if ortInitErr != nil {
		return fmt.Errorf("初始化 ONNX Runtime 失败: %w", ortInitErr)
	}
	return nil
}

func LoadSharedONNXSession(modelPath string, profile ONNXModelProfile) (*SharedONNXSession, error) {
	inputs, outputs, err := ort.GetInputOutputInfo(modelPath)
	if err != nil {
		return nil, fmt.Errorf("读取 ONNX 输入输出信息失败: %w", err)
	}
	if len(inputs) != 1 || len(outputs) != 1 {
		return nil, fmt.Errorf("当前仅支持单输入单输出模型，实际输入=%d 输出=%d", len(inputs), len(outputs))
	}

	inputShape, err := resolveProfileInputShape(inputs[0].Dimensions, profile)
	if err != nil {
		return nil, fmt.Errorf("解析 ONNX 输入形状失败: %w", err)
	}
	outputShape, err := resolveProfileOutputShape(outputs[0].Dimensions, profile)
	if err != nil {
		return nil, fmt.Errorf("解析 ONNX 输出形状失败: %w", err)
	}

	options, err := buildSessionOptions()
	if err != nil {
		return nil, err
	}
	defer options.Destroy()

	session, err := ort.NewDynamicAdvancedSession(
		modelPath,
		[]string{inputs[0].Name},
		[]string{outputs[0].Name},
		options,
	)
	if err != nil {
		return nil, fmt.Errorf("创建 ONNX Dynamic Session 失败: %w", err)
	}

	return &SharedONNXSession{
		session:     session,
		inputShape:  inputShape,
		outputShape: outputShape,
		inputName:   inputs[0].Name,
		outputName:  outputs[0].Name,
		profile:     profile,
		modelPath:   modelPath,
	}, nil
}

func (s *SharedONNXSession) Destroy() error {
	if s.session != nil {
		return s.session.Destroy()
	}
	return nil
}

func newONNXModelFromShared(shared *SharedONNXSession) (*onnxModel, error) {
	inputTensor, err := ort.NewEmptyTensor[float32](shared.inputShape)
	if err != nil {
		return nil, fmt.Errorf("创建 ONNX 输入张量失败: %w", err)
	}
	outputTensor, err := ort.NewEmptyTensor[float32](shared.outputShape)
	if err != nil {
		_ = inputTensor.Destroy()
		return nil, fmt.Errorf("创建 ONNX 输出张量失败: %w", err)
	}

	return &onnxModel{
		path:         shared.modelPath,
		inputName:    shared.inputName,
		outputName:   shared.outputName,
		shared:       shared,
		inputTensor:  inputTensor,
		outputTensor: outputTensor,
	}, nil
}

func loadONNXModelFull(modelPath string, profile ONNXModelProfile) (*onnxModel, error) {
	shared, err := LoadSharedONNXSession(modelPath, profile)
	if err != nil {
		return nil, err
	}
	m, err := newONNXModelFromShared(shared)
	if err != nil {
		_ = shared.Destroy()
		return nil, err
	}
	return m, nil
}

func buildSessionOptions() (*ort.SessionOptions, error) {
	options, err := ort.NewSessionOptions()
	if err != nil {
		return nil, fmt.Errorf("创建 ONNX Session 选项失败: %w", err)
	}

	if err := options.SetGraphOptimizationLevel(ort.GraphOptimizationLevelEnableAll); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX 图优化级别失败: %w", err)
	}
	if err := options.SetExecutionMode(ort.ExecutionModeParallel); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX 执行模式失败: %w", err)
	}

	intraThreads := runtime.NumCPU() / 2
	if intraThreads < 1 {
		intraThreads = 1
	}
	if intraThreads > 6 {
		intraThreads = 6
	}
	if err := options.SetIntraOpNumThreads(intraThreads); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX IntraOp 线程数失败: %w", err)
	}
	if err := options.SetInterOpNumThreads(1); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX InterOp 线程数失败: %w", err)
	}
	if err := options.AddSessionConfigEntry("session.intra_op.allow_spinning", "0"); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX intra-op 自旋失败: %w", err)
	}
	if err := options.AddSessionConfigEntry("session.inter_op.allow_spinning", "0"); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX inter-op 自旋失败: %w", err)
	}
	if err := options.AddSessionConfigEntry("session.force_spinning_stop", "1"); err != nil {
		_ = options.Destroy()
		return nil, fmt.Errorf("设置 ONNX 强制停止自旋失败: %w", err)
	}
	return options, nil
}

func resolveProfileInputShape(shape ort.Shape, profile ONNXModelProfile) (ort.Shape, error) {
	if len(shape) != 4 {
		return nil, fmt.Errorf("输入形状必须是 4 维，当前是 %s", shape.String())
	}
	resolved := shape.Clone()
	expected := []int64{1, int64(profile.Channels), int64(profile.FreqBins), int64(profile.Frames)}
	for i := range resolved {
		if i == 0 {
			if resolved[i] <= 0 {
				resolved[i] = 1
				continue
			}
			if resolved[i] != 1 {
				return nil, fmt.Errorf("输入 batch 维必须是 1，当前是 %d", resolved[i])
			}
			continue
		}
		if resolved[i] <= 0 {
			resolved[i] = expected[i]
			continue
		}
		if resolved[i] != expected[i] {
			return nil, fmt.Errorf("输入第 %d 维与配置不一致：模型=%d 配置=%d", i, resolved[i], expected[i])
		}
	}
	if err := resolved.Validate(); err != nil {
		return nil, err
	}
	return resolved, nil
}

func resolveProfileOutputShape(shape ort.Shape, profile ONNXModelProfile) (ort.Shape, error) {
	expectedRank := 4
	expected := []int64{1, int64(profile.Channels), int64(profile.FreqBins), int64(profile.Frames)}
	if profile.OutputStemCount > 1 {
		expectedRank = 5
		expected = []int64{
			1,
			int64(profile.OutputStemCount),
			int64(profile.Channels),
			int64(profile.FreqBins),
			int64(profile.Frames),
		}
	}
	if len(shape) != expectedRank {
		return nil, fmt.Errorf("输出形状必须是 %d 维，当前是 %s", expectedRank, shape.String())
	}
	resolved := shape.Clone()
	for i := range resolved {
		if i == 0 {
			if resolved[i] <= 0 {
				resolved[i] = 1
				continue
			}
			if resolved[i] != 1 {
				return nil, fmt.Errorf("输出 batch 维必须是 1，当前是 %d", resolved[i])
			}
			continue
		}
		if resolved[i] <= 0 {
			resolved[i] = expected[i]
			continue
		}
		if resolved[i] != expected[i] {
			return nil, fmt.Errorf("输出第 %d 维与配置不一致：模型=%d 配置=%d", i, resolved[i], expected[i])
		}
	}
	if err := resolved.Validate(); err != nil {
		return nil, err
	}
	return resolved, nil
}

func (m *onnxModel) Run(input []float32) ([]float32, error) {
	want := len(m.inputTensor.GetData())
	if len(input) != want {
		return nil, fmt.Errorf("ONNX 输入长度不匹配：期望 %d，实际 %d", want, len(input))
	}
	copy(m.inputTensor.GetData(), input)
	m.outputTensor.ZeroContents()

	err := m.shared.session.Run([]ort.Value{m.inputTensor}, []ort.Value{m.outputTensor})

	if err != nil {
		return nil, err
	}
	out := make([]float32, len(m.outputTensor.GetData()))
	copy(out, m.outputTensor.GetData())
	return out, nil
}

func (m *onnxModel) Destroy() error {
	var firstErr error
	if m.outputTensor != nil {
		if err := m.outputTensor.Destroy(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	if m.inputTensor != nil {
		if err := m.inputTensor.Destroy(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (s *ONNX) accumulateWindow(vocals, accomp []audio.Sample, norm []float64) {
	windowLen := len(vocals)
	if len(s.accumVocals) < windowLen {
		extra := windowLen - len(s.accumVocals)
		s.accumVocals = append(s.accumVocals, make([]audio.Sample, extra)...)
		s.accumAccomp = append(s.accumAccomp, make([]audio.Sample, extra)...)
		s.accumNorm = append(s.accumNorm, make([]float64, extra)...)
	}
	for i := 0; i < windowLen; i++ {
		s.accumVocals[i][0] += vocals[i][0]
		s.accumVocals[i][1] += vocals[i][1]
		s.accumAccomp[i][0] += accomp[i][0]
		s.accumAccomp[i][1] += accomp[i][1]
		s.accumNorm[i] += norm[i]
	}
}

func (s *ONNX) emitStepToQueues() {
	emit := minInt(s.stepSamples, len(s.accumVocals))
	if emit <= 0 {
		return
	}

	const normFloor = 1e-8
	vocalsOut := make([]audio.Sample, emit)
	accompOut := make([]audio.Sample, emit)
	for i := 0; i < emit; i++ {
		n := s.accumNorm[i]
		if n < normFloor {
			vocalsOut[i] = audio.Sample{}
			accompOut[i] = audio.Sample{}
		} else {
			vocalsOut[i][0] = s.accumVocals[i][0] / n
			vocalsOut[i][1] = s.accumVocals[i][1] / n
			accompOut[i][0] = s.accumAccomp[i][0] / n
			accompOut[i][1] = s.accumAccomp[i][1] / n
		}
	}

	s.vocalsQueue = append(s.vocalsQueue, vocalsOut...)
	s.accompQueue = append(s.accompQueue, accompOut...)
	s.queueEndSample += int64(emit)

	s.accumVocals = shiftSamplesLeft(s.accumVocals, emit)
	s.accumAccomp = shiftSamplesLeft(s.accumAccomp, emit)
	s.accumNorm = shiftFloat64Left(s.accumNorm, emit)
}

func (s *ONNX) flushAccumToQueues() {
	appended := len(s.accumVocals)
	if appended > 0 {
		const normFloor = 1e-8
		normalized := make([]audio.Sample, appended)
		normalizedA := make([]audio.Sample, appended)
		for i := 0; i < appended; i++ {
			n := s.accumNorm[i]
			if n < normFloor {
				normalized[i] = s.accumVocals[i]
				normalizedA[i] = s.accumAccomp[i]
			} else {
				normalized[i][0] = s.accumVocals[i][0] / n
				normalized[i][1] = s.accumVocals[i][1] / n
				normalizedA[i][0] = s.accumAccomp[i][0] / n
				normalizedA[i][1] = s.accumAccomp[i][1] / n
			}
		}
		s.vocalsQueue = append(s.vocalsQueue, normalized...)
		s.accompQueue = append(s.accompQueue, normalizedA...)
		s.queueEndSample += int64(appended)
	}
	s.accumVocals = nil
	s.accumAccomp = nil
	s.accumNorm = nil
}

func lerpSample(a, b audio.Sample, t float64) audio.Sample {
	return audio.Sample{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
	}
}
