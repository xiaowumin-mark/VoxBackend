package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/doquangtan/socketio/v4"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/xiaowumin-mark/VoxBackend/player"
)

var Io *socketio.Io
var pluginServer *http.Server
var pluginServerMu sync.Mutex

func StartServer(addr string) {
	if strings.TrimSpace(addr) == "" {
		addr = ":54199"
	}
	if !strings.Contains(addr, ":") {
		addr = ":" + addr
	}

	pluginServerMu.Lock()
	if pluginServer != nil {
		pluginServerMu.Unlock()
		logServer("插件服务已在运行")
		return
	}
	pluginServerMu.Unlock()

	Io = socketio.New()

	Io.OnConnection(func(socket *socketio.Socket) {
		logServer("插件已连接")

		socket.On("songs", func(event *socketio.EventPayload) {
			data, ok := payloadAt(event, 0)
			if !ok {
				return
			}
			raw, ok := data.([]interface{})
			if !ok {
				logServer("无效 songs payload: %T", data)
				return
			}

			tracks := make([]player.Track, 0, len(raw))
			for _, v := range raw {
				song, ok := v.(map[string]interface{})
				if !ok {
					continue
				}
				path := songString(song, "filePath")
				if path == "" {
					continue
				}
				tracks = append(tracks, player.Track{
					Path:   path,
					Title:  songString(song, "songName"),
					Album:  songString(song, "songAlbum"),
					Artist: songString(song, "songArtists"),
					Meta:   cloneMeta(song),
				})
			}
			logServer("已从插件加入 %d 首歌曲", len(tracks))
			if Player != nil {
				Player.AddTracks(tracks...)
			}
		})

		socket.On("play", func(event *socketio.EventPayload) {
			if Player != nil {
				Player.SetPaused(false)
			}
		})
		socket.On("pause", func(event *socketio.EventPayload) {
			if Player != nil {
				Player.SetPaused(true)
			}
		})
		socket.On("seek", func(event *socketio.EventPayload) {
			p, ok := payloadFloat(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.SeekTo(time.Duration(p * float64(time.Second)))
		})
		socket.On("volume", func(event *socketio.EventPayload) {
			p, ok := payloadFloat(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.SetMasterVolume(p)
			if Clientws != nil {
				Clientws.SendVolume(p)
			}
		})
		socket.On("vocal-gain", func(event *socketio.EventPayload) {
			p, ok := payloadFloat(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.SetVocalGain(p)
		})
		socket.On("crossfade", func(event *socketio.EventPayload) {
			p, ok := payloadFloat(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.SetCrossfade(time.Duration(p * float64(time.Second)))
		})
		socket.On("dsp", func(event *socketio.EventPayload) {
			p, ok := payloadString(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.SetDSPMode(player.DSPMode(p))
		})
		socket.On("vocal-gain-ramp", func(event *socketio.EventPayload) {
			p, ok := payloadFloat(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.SetVocalRamp(time.Duration(p * float64(time.Millisecond)))
		})
		socket.On("rm-all-songs", func(event *socketio.EventPayload) {
			if Player != nil {
				Player.ClearPlaylist()
			}
		})
		socket.On("rm", func(event *socketio.EventPayload) {
			idx, ok := payloadFloat(event, 0)
			if !ok || Player == nil {
				return
			}
			Player.RemoveTrack(int(idx))
		})
		socket.On("mv", func(event *socketio.EventPayload) {
			from, okFrom := payloadFloat(event, 0)
			to, okTo := payloadFloat(event, 1)
			if !okFrom || !okTo || Player == nil {
				return
			}
			Player.MoveTrack(int(from), int(to))
		})
		socket.On("shuffle-upcoming", func(event *socketio.EventPayload) {
			if Player != nil {
				if idx, ok := shuffleCurrentIndex(event); ok {
					Player.ShuffleUpcomingFrom(idx)
					return
				}
				Player.ShuffleUpcoming()
			}
		})
		socket.On("disconnect", func(event *socketio.EventPayload) {
			logServer("插件已断开")
		})
	})

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.GET("/socket.io/*any", gin.WrapH(Io.HttpHandler()))
	server := &http.Server{Addr: addr, Handler: router}
	pluginServerMu.Lock()
	if pluginServer != nil {
		pluginServerMu.Unlock()
		logServer("插件服务已在运行")
		return
	}
	pluginServer = server
	pluginServerMu.Unlock()

	logServer("插件服务监听 %s", addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		logServer("插件服务停止: %v", err)
	}
	pluginServerMu.Lock()
	if pluginServer == server {
		pluginServer = nil
	}
	pluginServerMu.Unlock()
}

func StopServer() {
	if Io != nil {
		Io.Close()
		Io = nil
	}
	pluginServerMu.Lock()
	server := pluginServer
	pluginServerMu.Unlock()
	if server == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logServer("插件服务关闭失败: %v", err)
	}
}

func payloadAt(event *socketio.EventPayload, index int) (interface{}, bool) {
	if event == nil || index < 0 || index >= len(event.Data) {
		return nil, false
	}
	return event.Data[index], true
}

func payloadFloat(event *socketio.EventPayload, index int) (float64, bool) {
	v, ok := payloadAt(event, index)
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

func payloadString(event *socketio.EventPayload, index int) (string, bool) {
	v, ok := payloadAt(event, index)
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

func shuffleCurrentIndex(event *socketio.EventPayload) (int, bool) {
	data, ok := payloadAt(event, 0)
	if !ok {
		return -1, false
	}
	payload, ok := data.(map[string]interface{})
	if !ok {
		return -1, false
	}
	v, ok := payload["currentIndex"]
	if !ok {
		return -1, false
	}
	idx, ok := numberToInt(v)
	if !ok || idx < 0 {
		return -1, false
	}
	return idx, true
}

func numberToInt(v interface{}) (int, bool) {
	switch n := v.(type) {
	case float64:
		return int(n), true
	case float32:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	case string:
		var idx int
		if _, err := fmt.Sscanf(n, "%d", &idx); err == nil {
			return idx, true
		}
	}
	return -1, false
}

func songString(song map[string]interface{}, key string) string {
	v, ok := song[key]
	if !ok {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprint(v)
}

func cloneMeta(src map[string]interface{}) map[string]any {
	dst := make(map[string]any, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func logServer(format string, args ...interface{}) {
	uiLog(format, args...)
}
