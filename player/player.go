package player

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/xiaowumin-mark/VoxBackend/audio"
	"github.com/xiaowumin-mark/VoxBackend/separator"
)

type Player struct {
	cfgMu sync.RWMutex
	cfg   Config

	eventMu      sync.RWMutex
	events       chan Event
	eventsClosed bool

	managerMu sync.RWMutex
	manager   *PlaylistManager
	ctrl      *beep.Ctrl

	paused     atomic.Bool
	sampleRate atomic.Int64

	done      chan struct{}
	startOnce sync.Once
	stopOnce  sync.Once
	cancel    context.CancelFunc

	errMu sync.Mutex
	err   error
}

func New(cfg Config) *Player {
	cfg = normalizeConfig(cfg)
	return &Player{
		cfg:    cfg,
		events: make(chan Event, 2048),
		done:   make(chan struct{}),
	}
}

func (p *Player) Start(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	started := false
	p.startOnce.Do(func() {
		started = true
		runCtx, cancel := context.WithCancel(ctx)
		p.cancel = cancel
		go p.dispatchEvents()
		go p.run(runCtx)
	})
	if !started {
		return errors.New("播放器已经启动")
	}
	return nil
}

func (p *Player) Wait() error {
	<-p.done
	p.errMu.Lock()
	defer p.errMu.Unlock()
	return p.err
}

func (p *Player) Stop() {
	p.stopOnce.Do(func() {
		if p.cancel != nil {
			p.cancel()
		}
		p.managerMu.RLock()
		manager := p.manager
		ctrl := p.ctrl
		p.managerMu.RUnlock()
		if manager != nil {
			manager.Stop()
		}
		if ctrl != nil {
			speaker.Lock()
			ctrl.Paused = false
			speaker.Unlock()
		}
	})
}

func (p *Player) run(ctx context.Context) {
	defer close(p.done)
	defer p.closeEvents()

	cfg := p.config()
	p.emit(newEvent(EventStarting, "播放器启动中"))

	playbackRate := resolvePlaybackRate(cfg)
	p.sampleRate.Store(int64(playbackRate))

	if err := speaker.Init(playbackRate, cfg.SpeakerBufferSize); err != nil {
		p.finishWithError(fmt.Errorf("初始化音频输出失败: %w", err))
		return
	}
	defer speaker.Close()

	manager := NewPlaylistManager(ctx, cfg, cloneTracks(cfg.Tracks), int(playbackRate), p.emit)
	if err := manager.Start(); err != nil {
		p.finishWithError(fmt.Errorf("启动播放列表管理失败: %w", err))
		return
	}

	ctrl := &beep.Ctrl{Streamer: manager, Paused: p.paused.Load()}
	p.managerMu.Lock()
	p.manager = manager
	p.ctrl = ctrl
	p.managerMu.Unlock()

	stateDone := make(chan struct{})
	go p.emitStateLoop(ctx, stateDone)
	defer func() {
		close(stateDone)
		manager.Stop()
	}()

	go func() {
		<-ctx.Done()
		manager.Stop()
		speaker.Lock()
		ctrl.Paused = false
		speaker.Unlock()
	}()

	speaker.PlayAndWait(ctrl)
	p.emit(newEvent(EventStopped, "播放器已停止"))
}

func resolvePlaybackRate(cfg Config) beep.SampleRate {
	if cfg.SeparatorMode == SeparatorONNX {
		profile, err := separator.ParseONNXProfile(cfg.ONNX.Profile)
		if err != nil {
			return ONNXSampleRate
		}
		return beep.SampleRate(profile.SampleRate)
	}

	if len(cfg.Tracks) == 0 {
		return defaultSampleRate
	}

	streamer, format, err := audio.DecodeFile(cfg.Tracks[0].Path)
	if err != nil {
		return defaultSampleRate
	}
	_ = streamer.Close()
	return format.SampleRate
}

const (
	ONNXSampleRate    = 44100
	defaultSampleRate = 44100
)

func (p *Player) Play() {
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.PostCommand(PlayerCmd{Type: CmdPlay})
	}
}

func (p *Player) PlayIndex(index int) {
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.PostCommand(PlayerCmd{Type: CmdPlayIndex, Index: index})
	}
}

func (p *Player) AddTracks(tracks ...Track) {
	p.updateConfig(func(cfg *Config) {
		cfg.Tracks = append(cfg.Tracks, tracks...)
	})
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.AddTracks(tracks)
	}
}

func (p *Player) AddPaths(paths ...string) {
	tracks := make([]Track, len(paths))
	for i, pth := range paths {
		tracks[i] = NewTrack(pth)
	}
	p.AddTracks(tracks...)
}

func (p *Player) RemoveTrack(index int) {
	p.updateConfig(func(cfg *Config) {
		if index < 0 || index >= len(cfg.Tracks) {
			return
		}
		cfg.Tracks = append(cfg.Tracks[:index], cfg.Tracks[index+1:]...)
	})
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.RemoveTrack(index)
	}
}

func (p *Player) MoveTrack(from, to int) {
	p.updateConfig(func(cfg *Config) {
		if from < 0 || from >= len(cfg.Tracks) || to < 0 || to >= len(cfg.Tracks) || from == to {
			return
		}
		t := cfg.Tracks[from]
		cfg.Tracks = append(cfg.Tracks[:from], cfg.Tracks[from+1:]...)
		if to > from {
			to--
		}
		cfg.Tracks = append(cfg.Tracks[:to], append([]Track{t}, cfg.Tracks[to:]...)...)
	})
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.MoveTrack(from, to)
	}
}

func (p *Player) ClearPlaylist() {
	p.updateConfig(func(cfg *Config) { cfg.Tracks = nil })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.ClearPlaylist()
	}
}

func (p *Player) SetTrack(index int, track Track) {
	p.updateConfig(func(cfg *Config) {
		if index < 0 || index >= len(cfg.Tracks) {
			return
		}
		cfg.Tracks[index] = track
	})
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetTrack(index, track)
	}
}

func (p *Player) SetTrackMeta(index int, key string, val any) {
	p.updateConfig(func(cfg *Config) {
		if index < 0 || index >= len(cfg.Tracks) {
			return
		}
		if cfg.Tracks[index].Meta == nil {
			cfg.Tracks[index].Meta = make(map[string]any)
		}
		cfg.Tracks[index].Meta[key] = val
	})
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetTrackMeta(index, key, val)
	}
}

func (p *Player) Playlist() []Track {
	cfg := p.config()
	return cloneTracks(cfg.Tracks)
}

func (p *Player) emitStateLoop(ctx context.Context, done <-chan struct{}) {
	cfg := p.config()
	fps := cfg.StateFrameRate
	if fps <= 0 {
		fps = DefaultStateFrameRate
	}
	interval := time.Second / time.Duration(fps)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			cb := p.config().Callbacks
			if cb.OnState != nil {
				cb.OnState(p.Snapshot())
			}
		}
	}
}

func (p *Player) dispatchEvents() {
	for ev := range p.events {
		state := p.Snapshot()
		ev.State = &state
		cb := p.config().Callbacks
		if cb.OnEvent != nil {
			cb.OnEvent(ev)
		}
		switch ev.Type {
		case EventPausedChanged:
			if cb.OnPausedChanged != nil {
				cb.OnPausedChanged(state)
			}
		case EventVolumeChanged:
			if cb.OnVolumeChanged != nil {
				cb.OnVolumeChanged(state)
			}
		case EventTrackChanged:
			if cb.OnTrackChanged != nil {
				cb.OnTrackChanged(state)
			}
		case EventVocalGainChanged, EventRampChanged:
			if cb.OnVocalChanged != nil {
				cb.OnVocalChanged(state)
			}
		case EventPlaylistChanged:
			if cb.OnPlaylistChanged != nil {
				cb.OnPlaylistChanged(state)
			}
		}
	}
}

func (p *Player) emit(ev Event) {
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	p.eventMu.RLock()
	defer p.eventMu.RUnlock()
	if p.eventsClosed {
		return
	}
	select {
	case p.events <- ev:
	default:
	}
}

func (p *Player) closeEvents() {
	p.eventMu.Lock()
	defer p.eventMu.Unlock()
	if p.eventsClosed {
		return
	}
	p.eventsClosed = true
	close(p.events)
}

func (p *Player) finishWithError(err error) {
	p.errMu.Lock()
	p.err = err
	p.errMu.Unlock()
	p.emit(Event{Type: EventError, Time: time.Now(), Message: err.Error(), Err: err})
}

func (p *Player) config() Config {
	p.cfgMu.RLock()
	defer p.cfgMu.RUnlock()
	return p.cfg
}

func (p *Player) updateConfig(fn func(*Config)) Config {
	p.cfgMu.Lock()
	defer p.cfgMu.Unlock()
	fn(&p.cfg)
	return p.cfg
}

func (p *Player) managerSnapshot() (*PlaylistManager, *beep.Ctrl) {
	p.managerMu.RLock()
	defer p.managerMu.RUnlock()
	return p.manager, p.ctrl
}

func (p *Player) Snapshot() State {
	manager, _ := p.managerSnapshot()
	if manager != nil {
		return manager.Snapshot(p.paused.Load())
	}
	cfg := p.config()
	return State{
		Paused:     p.paused.Load(),
		Volume:     cfg.MasterVolume,
		VocalGain:  cfg.VocalGain,
		VocalRamp:  cfg.VocalGainRamp,
		Crossfade:  cfg.Crossfade,
		Prewarm:    cfg.Prewarm,
		TrackCount: len(cfg.Tracks),
		Idle:       true,
	}
}

func (p *Player) SetPaused(paused bool) {
	p.paused.Store(paused)
	_, ctrl := p.managerSnapshot()
	if ctrl != nil {
		speaker.Lock()
		ctrl.Paused = paused
		speaker.Unlock()
	}
	if !paused {
		manager, _ := p.managerSnapshot()
		if manager != nil && !manager.HasTrack() {
			manager.PostCommand(PlayerCmd{Type: CmdPlay})
		}
	}
	msg := "恢复播放"
	if paused {
		msg = "暂停播放"
	}
	p.emit(newEvent(EventPausedChanged, msg))
}

func (p *Player) TogglePaused() {
	p.SetPaused(!p.paused.Load())
}

func (p *Player) SetMasterVolume(v float64) {
	p.updateConfig(func(cfg *Config) { cfg.MasterVolume = v })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetMasterVolume(v)
		return
	}
	p.emit(newEvent(EventVolumeChanged, "主音量已更新"))
}

func (p *Player) SetVocalGain(v float64) {
	p.updateConfig(func(cfg *Config) { cfg.VocalGain = v })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetVocalGain(v)
		return
	}
	p.emit(newEvent(EventVocalGainChanged, "人声增益已更新"))
}

func (p *Player) SetVocalRamp(d time.Duration) {
	p.updateConfig(func(cfg *Config) { cfg.VocalGainRamp = d })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetRampDuration(d)
		return
	}
	p.emit(newEvent(EventRampChanged, "人声变化平滑时间已更新"))
}

func (p *Player) SetCrossfade(d time.Duration) {
	p.updateConfig(func(cfg *Config) { cfg.Crossfade = d })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetCrossfade(d)
		return
	}
	p.emit(newEvent(EventCrossfadeChanged, "淡入淡出时长已更新"))
}

func (p *Player) SetPrewarm(d time.Duration) {
	p.updateConfig(func(cfg *Config) { cfg.Prewarm = d })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetPrewarm(d)
		return
	}
	p.emit(newEvent(EventPrewarmChanged, "预热提前量已更新"))
}

func (p *Player) SetFakeParams(vocalAmount, accompBleed float64) {
	p.updateConfig(func(cfg *Config) {
		cfg.FakeVocalAmount = vocalAmount
		cfg.FakeAccompBleed = accompBleed
	})
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetFakeParams(vocalAmount, accompBleed)
		return
	}
	p.emit(newEvent(EventFakeParamsChanged, "fake 分离器参数已更新"))
}

func (p *Player) SetONNXCompensation(v float64) {
	p.updateConfig(func(cfg *Config) { cfg.ONNX.Compensation = v })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetONNXCompensation(v)
		return
	}
	p.emit(newEvent(EventONNXCompChanged, "ONNX 补偿系数已更新"))
}

func (p *Player) SetDSPMode(mode DSPMode) {
	p.updateConfig(func(cfg *Config) { cfg.DSP.Mode = mode })
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.SetDSPMode(mode)
		return
	}
	p.emit(newEvent(EventDSPChanged, "DSP 模式已更新"))
}

func (p *Player) SetDSPEnabled(enabled bool) {
	if enabled {
		p.SetDSPMode(DSPModeOn)
	} else {
		p.SetDSPMode(DSPModeOff)
	}
}

func (p *Player) Next() {
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.PostCommand(PlayerCmd{Type: CmdNext})
	}
}

func (p *Player) Previous() {
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.PostCommand(PlayerCmd{Type: CmdPrev})
	}
}

func (p *Player) Jump(index int) {
	manager, _ := p.managerSnapshot()
	if manager != nil {
		manager.PostCommand(PlayerCmd{Type: CmdJump, Index: index})
	}
}

func (p *Player) SeekTo(position time.Duration) {
	if position < 0 {
		position = 0
	}
	manager, _ := p.managerSnapshot()
	rate := p.sampleRate.Load()
	if manager != nil && rate > 0 {
		manager.RequestSeek(int64(position.Seconds() * float64(rate)))
	}
}
