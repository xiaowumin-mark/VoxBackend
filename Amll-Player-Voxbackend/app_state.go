package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	amllwsclient "github.com/xiaowumin-mark/VoxBackend/amll-ws-client"
	"github.com/xiaowumin-mark/VoxBackend/player"
)

type appSnapshot struct {
	Config      AppConfig
	ConfigPath  string
	Candidates  []modelCandidate
	Profile     string
	ProfileNote string
	Logs        []string
	State       player.State
	Running     bool
	Starting    bool
	AMLLStatus  amllwsclient.Status
	Resources   resourceStats
	LastError   string
}

type appStore struct {
	mu sync.RWMutex

	config      AppConfig
	configPath  string
	candidates  []modelCandidate
	profile     string
	profileNote string
	logs        []string
	state       player.State
	running     bool
	starting    bool
	amllStatus  amllwsclient.Status
	resources   resourceStats
	lastError   string
}

func newAppStore(cfg AppConfig, cfgPath string) *appStore {
	cfg.normalize()
	candidates := candidatesWithCurrent(cfg.ModelPath)
	if len(candidates) > 0 && selectedCandidateIndex(candidates, cfg.ModelPath) < 0 {
		cfg.ModelPath = candidates[0].Path
	}
	profile, note := profileForDisplay(cfg.ModelPath)

	return &appStore{
		config:      cfg,
		configPath:  cfgPath,
		candidates:  candidates,
		profile:     profile,
		profileNote: note,
		amllStatus:  amllwsclient.StatusDisconnected,
		resources:   newResourceSampler().Sample(),
	}
}

func (s *appStore) Snapshot() appSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return appSnapshot{
		Config:      s.config,
		ConfigPath:  s.configPath,
		Candidates:  append([]modelCandidate(nil), s.candidates...),
		Profile:     s.profile,
		ProfileNote: s.profileNote,
		Logs:        append([]string(nil), s.logs...),
		State:       clonePlayerState(s.state),
		Running:     s.running,
		Starting:    s.starting,
		AMLLStatus:  s.amllStatus,
		Resources:   s.resources,
		LastError:   s.lastError,
	}
}

func (s *appStore) AppendLog(format string, args ...any) {
	if s == nil {
		return
	}
	msg := fmt.Sprintf(format, args...)
	line := time.Now().Format("15:04:05") + "  " + strings.TrimSpace(msg)
	if strings.TrimSpace(msg) == "" {
		return
	}

	s.mu.Lock()
	s.logs = append(s.logs, line)
	if len(s.logs) > 1200 {
		s.logs = append([]string(nil), s.logs[len(s.logs)-700:]...)
	}
	s.mu.Unlock()
}

func (s *appStore) SelectModel(path string) {
	if s == nil {
		return
	}
	path = strings.TrimSpace(path)
	if path != "" {
		path = filepath.Clean(path)
	}
	profile, note := profileForDisplay(path)

	s.mu.Lock()
	s.config.ModelPath = path
	s.profile = profile
	s.profileNote = note
	s.lastError = ""
	s.mu.Unlock()
}

func (s *appStore) RefreshModels() {
	if s == nil {
		return
	}
	s.mu.Lock()
	current := s.config.ModelPath
	candidates := candidatesWithCurrent(current)
	if current == "" && len(candidates) > 0 {
		current = candidates[0].Path
		s.config.ModelPath = current
	}
	s.candidates = candidates
	s.profile, s.profileNote = profileForDisplay(current)
	count := len(candidates)
	s.mu.Unlock()

	s.AppendLog("已刷新模型列表，找到 %d 个 ONNX 文件", count)
}

func (s *appStore) UpdatePlayerState(state player.State) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.state = clonePlayerState(state)
	s.mu.Unlock()
}

func (s *appStore) SetStarting(starting bool) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.starting = starting
	if starting {
		s.lastError = ""
	}
	s.mu.Unlock()
}

func (s *appStore) MarkStarted(msg startedMsg) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.config = msg.Config
	s.profile = msg.Profile
	s.profileNote = msg.ProfileNote
	s.running = true
	s.starting = false
	s.lastError = ""
	s.mu.Unlock()

	s.AppendLog("已启动，模型类型: %s (%s)", msg.Profile, msg.ProfileNote)
}

func (s *appStore) MarkStartFailed(err error) {
	if s == nil {
		return
	}
	text := "启动失败"
	if err != nil {
		text = err.Error()
	}
	s.mu.Lock()
	s.running = false
	s.starting = false
	s.lastError = text
	s.mu.Unlock()

	s.AppendLog("启动失败: %s", text)
}

func (s *appStore) MarkStopped(err error) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.running = false
	s.starting = false
	s.mu.Unlock()

	if err != nil {
		s.AppendLog("播放器已停止: %v", err)
		return
	}
	s.AppendLog("播放器已停止")
}

func (s *appStore) SetAMLLStatus(status amllwsclient.Status) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.amllStatus = status
	s.mu.Unlock()
}

func (s *appStore) UpdateResources(stats resourceStats) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.resources = stats
	s.mu.Unlock()
}

func candidatesWithCurrent(current string) []modelCandidate {
	candidates := discoverONNXModels()
	if strings.TrimSpace(current) == "" {
		return candidates
	}
	for _, c := range candidates {
		if samePath(c.Path, current) {
			return candidates
		}
	}
	profile, reason := detectProfileFromFilename(current)
	return append([]modelCandidate{{
		Path:    current,
		Name:    filepath.Base(current),
		Profile: profile,
		Reason:  reason,
	}}, candidates...)
}

func selectedCandidateIndex(candidates []modelCandidate, path string) int {
	for i, c := range candidates {
		if samePath(c.Path, path) {
			return i
		}
	}
	return -1
}

func profileForDisplay(path string) (string, string) {
	if strings.TrimSpace(path) == "" {
		return "未选择", "请选择 ONNX 模型文件"
	}
	profile, note, err := resolveConfiguredProfile(path)
	if err != nil {
		return "识别失败", err.Error()
	}
	return profile, note
}

func clonePlayerState(src player.State) player.State {
	dst := src
	if src.Track != nil {
		t := *src.Track
		if src.Track.Meta != nil {
			t.Meta = make(map[string]any, len(src.Track.Meta))
			for k, v := range src.Track.Meta {
				t.Meta[k] = v
			}
		}
		dst.Track = &t
	}
	return dst
}

func samePath(a, b string) bool {
	a = strings.TrimSpace(filepath.Clean(a))
	b = strings.TrimSpace(filepath.Clean(b))
	return strings.EqualFold(a, b)
}
