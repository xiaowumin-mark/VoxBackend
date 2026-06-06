package player

import (
	"context"
	"fmt"
	"math/rand"
	"os"
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
	CmdShuffleUpcoming
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
	switcher    *switchableStreamer
	ring        *audio.Ring
	mixer       *audio.Mixer
	separator   separator.Separator
	vocalChain  *dsp.Chain
	accompChain *dsp.Chain
	totalSample int64
	index       int
	track       Track
}

type switchableStreamer struct {
	mu      sync.Mutex
	current beep.Streamer
}

func newSwitchableStreamer(current beep.Streamer) *switchableStreamer {
	return &switchableStreamer{current: current}
}

func (s *switchableStreamer) Stream(samples [][2]float64) (int, bool) {
	s.mu.Lock()
	current := s.current
	s.mu.Unlock()
	if current == nil {
		return 0, false
	}
	return current.Stream(samples)
}

func (s *switchableStreamer) Err() error {
	s.mu.Lock()
	current := s.current
	s.mu.Unlock()
	if current == nil {
		return nil
	}
	return current.Err()
}

func (s *switchableStreamer) Set(current beep.Streamer) {
	s.mu.Lock()
	s.current = current
	s.mu.Unlock()
}

func (s *switchableStreamer) Current() beep.Streamer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
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
	epoch  uint64
	track  Track
	bundle *PipelineBundle
	err    error
}

type playlistSeekRequest struct {
	sampleOffset int64
}

type seekResult struct {
	err error
}

type PlaylistManager struct {
	ctx         context.Context
	loadCtx     context.Context
	cancelLoads context.CancelFunc
	cfg         Config
	tracks      []Track
	emit        func(Event)

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
	failedPrewarmIndex    int
	failedPrewarmPath     string

	seeking         atomic.Bool
	seekDone        atomic.Bool
	seekErr         atomic.Value
	loadEpoch       uint64
	seekRollbackPos int64

	sharedVocalSession *separator.SharedONNXSession
	sharedOtherSession *separator.SharedONNXSession

	sessionMu  sync.Mutex
	loadWG     sync.WaitGroup
	pipelineWG sync.WaitGroup

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
	loadCtx, cancelLoads := context.WithCancel(ctx)
	return &PlaylistManager{
		ctx:                ctx,
		loadCtx:            loadCtx,
		cancelLoads:        cancelLoads,
		cfg:                cfg,
		tracks:             cloneTracks(tracks),
		currentIndex:       0,
		sampleRate:         sampleRate,
		crossfadeSamples:   cf,
		prewarmSamples:     ps,
		cmdCh:              make(chan PlayerCmd, 16),
		nextReadyCh:        make(chan bundleResult, 1),
		hardCutReadyCh:     make(chan bundleResult, 1),
		seekReqCh:          make(chan playlistSeekRequest, 1),
		pendingHardCutIdx:  -1,
		failedPrewarmIndex: -1,
		emit:               emit,
	}
}

func (pm *PlaylistManager) Start() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if len(pm.tracks) == 0 {
		return nil
	}
	pm.pendingHardCutIdx = pm.currentIndex
	pm.loadingNext.Store(true)
	pm.startHardCutLoadLocked(pm.currentIndex)
	return nil
}

func (pm *PlaylistManager) ensureSharedSessions(cfg Config) error {
	pm.sessionMu.Lock()
	defer pm.sessionMu.Unlock()

	if pm.sharedVocalSession != nil {
		return nil
	}
	if cfg.ONNX.RuntimeLibraryPath != "" {
		if err := separator.InitONNXRuntime(cfg.ONNX.RuntimeLibraryPath); err != nil {
			return err
		}
	}
	profile, err := separator.ParseONNXProfile(cfg.ONNX.Profile)
	if err != nil {
		return err
	}
	shared, err := separator.LoadSharedONNXSession(cfg.ONNX.ModelPath, profile)
	if err != nil {
		return err
	}
	pm.sharedVocalSession = shared

	if !profile.UseComplex4 && cfg.ONNX.OtherModelPath != "" {
		if _, statErr := os.Stat(cfg.ONNX.OtherModelPath); statErr == nil {
			other, otherErr := separator.LoadSharedONNXSession(cfg.ONNX.OtherModelPath, profile)
			if otherErr != nil {
				_ = pm.sharedVocalSession.Destroy()
				pm.sharedVocalSession = nil
				return otherErr
			}
			pm.sharedOtherSession = other
		}
	}
	return nil
}

func (pm *PlaylistManager) effectivePrewarmSamples() int64 {
	if v := pm.measuredWarmupSamples.Load(); v > pm.prewarmSamples {
		return v
	}
	return pm.prewarmSamples
}

func (pm *PlaylistManager) createBundle(ctx context.Context, cfg Config, index int, track Track, reason string) (*PipelineBundle, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	pm.emitEvent(EventTrackLoading, index, track.Path, reason)

	streamer, format, err := audio.DecodeFile(track.Path)
	if err != nil {
		return nil, err
	}
	select {
	case <-ctx.Done():
		_ = streamer.Close()
		return nil, ctx.Err()
	default:
	}

	playbackRate := beep.SampleRate(pm.sampleRate)
	if cfg.SeparatorMode == SeparatorONNX {
		profile, err := separator.ParseONNXProfile(cfg.ONNX.Profile)
		if err != nil {
			_ = streamer.Close()
			return nil, err
		}
		playbackRate = beep.SampleRate(profile.SampleRate)
	}

	pm.emitEvent(EventSeparatorCreating, index, track.Path, "创建分离器")
	sep, err := pm.newSeparator(cfg, int(playbackRate))
	if err != nil {
		_ = streamer.Close()
		return nil, err
	}
	select {
	case <-ctx.Done():
		_ = sep.Close()
		_ = streamer.Close()
		return nil, ctx.Err()
	default:
	}

	crossfadeSamples := int64(cfg.Crossfade.Seconds() * float64(pm.sampleRate))
	ringCapacity := maxInt(2*ChunkSize*BufferChunks, 2*(PrefillTargetSamples(sep)+4*ChunkSize)+2*int(crossfadeSamples))
	ring := audio.NewRing(ringCapacity)
	mixer := audio.NewMixer(cfg.VocalGain, cfg.MasterVolume, int(playbackRate), float64(cfg.VocalGainRamp)/float64(time.Millisecond))
	pipeline := NewPipeline(streamer, streamer, sep, ring, format.SampleRate, playbackRate)
	pipeline.SetSeekDoneCallback(func(err error) {
		pm.seekErr.Store(seekResult{err: err})
		pm.seekDone.Store(true)
		if err != nil {
			pm.emitError(index, track.Path, fmt.Errorf("seek failed: %w", err))
			return
		}
		pm.emitEvent(EventSeekFinished, index, track.Path, "seek 完成")
	})

	var rt beep.Streamer
	var vocalChain, accompChain *dsp.Chain
	mode := cfg.DSP.Mode
	if mode == "" {
		mode = DSPModeAuto
	}
	if mode != DSPModeOff {
		vocalChain = dsp.PresetVocalChain(int(playbackRate))
		accompChain = dsp.PresetAccompChain(int(playbackRate))
		ds := newDSPStreamer(ring, mixer, vocalChain, accompChain, mode)
		if mode == DSPModeAuto {
			ds.SetInitialVocalGain(cfg.VocalGain)
		}
		rt = ds
	} else {
		rt = audio.NewRealtimeStreamer(ring, mixer)
	}

	pm.emitEvent(EventPipelineStarted, index, track.Path, "启动音频处理流水线")
	pm.pipelineWG.Add(1)
	go func() {
		defer pm.pipelineWG.Done()
		pipeline.Run()
	}()

	prefill := PrefillTargetSamples(sep)
	pm.emitEvent(EventWarmupWaiting, index, track.Path, "等待分离器预热")
	if err := waitForWarmup(ctx, sep, ring, prefill); err != nil {
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

	switcher := newSwitchableStreamer(rt)
	bundle := &PipelineBundle{
		pipeline:    pipeline,
		streamer:    switcher,
		switcher:    switcher,
		ring:        ring,
		mixer:       mixer,
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

func (pm *PlaylistManager) newSeparator(cfg Config, sampleRate int) (separator.Separator, error) {
	switch cfg.SeparatorMode {
	case "", SeparatorFake:
		return separator.NewFake(cfg.FakeVocalAmount, cfg.FakeAccompBleed, cfg.AILatency), nil
	case SeparatorONNX:
		profile, err := separator.ParseONNXProfile(cfg.ONNX.Profile)
		if err != nil {
			return nil, err
		}
		if sampleRate != profile.SampleRate {
			return nil, fmt.Errorf("onnx(%s) 要求输入采样率 %d Hz，当前是 %d Hz", profile.Name, profile.SampleRate, sampleRate)
		}
		if err := pm.ensureSharedSessions(cfg); err != nil {
			return nil, err
		}
		pm.sessionMu.Lock()
		defer pm.sessionMu.Unlock()
		return separator.NewONNXWithShared(cfg.ONNX, pm.sharedVocalSession, pm.sharedOtherSession)
	default:
		return nil, fmt.Errorf("unknown separator mode %q", cfg.SeparatorMode)
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

func (pm *PlaylistManager) loadNextAsync(index int, epoch uint64, cfg Config, track Track) {
	pm.emitEvent(EventPrewarmStarted, index, track.Path, "后台预热下一曲")
	t0 := time.Now()
	bundle, err := pm.createBundle(pm.loadCtx, cfg, index, track, "后台预热")
	elapsed := time.Since(t0)
	if err == nil {
		measured := int64(float64(elapsed.Seconds()) * float64(pm.sampleRate) * 1.2)
		if measured > pm.measuredWarmupSamples.Load() {
			pm.measuredWarmupSamples.Store(measured)
		}
	}
	pm.deliverNext(bundleResult{index: index, epoch: epoch, track: track, bundle: bundle, err: err})
}

func (pm *PlaylistManager) PostCommand(cmd PlayerCmd) {
	pm.cmdMu.Lock()
	defer pm.cmdMu.Unlock()
	if isSwitchCmd(cmd.Type) {
		if time.Since(pm.lastSwitch) < 800*time.Millisecond {
			return
		}
		pm.lastSwitch = time.Now()
	}

	select {
	case pm.cmdCh <- cmd:
	case <-time.After(50 * time.Millisecond):
	}
}

func (pm *PlaylistManager) enqueueCommand(cmd PlayerCmd) bool {
	select {
	case pm.cmdCh <- cmd:
		return true
	case <-time.After(50 * time.Millisecond):
		return false
	}
}

func isSwitchCmd(t PlaylistCmd) bool {
	return t == CmdNext || t == CmdPrev || t == CmdJump
}

func (pm *PlaylistManager) AddTracks(tracks []Track) {
	pm.enqueueCommand(PlayerCmd{Type: CmdAddTracks, Tracks: tracks})
}

func (pm *PlaylistManager) RemoveTrack(index int) {
	pm.enqueueCommand(PlayerCmd{Type: CmdRemoveTrack, Index: index})
}

func (pm *PlaylistManager) MoveTrack(from, to int) {
	pm.enqueueCommand(PlayerCmd{Type: CmdMoveTrack, Index: from, Target: to})
}

func (pm *PlaylistManager) ClearPlaylist() {
	pm.enqueueCommand(PlayerCmd{Type: CmdClearPlaylist})
}

func (pm *PlaylistManager) ShuffleUpcoming() {
	pm.enqueueCommand(PlayerCmd{Type: CmdShuffleUpcoming})
}

func (pm *PlaylistManager) SetTrack(index int, track Track) {
	pm.enqueueCommand(PlayerCmd{Type: CmdSetTrack, Index: index, Track: track})
}

func (pm *PlaylistManager) SetTrackMeta(index int, key string, val any) {
	pm.enqueueCommand(PlayerCmd{Type: CmdSetTrackMeta, Index: index, MetaKey: key, MetaValue: val})
}

func (pm *PlaylistManager) Stream(dst [][2]float64) (int, bool) {
	pm.mu.Lock()

	if pm.stopped {
		pm.mu.Unlock()
		return 0, false
	}

	select {
	case req := <-pm.seekReqCh:
		pm.applySeek(req.sampleOffset)
	default:
	}

	if pm.seeking.Load() {
		if pm.seekDone.Load() {
			if v := pm.seekErr.Load(); v != nil {
				if res, ok := v.(seekResult); ok && res.err != nil {
					pm.currentPos = pm.seekRollbackPos
				}
			}
			pm.seeking.Store(false)
			pm.seekDone.Store(false)
		} else {
			for i := range dst {
				dst[i] = [2]float64{}
			}
			pm.mu.Unlock()
			return len(dst), true
		}
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
			if pm.currentIndex < 0 || pm.currentIndex >= len(pm.tracks) {
				pm.currentIndex = 0
			}
			pm.pendingHardCutIdx = pm.currentIndex
			pm.loadingNext.Store(true)
			pm.startHardCutLoadLocked(pm.currentIndex)
			pm.needsLoad = false
		}
	}

	if pm.current == nil {
		for i := range dst {
			dst[i] = [2]float64{}
		}
		pm.mu.Unlock()
		return len(dst), true
	}

	current := pm.current
	currentPosBefore := pm.currentPos
	crossfadeSamples := pm.crossfadeSamples
	crossfadeStart := current.totalSample - crossfadeSamples
	forceHardCut := pm.forceHardCut
	next := pm.next
	shouldReadNext := !forceHardCut && crossfadeSamples > 0 &&
		current.totalSample > 0 &&
		currentPosBefore+int64(len(dst)) > crossfadeStart &&
		next != nil
	pm.mu.Unlock()

	n, ok := current.streamer.Stream(dst)
	var temp [][2]float64
	var nn int
	nextStart := 0
	if shouldReadNext && n > 0 {
		nextStart = int(crossfadeStart - currentPosBefore)
		if nextStart < 0 {
			nextStart = 0
		}
		if nextStart > n {
			nextStart = n
		}
		temp = make([][2]float64, n-nextStart)
		nn, _ = next.streamer.Stream(temp)
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.stopped || pm.current != current {
		return n, true
	}
	pm.currentPos += int64(n)

	if !pm.isCrossfading && pm.current.totalSample > 0 &&
		pm.current.totalSample-pm.currentPos <= int64(2*pm.sampleRate) {
		pm.forceHardCut = true
	}

	nextIdx := pm.currentIndex + 1
	failedPrewarm := nextIdx < len(pm.tracks) &&
		pm.failedPrewarmIndex == nextIdx &&
		pm.failedPrewarmPath == pm.tracks[nextIdx].Path
	if nextIdx < len(pm.tracks) &&
		!failedPrewarm &&
		!pm.isCrossfading &&
		!pm.forceHardCut &&
		!pm.loadingNext.Load() &&
		pm.next == nil &&
		pm.pendingHardCutIdx < 0 {
		prewarm := pm.effectivePrewarmSamples()
		triggerPoint := pm.current.totalSample - pm.crossfadeSamples - prewarm
		if triggerPoint < 0 {
			triggerPoint = 0
		}
		if pm.currentPos >= triggerPoint {
			pm.loadingNext.Store(true)
			pm.loadEpoch++
			epoch := pm.loadEpoch
			cfg := pm.cfg
			track := cloneTrack(pm.tracks[nextIdx])
			pm.loadWG.Add(1)
			go func() {
				defer pm.loadWG.Done()
				pm.loadNextAsync(nextIdx, epoch, cfg, track)
			}()
		}
	}

	posBeforeThisFrame := pm.currentPos - int64(n)
	if !pm.forceHardCut && pm.crossfadeSamples > 0 && posBeforeThisFrame+int64(n) > pm.current.totalSample-pm.crossfadeSamples {
		if !pm.isCrossfading {
			pm.emitEvent(EventCrossfadeStarted, pm.currentIndex, pm.current.track.Path, "开始淡入淡出")
		}
		pm.isCrossfading = true
		if pm.next != nil && pm.next == next && temp != nil {
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
				nextOffset := i - nextStart
				if nextOffset >= 0 && nextOffset < nn {
					dst[i][0] += temp[nextOffset][0] * alphaNext
					dst[i][1] += temp[nextOffset][1] * alphaNext
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
			pm.resetStateLocked()
			pm.emitTrackChanged(pm.current)
		} else if nextIdx < len(pm.tracks) {
			if pm.next != nil {
				pm.next.Close()
				pm.next = nil
			}
			pm.currentIndex = nextIdx
			pm.current = nil
			pm.resetStateLocked()
			pm.pendingHardCutIdx = nextIdx
			pm.loadingNext.Store(true)
			pm.startHardCutLoadLocked(nextIdx)
		} else {
			if pm.next != nil {
				pm.next.Close()
				pm.next = nil
			}
			pm.current = nil
			pm.emitPlaybackIdle()
			pm.resetStateLocked()
		}
	}

	return n, true
}

func (pm *PlaylistManager) processCommandLocked(cmd PlayerCmd) {
	switch cmd.Type {
	case CmdNext, CmdPrev, CmdJump:
		pm.startHardCut(cmd)
	case CmdPlay, CmdPlayIndex:
		if cmd.Type == CmdPlayIndex {
			if cmd.Index < 0 || cmd.Index >= len(pm.tracks) {
				return
			}
			pm.currentIndex = cmd.Index
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
			pm.resetStateLocked()
			pm.pendingHardCutIdx = pm.currentIndex
			pm.loadingNext.Store(true)
			pm.startHardCutLoadLocked(pm.currentIndex)
			pm.needsLoad = false
		}
	case CmdAddTracks:
		wasEmpty := len(pm.tracks) == 0
		for _, t := range cmd.Tracks {
			pm.tracks = append(pm.tracks, cloneTrack(t))
		}
		if wasEmpty && len(pm.tracks) > 0 {
			pm.currentIndex = 0
		}
		pm.emitPlaylistChanged()
	case CmdRemoveTrack:
		pm.handleRemoveTrackLocked(cmd.Index)
	case CmdMoveTrack:
		pm.handleMoveTrackLocked(cmd.Index, cmd.Target)
	case CmdClearPlaylist:
		pm.handleClearPlaylistLocked()
	case CmdShuffleUpcoming:
		n := len(pm.tracks) - pm.currentIndex - 1
		if n >= 2 {
			upcoming := pm.tracks[pm.currentIndex+1:]
			for i := n - 1; i > 0; i-- {
				j := rand.Intn(i + 1)
				upcoming[i], upcoming[j] = upcoming[j], upcoming[i]
			}
			pm.emitPlaylistChanged()
		}
	case CmdSetTrack:
		if cmd.Index >= 0 && cmd.Index < len(pm.tracks) {
			pm.tracks[cmd.Index] = cloneTrack(cmd.Track)
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
	pm.invalidatePendingLoadsLocked()

	if pm.current != nil && index == pm.currentIndex {
		pm.current.Close()
		pm.current = nil
		pm.drainReadyChannelsLocked()
	}

	if pm.next != nil {
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
	if pm.current != nil {
		pm.current.index = pm.currentIndex
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
	pm.invalidatePendingLoadsLocked()

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
	if pm.current != nil {
		pm.current.index = pm.currentIndex
	}

	if pm.next != nil {
		pm.next.Close()
		pm.next = nil
		pm.drainReadyChannelsLocked()
	}
	if pm.current == nil && len(pm.tracks) > 0 {
		pm.needsLoad = true
		pm.resetStateLocked()
	}

	pm.emitPlaylistChanged()
}

func (pm *PlaylistManager) handleClearPlaylistLocked() {
	pm.invalidatePendingLoadsLocked()
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
	pm.pendingHardCutIdx = -1
	pm.needsLoad = false
	pm.resetStateLocked()
	pm.emitPlaylistChanged()
	pm.emitPlaybackIdle()
}

func (pm *PlaylistManager) invalidatePendingLoadsLocked() {
	pm.loadEpoch++
	pm.pendingHardCutIdx = -1
	pm.loadingNext.Store(false)
	pm.failedPrewarmIndex = -1
	pm.failedPrewarmPath = ""
	pm.drainReadyChannelsLocked()
}

func (pm *PlaylistManager) consumeHardCutResult(result bundleResult) {
	if result.epoch != pm.loadEpoch || result.index != pm.pendingHardCutIdx ||
		result.index < 0 || result.index >= len(pm.tracks) || pm.tracks[result.index].Path != result.track.Path {
		if result.bundle != nil {
			result.bundle.Close()
		}
		return
	}

	if result.err != nil {
		pm.emitError(result.index, result.track.Path, result.err)
		pm.current = nil
		pm.pendingHardCutIdx = -1
		pm.loadingNext.Store(false)
		return
	}

	pm.applyRuntimeConfigToBundleLocked(result.bundle)
	pm.current = result.bundle
	pm.pendingHardCutIdx = -1
	pm.loadingNext.Store(false)
	pm.currentPos = 0
	pm.tracks[result.index].Duration = samplesToDuration(result.bundle.totalSample, pm.sampleRate)
	pm.emitTrackChanged(result.bundle)
}

func (pm *PlaylistManager) consumeNextResult(result bundleResult) {
	if pm.nextResultStaleLocked(result) {
		if result.bundle != nil {
			result.bundle.Close()
		}
		pm.loadingNext.Store(false)
		return
	}
	if result.err != nil {
		pm.emitError(result.index, result.track.Path, result.err)
		pm.loadingNext.Store(false)
		return
	}
	if pm.next != nil {
		pm.next.Close()
	}
	pm.applyRuntimeConfigToBundleLocked(result.bundle)
	pm.next = result.bundle
	pm.tracks[result.index].Duration = samplesToDuration(result.bundle.totalSample, pm.sampleRate)
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
	pm.resetStateLocked()
	pm.pendingHardCutIdx = targetIdx
	pm.loadingNext.Store(true)

	pm.startHardCutLoadLocked(targetIdx)
}

func (pm *PlaylistManager) startHardCutLoadLocked(idx int) {
	if idx < 0 || idx >= len(pm.tracks) || pm.stopped {
		return
	}
	pm.loadEpoch++
	epoch := pm.loadEpoch
	cfg := pm.cfg
	track := cloneTrack(pm.tracks[idx])
	pm.loadWG.Add(1)
	go func() {
		defer pm.loadWG.Done()
		pm.loadBundleForHardCut(idx, epoch, cfg, track)
	}()
}

func (pm *PlaylistManager) loadBundleForHardCut(idx int, epoch uint64, cfg Config, track Track) {
	bundle, err := pm.createBundle(pm.loadCtx, cfg, idx, track, "hard cut load")
	result := bundleResult{index: idx, epoch: epoch, track: track, bundle: bundle, err: err}
	pm.deliverHardCut(result)
}

func (pm *PlaylistManager) deliverHardCut(result bundleResult) {
	pm.mu.Lock()
	stale := pm.stopped ||
		result.epoch != pm.loadEpoch ||
		result.index != pm.pendingHardCutIdx ||
		result.index < 0 ||
		result.index >= len(pm.tracks) ||
		pm.tracks[result.index].Path != result.track.Path
	if stale {
		pm.mu.Unlock()
		if result.bundle != nil {
			result.bundle.Close()
		}
		return
	}
	delivered := pm.offerResultLocked(pm.hardCutReadyCh, result)
	pm.mu.Unlock()
	if !delivered && result.bundle != nil {
		result.bundle.Close()
	}
}

func (pm *PlaylistManager) deliverNext(result bundleResult) {
	pm.mu.Lock()
	stale := pm.nextResultStaleLocked(result)
	if stale {
		pm.mu.Unlock()
		if result.bundle != nil {
			result.bundle.Close()
		}
		return
	}

	if result.err != nil {
		pm.failedPrewarmIndex = result.index
		pm.failedPrewarmPath = result.track.Path
		pm.loadingNext.Store(false)
		pm.mu.Unlock()
		pm.emitError(result.index, result.track.Path, fmt.Errorf("预热下一首失败: %w", result.err))
		return
	}
	delivered := pm.offerResultLocked(pm.nextReadyCh, result)
	if !delivered {
		pm.loadingNext.Store(false)
	}
	pm.mu.Unlock()
	if !delivered && result.bundle != nil {
		result.bundle.Close()
	}
	if delivered {
		pm.emitEvent(EventPrewarmReady, result.index, result.track.Path, "下一曲预热完成")
	}
}

func (pm *PlaylistManager) nextResultStaleLocked(result bundleResult) bool {
	expected := pm.currentIndex + 1
	return pm.stopped ||
		result.epoch != pm.loadEpoch ||
		result.index != expected ||
		pm.pendingHardCutIdx >= 0 ||
		result.index < 0 ||
		result.index >= len(pm.tracks) ||
		pm.tracks[result.index].Path != result.track.Path
}

func (pm *PlaylistManager) offerResultLocked(ch chan bundleResult, result bundleResult) bool {
	select {
	case ch <- result:
		return true
	default:
	}

	select {
	case old := <-ch:
		if old.bundle != nil {
			old.bundle.Close()
		}
	default:
	}

	select {
	case ch <- result:
		return true
	default:
		return false
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
	pm.seekRollbackPos = pm.currentPos
	pm.currentPos = sampleOffset
	pm.seeking.Store(true)
	pm.seekDone.Store(false)
	pm.seekErr.Store(seekResult{})
	if !pm.current.pipeline.SeekSamplesAsync(sampleOffset) {
		pm.seeking.Store(false)
		pm.seekDone.Store(false)
		pm.emitError(pm.current.index, pm.current.track.Path, ErrPipelineStopped)
		return
	}
	pm.current.pipeline.ring.DiscardAndReopen()
	if pm.current.totalSample > 0 &&
		pm.current.totalSample-sampleOffset <= int64(2*pm.sampleRate) {
		pm.forceHardCut = true
	}
}

func (pm *PlaylistManager) resetStateLocked() {
	pm.isCrossfading = false
	pm.forceHardCut = false
	pm.nextCrossfadeConsumed = 0
	pm.failedPrewarmIndex = -1
	pm.failedPrewarmPath = ""
	pm.loadingNext.Store(false)
	pm.seeking.Store(false)
	pm.seekDone.Store(false)
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
		select {
		case pm.seekReqCh <- playlistSeekRequest{sampleOffset: sampleOffset}:
		default:
		}
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
			if b != nil && b.switcher != nil {
				if ds, ok := b.switcher.Current().(*dspStreamer); ok {
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
	pm.failedPrewarmIndex = -1
	pm.failedPrewarmPath = ""
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
	pm.failedPrewarmIndex = -1
	pm.failedPrewarmPath = ""
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
	if !isValidDSPMode(mode) {
		pm.emitError(pm.currentIndex, pm.currentPathLocked(), fmt.Errorf("unknown DSP mode %q", mode))
		return
	}
	pm.cfg.DSP.Mode = mode
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		pm.setBundleDSPModeLocked(b, mode, false)
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
	if pm.stopped {
		pm.mu.Unlock()
		return
	}
	pm.stopped = true
	if pm.cancelLoads != nil {
		pm.cancelLoads()
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
	pm.mu.Unlock()

	pm.loadWG.Wait()
	pm.pipelineWG.Wait()

	pm.sessionMu.Lock()
	defer pm.sessionMu.Unlock()
	if pm.sharedVocalSession != nil {
		_ = pm.sharedVocalSession.Destroy()
		pm.sharedVocalSession = nil
	}
	if pm.sharedOtherSession != nil {
		_ = pm.sharedOtherSession.Destroy()
		pm.sharedOtherSession = nil
	}
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

func (pm *PlaylistManager) HasTrack() bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.current != nil || pm.pendingHardCutIdx >= 0
}

func (pm *PlaylistManager) TrackList() []Track {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return cloneTracks(pm.tracks)
}

func (pm *PlaylistManager) applyToBundlesLocked(fn func(*audio.Mixer)) {
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		if b == nil || b.mixer == nil {
			continue
		}
		fn(b.mixer)
	}
}

func (pm *PlaylistManager) applyToSeparatorsLocked(fn func(sep separator.Separator)) {
	for _, b := range []*PipelineBundle{pm.current, pm.next} {
		if b != nil && b.separator != nil {
			fn(b.separator)
		}
	}
}

func (pm *PlaylistManager) applyRuntimeConfigToBundleLocked(b *PipelineBundle) {
	if b == nil {
		return
	}
	if b.mixer != nil {
		b.mixer.SetVocalGain(pm.cfg.VocalGain)
		b.mixer.SetMasterVolume(pm.cfg.MasterVolume)
		b.mixer.SetRampDurationMs(float64(pm.cfg.VocalGainRamp) / float64(time.Millisecond))
	}
	if b.separator != nil {
		if p, ok := b.separator.(separator.ParameterizedFake); ok {
			p.SetFakeParams(pm.cfg.FakeVocalAmount, pm.cfg.FakeAccompBleed)
		}
		if p, ok := b.separator.(interface{ SetCompensation(float64) }); ok {
			p.SetCompensation(pm.cfg.ONNX.Compensation)
		}
	}
	pm.setBundleDSPModeLocked(b, pm.cfg.DSP.Mode, true)
}

func (pm *PlaylistManager) setBundleDSPModeLocked(b *PipelineBundle, mode DSPMode, initializing bool) {
	if b == nil || b.switcher == nil || b.ring == nil || b.mixer == nil {
		return
	}
	current := b.switcher.Current()
	if mode == DSPModeOff {
		if ds, ok := current.(*dspStreamer); ok {
			if initializing {
				ds.SetInitialBypassed(true)
			} else {
				ds.SetMode(mode)
			}
		} else {
			b.switcher.Set(audio.NewRealtimeStreamer(b.ring, b.mixer))
		}
		return
	}
	if ds, ok := current.(*dspStreamer); ok {
		if initializing {
			ds.SetModeImmediate(mode, pm.cfg.VocalGain)
		} else {
			ds.SetMode(mode)
			if mode == DSPModeAuto {
				ds.OnVocalGainChanged(pm.cfg.VocalGain)
			}
		}
		return
	}
	if b.vocalChain == nil {
		b.vocalChain = dsp.PresetVocalChain(pm.sampleRate)
	}
	if b.accompChain == nil {
		b.accompChain = dsp.PresetAccompChain(pm.sampleRate)
	}
	ds := newDSPStreamer(b.ring, b.mixer, b.vocalChain, b.accompChain, mode)
	if mode == DSPModeAuto {
		ds.SetInitialVocalGain(pm.cfg.VocalGain)
	}
	b.switcher.Set(ds)
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
