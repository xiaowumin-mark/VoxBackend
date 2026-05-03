package amllwsclient

import (
	"encoding/json"
	"fmt"
)

type payloadType struct {
	Type  string          `json:"type"`
	Value json.RawMessage `json:"value,omitempty"`
}

type commandType struct {
	Command string `json:"command"`
}

type stateType struct {
	Update string `json:"update"`
}

// ── 消息顶层（对齐 Rust MessageV2 + Payload） ──

// ParseIncoming 解析服务器发来的消息，返回 PayloadType 标识和原始 JSON 片段。
// Rust 用 serde(tag="type", content="value")，Go 用两阶段解析。
func ParseIncoming(raw []byte) (string, json.RawMessage, error) {
	var outer payloadType
	if err := json.Unmarshal(raw, &outer); err != nil {
		return "", nil, fmt.Errorf("解析外层失败: %w", err)
	}
	return outer.Type, outer.Value, nil
}

// PayloadType 对应 Rust Payload 枚举的 type 字段值。
type PayloadType string

const (
	PayloadInitialize PayloadType = "initialize"
	PayloadPing       PayloadType = "ping"
	PayloadPong       PayloadType = "pong"
	PayloadCommand    PayloadType = "command"
	PayloadState      PayloadType = "state"
)

// ── 命令（服务器 → 客户端，对齐 Rust Command） ──

type CommandType string

const (
	CmdPause            CommandType = "pause"
	CmdResume           CommandType = "resume"
	CmdForwardSong      CommandType = "forwardSong"
	CmdBackwardSong     CommandType = "backwardSong"
	CmdSetVolume        CommandType = "setVolume"
	CmdSeekPlayProgress CommandType = "seekPlayProgress"
	CmdSetRepeatMode    CommandType = "setRepeatMode"
	CmdSetShuffleMode   CommandType = "setShuffleMode"
)

type Command struct {
	Type     CommandType `json:"command"`
	Progress uint64      `json:"progress,omitempty"`
	Volume   float64     `json:"volume,omitempty"`
	Mode     string      `json:"mode,omitempty"`
	Enabled  *bool       `json:"enabled,omitempty"`
}

func ParseCommand(raw json.RawMessage) (*Command, error) {
	var cmd Command
	if err := json.Unmarshal(raw, &cmd); err != nil {
		return nil, fmt.Errorf("解析命令失败: %w", err)
	}
	return &cmd, nil
}

// ── 状态更新（客户端 → 服务器，对齐 Rust StateUpdate） ──

type StateUpdateType string

const (
	StateSetMusic    StateUpdateType = "setMusic"
	StateSetCover    StateUpdateType = "setCover"
	StateSetLyric    StateUpdateType = "setLyric"
	StateProgress    StateUpdateType = "progress"
	StateVolume      StateUpdateType = "volume"
	StatePaused      StateUpdateType = "paused"
	StateResumed     StateUpdateType = "resumed"
	StateAudioData   StateUpdateType = "audioData"
	StateModeChanged StateUpdateType = "modeChanged"
)

type RepeatMode string

const (
	RepeatOff RepeatMode = "off"
	RepeatAll RepeatMode = "all"
	RepeatOne RepeatMode = "one"
)

// ── 数据结构（对齐 Rust common.rs / v2.rs） ──

type Artist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type MusicInfo struct {
	MusicID   string   `json:"musicId"`
	MusicName string   `json:"musicName"`
	AlbumID   string   `json:"albumId"`
	AlbumName string   `json:"albumName"`
	Artists   []Artist `json:"artists"`
	Duration  uint64   `json:"duration"`
}

type ImageData struct {
	MimeType string `json:"mimeType,omitempty"`
	Data     []byte `json:"data"`
}

type AlbumCover struct {
	Source string     `json:"source,omitempty"`
	URL    string     `json:"url,omitempty"`
	Image  *ImageData `json:"image,omitempty"`
}

type LyricWord struct {
	StartTime uint64 `json:"startTime"`
	EndTime   uint64 `json:"endTime"`
	Word      string `json:"word"`
	RomanWord string `json:"romanWord,omitempty"`
}

type LyricLine struct {
	StartTime       uint64      `json:"startTime"`
	EndTime         uint64      `json:"endTime"`
	Words           []LyricWord `json:"words"`
	TranslatedLyric string      `json:"translatedLyric,omitempty"`
	RomanLyric      string      `json:"romanLyric,omitempty"`
	IsBG            bool        `json:"isBG"`
	IsDuet          bool        `json:"isDuet"`
}

type LyricContentFormat string

const (
	LyricFormatStructured LyricContentFormat = "structured"
	LyricFormatTTML       LyricContentFormat = "ttml"
)

type LyricContent struct {
	Format LyricContentFormat `json:"format"`
	Lines  []LyricLine        `json:"lines,omitempty"`
	Data   string             `json:"data,omitempty"`
}

// ── 消息序列化（对齐 Rust serde_json 输出） ──

func makePayloadV(kind string, value any) map[string]any {
	if value == nil {
		return map[string]any{"type": kind}
	}
	return map[string]any{"type": kind, "value": value}
}

func marshalPayload(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("序列化协议消息失败: " + err.Error())
	}
	return b
}

func marshalInitialize() []byte {
	return marshalPayload(makePayloadV("initialize", nil))
}

func marshalPing() []byte {
	return marshalPayload(makePayloadV("ping", nil))
}

func marshalPong() []byte {
	return marshalPayload(makePayloadV("pong", nil))
}

func marshalStateValue(update string, extra map[string]any) map[string]any {
	m := map[string]any{"update": update}
	for k, v := range extra {
		m[k] = v
	}
	return m
}

func marshalSetMusic(info MusicInfo) []byte {
	return marshalPayload(makePayloadV("state", map[string]any{
		"update":    "setMusic",
		"musicId":   info.MusicID,
		"musicName": info.MusicName,
		"albumId":   info.AlbumID,
		"albumName": info.AlbumName,
		"artists":   info.Artists,
		"duration":  info.Duration,
	}))
}

func marshalSetCover(cover AlbumCover) []byte {
	m := map[string]any{"update": "setCover", "source": cover.Source}
	if cover.URL != "" {
		m["url"] = cover.URL
	}
	if cover.Image != nil {
		m["image"] = map[string]any{
			"mimeType": cover.Image.MimeType,
			"data":     cover.Image.Data,
		}
	}
	return marshalPayload(makePayloadV("state", m))
}

func marshalSetLyric(content LyricContent) []byte {
	m := map[string]any{"update": "setLyric", "format": content.Format}
	if content.Format == LyricFormatStructured {
		m["lines"] = content.Lines
	} else {
		m["data"] = content.Data
	}
	return marshalPayload(makePayloadV("state", m))
}

func marshalProgress(progress uint64) []byte {
	return marshalPayload(makePayloadV("state", marshalStateValue("progress", map[string]any{
		"progress": progress,
	})))
}

func marshalVolume(volume float64) []byte {
	return marshalPayload(makePayloadV("state", marshalStateValue("volume", map[string]any{
		"volume": volume,
	})))
}

func marshalPaused() []byte {
	return marshalPayload(makePayloadV("state", marshalStateValue("paused", nil)))
}

func marshalResumed() []byte {
	return marshalPayload(makePayloadV("state", marshalStateValue("resumed", nil)))
}

func marshalAudioData(data []byte) []byte {
	return marshalPayload(makePayloadV("state", marshalStateValue("audioData", map[string]any{
		"data": data,
	})))
}

func marshalModeChanged(repeat string, shuffle bool) []byte {
	return marshalPayload(makePayloadV("state", marshalStateValue("modeChanged", map[string]any{
		"repeat":  repeat,
		"shuffle": shuffle,
	})))
}
