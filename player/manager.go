package player

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/xiaowumin-mark/VoxBackend/audio"
	"github.com/xiaowumin-mark/VoxBackend/dsp"
	"github.com/xiaowumin-mark/VoxBackend/separator"
)

type PlaylistCmd int

const (
	CmdNext PlaylistCmd = iota
	CmdPrev
	CmdJump
	CmdPlay
	CmdPlayIndex
	CmdAddTracks
	CmdRemoveTrack
	CmdMoveTrack
	CmdClearPlaylist
	CmdSetTrack
	CmdSetTrackMeta
)

type PlayerCmd struct {
	Type      PlaylistCmd
	Index     int
	Target    int
	Tracks    []Track
	Track     Track
	MetaKey   string
	MetaValue any
}

type PipelineBundle struct {
	pipeline    *Pipeline
	streamer    beep.Streamer
	separator   separator.Separator
	vocalChain  *dsp.Chain
	accompChain *dsp.Chain
	totalSample int64
	index       int
	track       Track
}

func (b *PipelineBundle) Close() {
	if b == nil {
		return
	}
	if b.pipeline != nil {
		b.pipeline.Stop()
	}
}

type bundleResult struct {
	index  int
	bundle *PipelineBundle
	err    error
}

type playlistSeekRequest struct {
	sampleOffset int64
}

type PlaylistManager struct {
	ctx    context.Context
	cfg    Config
	tracks []Track
	emit   func(Event)

	currentIndex int
	sampleRate   int

	crossfadeSamples int64
	prewarmSamples   int64

	measuredWarmupSamples atomic.Int64

	current     *PipelineBundle
	currentPos  int64
	next        *PipelineBundle
	nextReadyCh chan bundleResult
	loadingNext atomic.Bool
	needsLoad   bool

	hardCutReadyCh    chan bundleResult
	pendingHardCutIdx int
	seekReqCh         chan playlistSeekRequest

	isCrossfading bool
	forceHardCut  bool
	stopped       bool

	nextCrossfadeConsumed int64

	cmdCh      chan PlayerCmd
	lastSwitch time.Time
	cmdMu      sync.Mutex
	mu         sync.Mutex
}

func NewPlaylistManager(ctx context.Context, cfg Config, tracks []Track, sampleRate int, emit func(Event)) *PlaylistManager {
	if ctx == nil {
		ctx = context.Background()
	}
	if emit == nil {
		emit = func(Event) {}
	}
	cf := int64(cfg.Crossfade.Seconds() * float64(sampleRate))
	ps := int64(cfg.Prewarm.Seconds() * float64(sampleRate))
	return &PlaylistManager{
		ctx:               ctx,
		cfg:               cfg,
		tracks:            append([]Track{}, tracks...),
		currentIndex:      0,
		sampleRate:        sampleRate,
		crossfadeSamples:  cf,
		prewarmSamples:    ps,
		cmdCh:             make(chan PlayerCmd, 16),
		nextReadyCh:       make(chan bundleResult, 1),
		hardCutReadyCh:    make(chan bundleResult, 1),
		seekReqCh:         make(chan playlistSeekRequest, 1),
		pendingHardCutIdx: -1,
		emit:              emit,
	}
}

func (pm *PlaylistManager) Start() error {
	if len(pm.tracks) == 0 {
		return nil
	}
	return pm.loadFirstTrack()
}

func (pm *PlaylistManager) effectivePrewarmSamples() int64 {
	if v := pm.measuredWarmupSamples.Load(); v > pm.prewarmSamples {
		return v
	}
	return pm.prewarmSamples
}

func (pm *PlaylistManager) createBundle(index int, reason string) (*PipelineBundle, error) {
	select {
	case <-pm.ctx.Done():
		return nil, pm.ctx.Err()
	default:
	}

	track := pm.tracks[index]
	pm.emitEvent(EventTrackLoading, index, track.Path, reason)

	streamer, format, err := audio.DecodeFile(track.Path)
	if err != nil {
		return nil, err
	}

	playbackRate := beep.SampleRate(pm.sampleRate)
	if pm.cfg.SeparatorMode == SeparatorONNX {
		profile, err := separator.ParseONNXProfile(pm.cfg.ONNX.Profile)
		if err != nil {
			_ = streamer.Close()
			return nil, err
		}
		playbackRate = beep.SampleRate(profile.SampleRate)
	}

	pm.emitEvent(EventSeparatorCreating, index, track.Path, "创建分离器")
	sep, err := pm.newSeparator(int(playbackRate))
	if err != nil {
		_ = streamer.Close()
		return nil, err
	}

	ringCapacity := maxInt(2*ChunkSize*BufferChunks, 2*(PrefillTargetSamples(sep)+4*ChunkSize))
	ring := audio.NewRing(ringCapacity)
	mixer := audio.NewMixer(pm.cfg.VocalGain, pm.cfg.MasterVolume, int(playbackRate), float64(pm.cfg.VocalGainRamp)/float64(time.Millisecond))
	pipeline := NewPipeline(streamer, streamer, sep, ring, format.SampleRate, playbackRate)

	var rt beep.Streamer
	var vocalChain, accompChain *dsp.Chain
	mode := pm.cfg.DSP.Mode
	if mode == "" {
		mode = DSPModeAuto
	}
	if mode != DSPModeOff {
		vocalChain = dsp.PresetVocalChain(int(playbackRate))
		accompChain = dsp.PresetAccompChain(int(playbackRate))
		rt = newDSPStreamer(ring, mixer, vocalChain, accompChain, mode)
	} else {
		rt = audio.NewRealtimeStreamer(ring, mixer)
	}

	pm.emitEvent(EventPipelineStarted, index, track.Path, "启动音频处理流水线")
	go pipeline.Run()

	prefill := PrefillTargetSamples(sep)
	pm.emitEvent(EventWarmupWaiting, index, track.Path, "等待分离器预热")
	if err := waitForWarmup(pm.ctx, sep, ring, prefill); err != nil {
		pipeline.Stop()
		return nil, err
	}
	pm.emitEvent(EventWarmupReady, index, track.Path, "分离器预热完成")

	rawLen := int64(streamer.Len())
	var totalSample int64
	if format.SampleRate == playbackRate {
		totalSample = rawLen
	} else {
		totalSample = int64(float64(rawLen) * float64(playbackRate) / float64(format.SampleRate))
	}

	pm.tracks[index].Duration = samplesToDuration(totalSample, pm.sampleRate)

	bundle := &PipelineBundle{
		pipeline:    pipeline,
		streamer:    rt,
		separator:   sep,
		vocalChain:  vocalChain,
		accompChain: accompChain,
		totalSample: totalSample,
		index:       index,
		track:       cloneTrack(track),
	}
	pm.emitEvent(EventTrackReady, index, track.Path, "歌曲加载完成")
	return bundle, nil
}

func (pm *PlaylistManager) newSeparator(sampleRate int) (separator.Separator, error) {
	switch pm.cfg.SeparatorMode {
	case "", SeparatorFake:
		return separator.NewFake(pm.cfg.FakeVocalAmount, pm.cfg.FakeAccompBleed, pm.cfg.AILatency), nil
	case SeparatorONNX:
		profile, err := separator.ParseONNXProfile(pm.cfg.ONNX.Profile)
		if err != nil {
			return nil, err
		}
		if sampleRate != profile.SampleRate {
			return nil, fmt.Errorf("onnx(%s) 要求输入采样率 %d Hz，当前是 %d Hz", profile.Name, profile.SampleRate, sampleRate)
		}
		return separator.NewONNX(pm.cfg.ONNX)
	default:
		return nil, fmt.Errorf("未知分离器类型 %q", pm.cfg.SeparatorMode)
	}
}

func waitForWarmup(ctx context.Context, sep separator.Separator, ring *audio.Ring, prefill int) error {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-sep.WarmupReady():
			ring.DiscardAll()
			if err := ring.WaitForAtLeast(prefill * 2); err != nil {
				return fmt.Errorf("预填充音频失败: %w", err)
			}
			return nil
		case <-ticker.C:
			if ring.Closed() {
				if err := ring.Err(); err != nil {
					return err
				}
				return audio.ErrRingClosed
			}
		}
	}
}

func (pm *PlaylistManager) loadNextAsync(index int) {
	if index < 0 || index >= len(pm.tracks) {
		pm.loadingNext.Store(false)
		return
	}
	pm.emitEvent(EventPrewarmStarted, index, pm.tracks[index].Path, "后台预热下一首")
	t0 := time.Now()
	bundle, err := pm.createBundle(index, "后台预热")
	elapsed := time.Since(t0)
	if err == nil {
		measured := int64(float64(elapsed.Seconds()) * float64(pm.sampleRate) * 1.2)
		if measured > pm.measuredWarmupSamples.Load() {
			pm.measuredWarmupSamples.Store(measured)
		}
		pm.emitEvent(EventPrewarmReady, index, pm.tracks[index].Path, "下一首预热完成")
	}
	pm.deliverNext(bundleResult{index: index, bundle: bundle, err: err})
}

func (pm *PlaylistManager) deliverNext(result bundleResult) {
	pm.mu.Lock()
	stopped := pm.stopped
	pm.mu.Unlock()
	if stopped {
		if result.bundle != nil {
			result.bundle.Close()
		}
		pm.loadingNext.Store(false)
		return
	}

	if result.err != nil {
		pm.emitError(result.index, pm.tracks[result.index].Path, fmt.Errorf("预热下一首失败: %w", result.err))
		pm.loadingNext.Store(false)
		return
	}
	select {
	case pm.nextReadyCh <- result:
	default:
		if result.bundle != nil {
			result.bundle.Close()
		}
	}
}

func (pm *PlaylistManager) PostCommand(cmd PlayerCmd) {
	pm.cmdMu.Lock()
	defer pm.cmdMu.Unlock()
	if time.Since(pm.lastSwitch) < 800*time.Millisecond && isSwitchCmd(cmd.Type) {
		return
	}
	pm.lastSwitch = time.Now()

	select {
	case pm.cmdCh <- cmd:
	default:
	}
}

func isSwitchCmd(t PlaylistCmd) bool {
	return t == CmdNext || t == CmdPrev || t == CmdJump
}

func (pm *PlaylistManager) AddTracks(tracks []Track) {
	for _, t := range tracks {
		select {
		case pm.cmdCh <- PlayerCmd{Type: CmdAddTracks, Tracks: []Track{t}}:
		default:
		}
	}
}

func (pm *PlaylistManager) RemoveTrack(index int) {
	select {
	case pm.cmdCh <- PlayerCmd{Type: CmdRemoveTrack, Index: index}:
	default:
	}
}

func (pm *PlaylistManager) MoveTrack(from, to int) {
	select {
	case pm.cmdCh <- PlayerCmd{Type: CmdMoveTrack, Index: from, Target: to}:
	default:
	}
}

func (pm *PlaylistManager) ClearPlaylist() {
	select {
	case pm.cmdCh <- PlayerCmd{Type: CmdClearPlaylist}:
	default:
	}
}

func (pm *PlaylistManager) SetTrack(index int, track Track) {
	select {
	case pm.cmdCh <- PlayerCmd{Type: CmdSetTrack, Index: index, Track: track}:
	default:
	}
}

func (pm *PlaylistManager) SetTrackMeta(index int, key string, val any) {
	select {
	case pm.cmdCh <- PlayerCmd{Type: CmdSetTrackMeta, Index: index, MetaKey: key, MetaValue: val}:
	default:
	}
}

func (pm *PlaylistManager) Stream(dst [][2]float64) (int, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.stopped {
		return 0, false
	}

	select {
	case req := <-pm.seekReqCh:
		pm.applySeek(req.sampleOffset)
	default:
	}

	select {
	case result := <-pm.hardCutReadyCh:
		pm.consumeHardCutResult(result)
	default:
	}

	select {
	case result := <-pm.nextReadyCh:
		pm.consumeNextResult(result)
	default:
	}

	select {
	case cmd := <-pm.cmdCh:
		pm.processCommandLocked(cmd)
	default:
	}

	if pm.current == nil && (pm.pendingHardCutIdx < 0) {
		if len(pm.tracks) > 0 && pm.needsLoad {
			pm.loadCurrentTrack(0)
			pm.needsLoad = false
		}
	}

	if pm.current == nil {
		for i := range dst {
			dst[i] = [2]float64{}
		}
		return len(dst), true
	}

	n, ok := pm.current.streamer.Stream(dst)
	pm.currentPos += int64(n)

	if !pm.isCrossfading && pm.current.totalSample > 0 &&
		pm.current.totalSample-pm.currentPos <= int64(2*pm.sampleRate) {
		pm.forceHardCut = true
	}

	nextIdx := pm.currentIndex + 1
	if nextIdx < len(pm.tracks) &&
		!pm.isCrossfading &&
		!pm.forceHardCut &&
		!pm.loadingNext.Load() &&
		pm.pendingHardCutIdx < 0 {
		prewarm := pm.effectivePrewarmSamples()
		triggerPoint := pm.current.totalSample - pm.crossfadeSamples - prewarm
		if triggerPoint < 0 {
			triggerPoint = 0
		}
		if pm.currentPos >= triggerPoint {
			pm.loadingNext.Store(true)
			go pm.loadNextAsync(nextIdx)
		}
	}

	posBeforeThisFrame := pm.currentPos - int64(n)
	if !pm.forceHardCut && pm.crossfadeSamples > 0 && posBeforeThisFrame >= pm.current.totalSample-pm.crossfadeSamples {
		if !pm.isCrossfading {
			pm.emitEvent(EventCrossfadeStarted, pm.currentIndex, pm.current.track.Path, "开始淡入淡出")
		}
		pm.isCrossfading = true
		if pm.next != nil {
			temp := make([][2]float64, n)
			nn, _ := pm.next.streamer.Stream(temp)
			pm.nextCrossfadeConsumed += int64(nn)

			for i := 0; i < n; i++ {
				samplePos := posBeforeThisFrame + int64(i)
				fadeLeft := pm.current.totalSample - samplePos
				if fadeLeft < 0 {
					fadeLeft = 0
				}
				progress := 1.0 - float64(fadeLeft)/float64(pm.crossfadeSamples)
				progress = clamp(progress, 0.0, 1.0)

				alphaCurrent := 1.0 - progress
				alphaNext := progress

				dst[i][0] *= alphaCurrent
				dst[i][1] *= alphaCurrent
				if i < nn {
					dst[i][0] += temp[i][0] * alphaNext
					dst[i][1] += temp[i][1] * alphaNext
				}
			}
		} else {
			for i := 0; i < n; i++ {
				samplePos := posBeforeThisFrame + int64(i)
				fadeLeft := pm.current.totalSample - samplePos
				if fadeLeft < 0 {
					fadeLeft = 0
				}
				progress := float64(fadeLeft) / float64(pm.crossfadeSamples)
				progress = clamp(progress, 0.0, 1.0)
				dst[i][0] *= progress
				dst[i][1] *= progress
			}
		}
	}

	if !ok || (pm.current.totalSample > 0 && pm.currentPos >= pm.current.totalSample) {
		pm.current.Close()
		nextIdx := pm.currentIndex + 1
		if pm.next != nil && pm.next.index == nextIdx {
			pm.currentIndex = nextIdx
			pm.current = pm.next
			pm.next = nil
			pm.currentPos = pm.nextCrossfadeConsumed
			pm.nextCrossfadeConsumed = 0
			pm.emitTrackChanged(pm.current)
		} else if nextIdx < len(pm.tracks) {
			pm.currentIndex = nextIdx
			pm.current = nil
			pm.pendingHardCutIdx = nextIdx
			go pm.loadBundleForHardCut(nextIdx)
		} else {
			pm.current = nil
			pm.emitPlaybackIdle()
		}
		pm.resetStateLocked()
	}

	return n, true
}

func (pm *PlaylistManager) processCommandLocked(cmd PlayerCmd) {
	switch cmd.Type {
	case CmdNext, CmdPrev, CmdJump:
		pm.startHardCut(cmd)
	case CmdPlay, CmdPlayIndex:
		if cmd.Type == CmdPlayIndex {
			if cmd.Index >= 0 && cmd.Index < len(pm.tracks) {
				pm.currentIndex = cmd.Index
			}
		}
		if pm.current != nil {
			pm.current.Close()
			pm.current = nil
		}
		if pm.next != nil {
			pm.next.Close()
			pm.next = nil
		}
		pm.drainReadyChannelsLocked()
		if len(pm.tracks) > 0 {
			pm.needsLoad = true
			pm.resetStateLocked()
		}
	case CmdAddTracks:
		wasEmpty := len(pm.tracks) == 0
		for _, t := range cmd.Tracks {
			pm.tracks = append(pm.tracks, t)
		}
		if wasEmpty && len(pm.tracks) > 0 {
			pm.currentIndex = 0
			pm.needsLoad = pm.current == nil
		}
		pm.emitPlaylistChanged()
	case CmdRemoveTrack:
		pm.handleRemoveTrackLocked(cmd.Index)
	case CmdMoveTrack:
		pm.handleMoveTrackLocked(cmd.Index, cmd.Target)
	case CmdClearPlaylist:
		pm.handleClearPlaylistLocked()
	case CmdSetTrack:
		if cmd.Index >= 0 && cmd.Index < len(pm.tracks) {
			pm.tracks[cmd.Index] = cmd.Track
		}
		pm.emitPlaylistChanged()
	case CmdSetTrackMeta:
		if cmd.Index >= 0 && cmd.Index < len(pm.tracks) {
			if pm.tracks[cmd.Index].Meta == nil {
				pm.tracks[cmd.Index].Meta = make(map[string]any)
			}
			pm.tracks[cmd.Index].Meta[cmd.MetaKey] = cmd.MetaValue
		}
		pm.emitPlaylistChanged()
	}
}

func (pm *PlaylistManager) handleRemoveTrackLocked(index int) {
	if index < 0 || index >= len(pm.tracks) {
		return
	}

	if pm.current != nil && index == pm.currentIndex {
		pm.current.Close()
		pm.current = nil
		pm.drainReadyChannelsLocked()
	}

	if pm.next != nil && pm.next.index == index {
		pm.next.Close()
		pm.next = nil
		pm.drainReadyChannelsLocked()
	}

	pm.tracks = append(pm.tracks[:index], pm.tracks[index+1:]...)

	if index < pm.currentIndex {
		pm.currentIndex--
	}
	if pm.currentIndex >= len(pm.tracks) {
		pm.currentIndex = len(pm.tracks) - 1
	}
	if pm.currentIndex < 0 {
		pm.currentIndex = 0
	}

	if pm.current == nil && len(pm.tracks) > 0 && pm.currentIndex >= 0 {
		pm.needsLoad = true
		pm.resetStateLocked()
	}
	pm.emitPlaylistChanged()
}

func (pm *PlaylistManager) handleMoveTrackLocked(from, to int) {
	if from < 0 || from >= len(pm.tracks) || to < 0 || to >= len(pm.tracks) || from == to {
		return
	}

	t := pm.tracks[from]
	pm.tracks = append(pm.tracks[:from], pm.tracks[from+1:]...)
	if to > from {
		to--
	}
	pm.tracks = append(pm.tracks[:to], append([]Track{t}, pm.tracks[to:]...)...)

	if from == pm.currentIndex {
		pm.currentIndex = to
	} else if from < pm.currentIndex && to >= pm.currentIndex {
		pm.currentIndex--
	} else if from > pm.currentIndex && to <= pm.currentIndex {
		pm.currentIndex++
	}

	if pm.next != nil {
		pm.next.Close()
		pm.next = nil
		pm.drainReadyChannelsLocked()
	}

	pm.emitPlaylistChanged()
}

func (pm *PlaylistManager) handleClearPlaylistLocked() {
	if pm.current != nil {
		pm.current.Close()
		pm.current = nil
	}
	if pm.next != nil {
		pm.next.Close()
		pm.next = nil
	}
	pm.drainReadyChannelsLocked()
	pm.tracks = nil
	pm.currentIndex = 0
	pm.resetStateLocked()
	pm.emitPlaylistChanged()
	pm.emitPlaybackIdle()
}

func (pm *PlaylistManager) loadCurrentTrack(sampleOffset int64) {
	bundle, err := pm.createBundle(pm.currentIndex, "开始播放")
	if err != nil {
		pm.emitError(pm.currentIndex, pm.tracks[pm.currentIndex].Path, err)
		pm.currentIndex++
		if pm.currentIndex < len(pm.tracks) {
			pm.loadCurrentTrack(0)
		}
		return
	}
	pm.current = bundle
	pm.currentPos = sampleOffset
	pm.emitTrackChanged(bundle)
}

func (pm *PlaylistManager) loadFirstTrack() error {
	bundle, err := pm.createBundle(pm.currentIndex, "首曲加载")
	if err != nil {
		return err
	}
	pm.current = bundle
	pm.currentPos = 0
	pm.emitTrackChanged(bundle)
	return nil
}

func (pm *PlaylistManager) consumeHardCutResult(result bundleResult) {
	if result.index != pm.pendingHardCutIdx {
		if result.bundle != nil {
			result.bundle.Close()
		}
		return
	}

	if result.err != nil {
		pm.emitError(result.index, pm.tracks[result.index].Path, result.err)
		pm.current = nil
		pm.pendingHardCutIdx = -1
		return
	}

	pm.current = result.bundle
	pm.pendingHardCutIdx = -1
	pm.currentPos = 0
	pm.emitTrackChanged(result.bundle)
}

func (pm *PlaylistManager) consumeNextResult(result bundleResult) {
	expected := pm.currentIndex + 1
	if result.err != nil {
		pm.emitError(result.index, pm.tracks[result.index].Path, result.err)
		pm.loadingNext.Store(false)
		return
	}
	if result.index != expected || pm.pendingHardCutIdx >= 0 || pm.stopped {
		if result.bundle != nil {
			result.bundle.Close()
		}
		pm.loadingNext.Store(false)
		return
	}
	if pm.next != nil {
		pm.next.Close()
	}
	pm.next = result.bundle
	pm.loadingNext.Store(false)
}

func (pm *PlaylistManager) startHardCut(cmd PlayerCmd) {
	targetIdx := pm.currentIndex
	switch cmd.Type {
	case CmdNext:
		targetIdx++
	case CmdPrev:
		targetIdx--
	case CmdJump:
		targetIdx = cmd.Index
	}

	if targetIdx < 0 || targetIdx >= len(pm.tracks) {
		return
	}

	pm.emitEvent(EventHardCutStarted, targetIdx, pm.tracks[targetIdx].Path, "硬切歌曲")

	if pm.current != nil {
		pm.current.Close()
		pm.current = nil
	}
	if pm.next != nil {
		pm.next.Close()
		pm.next = nil
	}
	pm.drainReadyChannelsLocked()

	pm.currentIndex = targetIdx
	pm.pendingHardCutIdx = targetIdx
	pm.resetStateLocked()

	go pm.loadBundleForHardCut(targetIdx)
}

func (pm *PlaylistManager) loadBundleForHardCut(idx int) {
	bundle, err := pm.createBundle(idx, "硬切加载")
	result := bundleResult{index: idx, bundle: bundle, err: err}
	pm.mu.Lock()
	stopped := pm.stopped
	pm.mu.Unlock()
	if stopped {
		if bundle != nil {
			bundle.Close()
		}
		return
	}
	select {
	case pm.hardCutReadyCh <- result:
	default:
		if bundle != nil {
			bundle.Close()
		}
	}
}

func (pm *PlaylistManager) applySeek(sampleOffset int64) {
	if pm.current == nil {
		return
	}
	if sampleOffset < 0 {
		sampleOffset = 0
	}
	if pm.current.totalSample > 0 && sampleOffset > pm.current.totalSample {
		sampleOffset = pm.current.totalSample
	}
	pm.currentPos = sampleOffset
	pm.current.pipeline.ring.DiscardAndReopen()
	pm.emitEvent(EventSeekStarted, pm.currentIndex, pm.current.track.Path, "开始 seek")
	if err := pm.current.pipeline.SeekSamples(sampleOffset); err != nil {
		pm.emitError(pm.currentIndex, pm.current.track.Path, err)
	} else {
		pm.emitEvent(EventSeekFinished, pm.currentIndex, pm.current.track.Path, "seek 完成")
	}
	if pm.current.totalSample > 0 &&
		pm.current.totalSample-sampleOffset <= int64(2*pm.sampleRate) {
		pm.forceHardCut = true
	}
}

func (pm *PlaylistManager) resetStateLocked() {
	pm.isCrossfading = false
	pm.forceHardCut = false
	pm.loadingNext.Store(false)
	pm.drainReadyChannelsLocked()
}

func (pm *PlaylistManager) drainReadyChannelsLocked() {
	for {
		select {
		case result := <-pm.nextReadyCh:
			if result.bundle != nil {
				result.bundle.Close()
			}
		default:
			goto hardCuts
		}
	}
hardCuts:
	for {
		select {
		case result := <-pm.hardCutReadyCh:
			if result.bundle != nil {
				result.bundle.Close()
			}
		default:
			return
		}
	}
}

func (pm *PlaylistManager) Err() error {
	return nil
}

func (pm *PlaylistManager) RequestSeek(sampleOffset int64) {
	select {
	case pm.seekReqCh <- playlistSeekRequest{sampleOffset: sampleOffset}:
	default:
		select {
		case <-pm.seekReqCh:
		default:
		}
		pm.seekReqCh <- playlistSeekRequest{sampleOffset: sampleOffset}
	}
}

func (pm *PlaylistManager) SetVocalGain(v float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.VocalGain = v
	pm.applyToBundlesLocked(func(m *audio.Mixer) {
		m.SetVocalGain(v)
	})
	if pm.cfg.DSP.Mode == DSPModeAuto {
		for _, b := range []*PipelineBundle{pm.current, pm.next} {
			if b != nil {
				if ds, ok := b.streamer.(*dspStreamer); ok {
					ds.OnVocalGainChanged(v)
				}
			}
		}
	} else {
		pm.updateGateThresholdLocked(v)
	}
	pm.emitEvent(EventVocalGainChanged, pm.currentIndex, pm.currentPathLocked(), "人声增益已更新")
}

func (pm *PlaylistManager) SetRampDuration(d time.Duration) {
	if d < 0 {
		d = 0
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.VocalGainRamp = d
	ms := float64(d) / float64(time.Millisecond)
	pm.applyToBundlesLocked(func(m *audio.Mixer) {
		m.SetRampDurationMs(ms)
	})
	pm.emitEvent(EventRampChanged, pm.currentIndex, pm.currentPathLocked(), "人声变化平滑时间已更新")
}

func (pm *PlaylistManager) SetMasterVolume(v float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.MasterVolume = v
	pm.applyToBundlesLocked(func(m *audio.Mixer) {
		m.SetMasterVolume(v)
	})
	pm.emitEvent(EventVolumeChanged, pm.currentIndex, pm.currentPathLocked(), "主音量已更新")
}

func (pm *PlaylistManager) updateGateThresholdLocked(vocalGain float64) {
	baseDB := -40.0
	linkedDB := baseDB + (1.0-vocalGain)*20.0
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		if b == nil || b.vocalChain == nil {
			continue
		}
		for _, p := range b.vocalChain.Processors() {
			if g, ok := p.(*dsp.Gate); ok {
				g.SetThresholdDB(linkedDB)
			}
		}
	}
}

func (pm *PlaylistManager) SetCrossfade(d time.Duration) {
	if d < 0 {
		d = 0
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.Crossfade = d
	pm.crossfadeSamples = int64(d.Seconds() * float64(pm.sampleRate))
	pm.emitEvent(EventCrossfadeChanged, pm.currentIndex, pm.currentPathLocked(), "淡入淡出时长已更新")
}

func (pm *PlaylistManager) SetPrewarm(d time.Duration) {
	if d < 0 {
		d = 0
	}
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.Prewarm = d
	pm.prewarmSamples = int64(d.Seconds() * float64(pm.sampleRate))
	pm.emitEvent(EventPrewarmChanged, pm.currentIndex, pm.currentPathLocked(), "预热提前量已更新")
}

func (pm *PlaylistManager) SetFakeParams(vocalAmount, accompBleed float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.FakeVocalAmount = vocalAmount
	pm.cfg.FakeAccompBleed = accompBleed
	pm.applyToSeparatorsLocked(func(sep separator.Separator) {
		if p, ok := sep.(separator.ParameterizedFake); ok {
			p.SetFakeParams(vocalAmount, accompBleed)
		}
	})
	pm.emitEvent(EventFakeParamsChanged, pm.currentIndex, pm.currentPathLocked(), "fake 分离器参数已更新")
}

func (pm *PlaylistManager) SetONNXCompensation(v float64) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.ONNX.Compensation = v
	pm.applyToSeparatorsLocked(func(sep separator.Separator) {
		if p, ok := sep.(interface{ SetCompensation(float64) }); ok {
			p.SetCompensation(v)
		}
	})
	pm.emitEvent(EventONNXCompChanged, pm.currentIndex, pm.currentPathLocked(), "ONNX 补偿系数已更新")
}

func (pm *PlaylistManager) SetDSPMode(mode DSPMode) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.cfg.DSP.Mode = mode
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		if b == nil || b.streamer == nil {
			continue
		}
		if ds, ok := b.streamer.(*dspStreamer); ok {
			ds.SetMode(mode)
		}
	}
	pm.emitEvent(EventDSPChanged, pm.currentIndex, pm.currentPathLocked(), "DSP 模式已更新")
}

func (pm *PlaylistManager) SetDSPEnabled(enabled bool) {
	if enabled {
		pm.SetDSPMode(DSPModeOn)
	} else {
		pm.SetDSPMode(DSPModeOff)
	}
}

func (pm *PlaylistManager) Stop() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.stopped {
		return
	}
	pm.stopped = true
	if pm.current != nil {
		pm.current.Close()
		pm.current = nil
	}
	if pm.next != nil {
		pm.next.Close()
		pm.next = nil
	}
	pm.drainReadyChannelsLocked()
}

func (pm *PlaylistManager) Snapshot(paused bool) State {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state := State{
		TrackIndex:  -1,
		Paused:      paused,
		Loading:     pm.loadingNext.Load() || pm.pendingHardCutIdx >= 0,
		Crossfading: pm.isCrossfading,
		Volume:      pm.cfg.MasterVolume,
		VocalGain:   pm.cfg.VocalGain,
		VocalRamp:   pm.cfg.VocalGainRamp,
		Crossfade:   pm.cfg.Crossfade,
		Prewarm:     pm.cfg.Prewarm,
		TrackCount:  len(pm.tracks),
		Idle:        pm.current == nil && pm.pendingHardCutIdx < 0,
		Finished:    false,
	}
	if pm.current != nil {
		state.TrackIndex = pm.current.index
		state.TrackPath = pm.current.track.Path
		state.Idle = false
		t := cloneTrack(pm.current.track)
		state.Track = &t
		state.Position = samplesToDuration(pm.currentPos, pm.sampleRate)
		state.Duration = samplesToDuration(pm.current.totalSample, pm.sampleRate)
	} else if pm.currentIndex >= 0 && pm.currentIndex < len(pm.tracks) {
		state.TrackPath = pm.tracks[pm.currentIndex].Path
		t := cloneTrack(pm.tracks[pm.currentIndex])
		state.Track = &t
	}
	return state
}

func (pm *PlaylistManager) applyToBundlesLocked(fn func(*audio.Mixer)) {
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		if b == nil || b.streamer == nil {
			continue
		}
		if sm, ok := b.streamer.(interface{ Mixer() *audio.Mixer }); ok {
			fn(sm.Mixer())
		}
	}
}

func (pm *PlaylistManager) applyToSeparatorsLocked(fn func(sep separator.Separator)) {
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		if b != nil && b.separator != nil {
			fn(b.separator)
		}
	}
}

func (pm *PlaylistManager) currentPathLocked() string {
	if pm.current != nil {
		return pm.current.track.Path
	}
	if pm.currentIndex >= 0 && pm.currentIndex < len(pm.tracks) {
		return pm.tracks[pm.currentIndex].Path
	}
	return ""
}

func (pm *PlaylistManager) emitTrackChanged(bundle *PipelineBundle) {
	if bundle == nil {
		return
	}
	t := cloneTrack(bundle.track)
	ev := newEvent(EventTrackChanged, "当前歌曲已切换")
	ev.TrackIndex = bundle.index
	ev.TrackPath = bundle.track.Path
	ev.Track = &t
	pm.emit(ev)
}

func (pm *PlaylistManager) emitPlaylistChanged() {
	pm.emit(newEvent(EventPlaylistChanged, "播放列表已更新"))
}

func (pm *PlaylistManager) emitPlaybackIdle() {
	pm.emit(newEvent(EventPlaybackIdle, "播放器空闲"))
}

func (pm *PlaylistManager) emitError(index int, path string, err error) {
	if err == nil {
		return
	}
	ev := newEvent(EventError, err.Error())
	ev.TrackIndex = index
	ev.TrackPath = path
	ev.Err = err
	pm.emit(ev)
}

func (pm *PlaylistManager) emitEvent(t EventType, index int, path, message string) {
	ev := newEvent(t, message)
	ev.TrackIndex = index
	ev.TrackPath = path
	pm.emit(ev)
}

func samplesToDuration(samples int64, sampleRate int) time.Duration {
	if sampleRate <= 0 || samples <= 0 {
		return 0
	}
	return time.Duration(float64(samples) / float64(sampleRate) * float64(time.Second))
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
