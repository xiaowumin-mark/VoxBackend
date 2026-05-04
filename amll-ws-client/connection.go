package amllwsclient

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Rust: CONNECT_TIMEOUT_DURATION = 10s
	connectTimeout = 10 * time.Second

	// Rust: APP_PING_INTERVAL = 5s
	appPingInterval = 5 * time.Second

	// Rust: tokio_channel(CHANNEL_BUFFER_SIZE) = 32
	channelBufferSize = 32

	writeWait = 10 * time.Second
)

type conn struct {
	url    string
	dialer *websocket.Dialer

	ws *websocket.Conn

	outgoing  chan []byte
	shutdown  chan struct{}
	closeOnce sync.Once

	handler commandHandler

	readDone chan struct{}
}

type commandHandler interface {
	OnCommand(cmd *Command)
	OnStatus(st Status)
}

func newConn(url string, dialer *websocket.Dialer, handler commandHandler) *conn {
	return &conn{
		url:      url,
		dialer:   dialer,
		outgoing: make(chan []byte, channelBufferSize),
		shutdown: make(chan struct{}),
		handler:  handler,
		readDone: make(chan struct{}),
	}
}

// Rust: connect_async with CONNECT_TIMEOUT_DURATION
func (c *conn) dial(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, connectTimeout)
	defer cancel()

	ws, resp, err := c.dialer.DialContext(ctx, c.url, http.Header{})
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("连接超时 (超过 %v)", connectTimeout)
		}
		return fmt.Errorf("连接握手失败: %w", err)
	}
	if resp != nil {
		log.Printf("[WebSocket 客户端] 成功连接到服务器。HTTP 状态码: %d", resp.StatusCode)
	}
	c.ws = ws
	return nil
}

// Rust: handle_connection — 单 select! 事件循环
func (c *conn) run() error {
	// Rust: tokio::select! biased; shutdown_rx 优先级最高
	// Go select 不支持 biased，用外层非阻塞检查 + 双重 select 模拟

	rawMsgs := make(chan []byte, channelBufferSize)
	readErr := make(chan error, 1)

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(rawMsgs)
		for {
			_, msg, err := c.ws.ReadMessage()
			if err != nil {
				select {
				case readErr <- err:
				default:
				}
				return
			}
			select {
			case rawMsgs <- msg:
			case <-c.shutdown:
				return
			}
		}
	}()

	// Rust: APP_PING_INTERVAL ticker
	pingTicker := time.NewTicker(appPingInterval)
	defer pingTicker.Stop()

	// Rust: waiting_for_app_pong
	waitingForPong := false

	var runErr error

	// Rust: loop { tokio::select! { ... } }
	for {
		// 模拟 biased select：外层优先检查 shutdown
		select {
		case <-c.shutdown:
			c.ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			wg.Wait()
			return nil
		default:
		}

		select {
		case <-c.shutdown:
			c.ws.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			wg.Wait()
			return nil

		case msg, ok := <-c.outgoing:
			if !ok {
				runErr = fmt.Errorf("主发送通道已关闭")
				goto done
			}
			if err := c.ws.WriteMessage(websocket.TextMessage, msg); err != nil {
				runErr = fmt.Errorf("发送消息失败: %w", err)
				goto done
			}

		case raw, ok := <-rawMsgs:
			if !ok {
				runErr = fmt.Errorf("WebSocket 流已关闭")
				goto done
			}
			if err := c.handleMessage(raw, &waitingForPong); err != nil {
				runErr = err
				goto done
			}

		case <-pingTicker.C:
			// Rust: 发送应用层 Ping
			if err := c.ws.WriteMessage(websocket.TextMessage, marshalPing()); err != nil {
				runErr = fmt.Errorf("发送 Ping 失败: %w", err)
				goto done
			}
			waitingForPong = true

		case err := <-readErr:
			runErr = fmt.Errorf("WebSocket 读取错误: %w", err)
			goto done
		}
	}
done:
	wg.Wait()
	return runErr
}

// Rust: handle_ws_message + handle_v2_message
func (c *conn) handleMessage(raw []byte, waitingForPong *bool) error {
	msgType, value, err := ParseIncoming(raw)
	if err != nil {
		return fmt.Errorf("反序列化服务器消息失败: %w", err)
	}
	switch PayloadType(msgType) {
	case PayloadPing:
		// Rust: trace!("收到服务器的 Ping。回复 Pong。")
		pong := marshalPong()
		if err := c.ws.WriteMessage(websocket.TextMessage, pong); err != nil {
			return fmt.Errorf("回复 Pong 失败: %w", err)
		}

	case PayloadPong:
		// Rust: trace!("收到服务器的 Pong。")
		*waitingForPong = false

	case PayloadCommand:
		cmd, err := ParseCommand(value)
		if err != nil {
			return fmt.Errorf("解析命令失败: %w", err)
		}
		switch cmd.Type {
		case CmdPause, CmdResume, CmdForwardSong, CmdBackwardSong:
			// Rust: last_seek_request_info = None
		case CmdSetVolume:
			if cmd.Volume < 0 || cmd.Volume > 1 {
				return fmt.Errorf("收到无效的音量值: %f", cmd.Volume)
			}
		}
		c.handler.OnCommand(cmd)

	case PayloadInitialize, PayloadState:
		// Rust: warn!("收到意外的 Initialize/State 消息 (应该是我们发送的)")
		log.Printf("[WebSocket 客户端] 收到意外的 %s 消息 (应该是我们发送的)", msgType)
	}

	return nil
}

func (c *conn) close() {
	c.closeOnce.Do(func() {
		close(c.shutdown)
	})
}

func (c *conn) send(msg []byte) bool {
	select {
	case c.outgoing <- msg:
		return true
	default:
		return false
	}
}
