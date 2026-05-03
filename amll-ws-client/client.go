package amllwsclient

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type Status int

const (
	StatusDisconnected Status = iota
	StatusConnecting
	StatusConnected
	StatusError
)

func (s Status) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusError:
		return "error"
	}
	return fmt.Sprintf("unknown(%d)", s)
}

type Config struct {
	URL            string
	ConnectTimeout time.Duration

	OnCommand      func(cmd *Command)
	OnStatusChange func(st Status)
}

func (c Config) withDefaults() Config {
	if c.URL == "" {
		c.URL = "ws://localhost:11444"
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = connectTimeout
	}
	return c
}

type Client struct {
	cfg   Config
	inner *conn
	ctx   context.Context

	status atomic.Int32

	mu       sync.Mutex
	done     chan struct{} // closed on complete shutdown
	cancelFn context.CancelFunc

	// Rust: seek debounce
	lastSeekTime time.Time
	seekMu       sync.Mutex

	// Rust: volume throttle (100ms)
	lastVolumeTime time.Time
	volMu          sync.Mutex

	// Rust: audio data throttle (10ms)
	lastAudioTime time.Time
	audioMu       sync.Mutex
}

func New(cfg Config) *Client {
	cfg = cfg.withDefaults()
	ctx, cancel := context.WithCancel(context.Background())
	c := &Client{
		cfg:      cfg,
		ctx:      ctx,
		done:     make(chan struct{}),
		cancelFn: cancel,
	}
	c.setStatus(StatusDisconnected)
	return c
}

// Rust: run_websocket_client — 异步连接，立即返回
func (c *Client) Connect() {
	go c.connectLoop()
}

// Rust: shutdown_rx → 优雅关闭
func (c *Client) Close() {
	c.cancelFn()
	if c.inner != nil {
		c.inner.close()
	}
	<-c.done
}

// ── 发送（对齐 Rust worker.rs send_*_to_ws） ──

func (c *Client) SendInitialize() {
	c.sendRaw(marshalInitialize())
}

func (c *Client) SendMusic(info MusicInfo) {
	c.sendRaw(marshalSetMusic(info))
}

func (c *Client) SendProgress(ms uint64) {
	c.sendRaw(marshalProgress(ms))
}

func (c *Client) SendPaused() {
	c.sendRaw(marshalPaused())
}

func (c *Client) SendResumed() {
	c.sendRaw(marshalResumed())
}

func (c *Client) SendVolume(vol float64) {
	// Rust: MIN_VOLUME_SET_INTERVAL = 100ms throttle
	c.volMu.Lock()
	if time.Since(c.lastVolumeTime) < 100*time.Millisecond {
		c.volMu.Unlock()
		return
	}
	c.lastVolumeTime = time.Now()
	c.volMu.Unlock()
	c.sendRaw(marshalVolume(vol))
}

func (c *Client) SendCover(cover AlbumCover) {
	c.sendRaw(marshalSetCover(cover))
}

func (c *Client) SendLyric(content LyricContent) {
	c.sendRaw(marshalSetLyric(content))
}

func (c *Client) SendModeChanged(repeat string, shuffle bool) {
	c.sendRaw(marshalModeChanged(repeat, shuffle))
}

// SendAudioData 发送音频数据（Rust: f32→i16 转换 + 10ms 节流）
func (c *Client) SendAudioData(data []byte) {
	// Rust: AUDIO_SEND_INTERVAL = 10ms
	c.audioMu.Lock()
	if time.Since(c.lastAudioTime) < 10*time.Millisecond {
		c.audioMu.Unlock()
		return
	}
	c.lastAudioTime = time.Now()
	c.audioMu.Unlock()
	c.sendRaw(marshalAudioData(data))
}

func (c *Client) sendRaw(msg []byte) {
	c.mu.Lock()
	inner := c.inner
	c.mu.Unlock()
	if inner != nil {
		inner.send(msg)
	}
}

// ── 内部连接循环（对齐 Rust worker.rs amll_connector_actor 的连接管理） ──

func (c *Client) connectLoop() {
	defer close(c.done)

	for {
		if c.ctx.Err() != nil {
			c.setStatus(StatusDisconnected)
			return
		}

		c.setStatus(StatusConnecting)

		dialer := &websocket.Dialer{
			HandshakeTimeout: c.cfg.ConnectTimeout,
		}
		inner := newConn(c.cfg.URL, dialer, c)
		if err := inner.dial(c.ctx); err != nil {
			c.setStatus(StatusError)
			log.Printf("[WebSocket 客户端] 连接失败: %v", err)
			return
		}

		c.mu.Lock()
		c.inner = inner
		c.mu.Unlock()

		// Rust: status_tx.send(WebsocketStatus::Connected)
		c.setStatus(StatusConnected)

		// Rust: 发送 Initialize → sleep 50ms → push_full_state
		inner.send(marshalInitialize())

		runErr := inner.run()

		// Rust: 连接结束，重置 session_ready
		c.mu.Lock()
		c.inner = nil
		c.mu.Unlock()

		if c.ctx.Err() != nil {
			// Rust: Ok(_) → 正常退出
			c.setStatus(StatusDisconnected)
			return
		}

		if runErr != nil {
			// Rust: 连接流错误 / 服务器关闭
			log.Printf("[WebSocket 客户端] 连接异常: %v", runErr)
			c.setStatus(StatusError)
		} else {
			c.setStatus(StatusDisconnected)
		}
		return
	}
}

func (c *Client) setStatus(st Status) {
	c.status.Store(int32(st))
	if c.cfg.OnStatusChange != nil {
		c.cfg.OnStatusChange(st)
	}
}

// Status 返回当前连接状态（对齐 Rust WebsocketStatus）
func (c *Client) Status() Status {
	return Status(c.status.Load())
}

// ── commandHandler 接口实现 ──

func (c *Client) OnCommand(cmd *Command) {
	if cmd == nil {
		return
	}

	// Rust: SEEK_DEBOUNCE_DURATION = 500ms
	if cmd.Type == CmdSeekPlayProgress {
		c.seekMu.Lock()
		if time.Since(c.lastSeekTime) < 500*time.Millisecond {
			c.seekMu.Unlock()
			return
		}
		c.lastSeekTime = time.Now()
		c.seekMu.Unlock()
	}

	// Rust: 检查 volume range
	if cmd.Type == CmdSetVolume && (cmd.Volume < 0 || cmd.Volume > 1) {
		log.Printf("[WebSocket 客户端] 收到无效的音量值: %f", cmd.Volume)
		return
	}

	if c.cfg.OnCommand != nil {
		c.cfg.OnCommand(cmd)
	}
}

func (c *Client) OnStatus(st Status) {
	// conn 内部状态变更通过 Config.OnStatusChange 回调
	if c.cfg.OnStatusChange != nil {
		c.cfg.OnStatusChange(st)
	}
}
