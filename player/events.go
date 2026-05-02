package player

import "time"

type EventType string

const (
	EventStarting          EventType = "starting"
	EventTracksLoaded      EventType = "tracks_loaded"
	EventTrackLoading      EventType = "track_loading"
	EventSeparatorCreating EventType = "separator_creating"
	EventPipelineStarted   EventType = "pipeline_started"
	EventWarmupWaiting     EventType = "warmup_waiting"
	EventWarmupReady       EventType = "warmup_ready"
	EventTrackReady        EventType = "track_ready"
	EventTrackChanged      EventType = "track_changed"
	EventPrewarmStarted    EventType = "prewarm_started"
	EventPrewarmReady      EventType = "prewarm_ready"
	EventCrossfadeStarted  EventType = "crossfade_started"
	EventHardCutStarted    EventType = "hardcut_started"
	EventSeekStarted       EventType = "seek_started"
	EventSeekFinished      EventType = "seek_finished"
	EventPausedChanged     EventType = "paused_changed"
	EventVolumeChanged     EventType = "volume_changed"
	EventVocalGainChanged  EventType = "vocal_gain_changed"
	EventRampChanged       EventType = "ramp_changed"
	EventCrossfadeChanged  EventType = "crossfade_changed"
	EventPrewarmChanged    EventType = "prewarm_changed"
	EventFakeParamsChanged EventType = "fake_params_changed"
	EventONNXCompChanged   EventType = "onnx_compensation_changed"
	EventDSPChanged        EventType = "dsp_changed"
	EventPlaylistChanged   EventType = "playlist_changed"
	EventPlaybackIdle      EventType = "playback_idle"
	EventStopped           EventType = "stopped"
	EventError             EventType = "error"
)

type Event struct {
	Type       EventType
	Time       time.Time
	TrackIndex int
	TrackPath  string
	Track      *Track
	Message    string
	Err        error
	Data       map[string]any
	State      *State
}

type State struct {
	TrackIndex  int
	TrackPath   string
	Track       *Track
	Position    time.Duration
	Duration    time.Duration
	Paused      bool
	Loading     bool
	Crossfading bool
	Volume      float64
	VocalGain   float64
	VocalRamp   time.Duration
	Crossfade   time.Duration
	Prewarm     time.Duration
	TrackCount  int
	Idle        bool
	Finished    bool
}

type Callbacks struct {
	OnEvent func(Event)
	OnState func(State)

	OnPausedChanged   func(State)
	OnVolumeChanged   func(State)
	OnTrackChanged    func(State)
	OnVocalChanged    func(State)
	OnPlaylistChanged func(State)
}

func newEvent(t EventType, message string) Event {
	return Event{Type: t, Time: time.Now(), Message: message}
}
