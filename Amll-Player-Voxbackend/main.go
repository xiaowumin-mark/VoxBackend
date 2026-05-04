package main

import (
	"context"
	"log"
	"time"

	amllwsclient "github.com/xiaowumin-mark/VoxBackend/amll-ws-client"
	"github.com/xiaowumin-mark/VoxBackend/player"
	"github.com/xiaowumin-mark/VoxBackend/separator"
)

var Cfg = player.DefaultConfig()
var Player *player.Player
var Clientws *amllwsclient.Client

func main() {

	Cfg.SeparatorMode = player.SeparatorONNX
	Cfg.ONNX.ModelPath = "./onnx/umxl_vocals.onnx"
	Cfg.ONNX.RuntimeLibraryPath = "./onnxruntime/lib/onnxruntime.dll"
	if err := separator.InitONNXRuntime(Cfg.ONNX.RuntimeLibraryPath); err != nil {
		panic(err)
	}
	Cfg.Callbacks.OnEvent = func(ev player.Event) {
		if Io != nil {
			Io.Emit("Event", map[string]any{
				"type":    ev.Type,
				"message": ev.Message,
			})
		}
		log.Printf("[%s] %s", ev.Type, ev.Message)
		if ev.Type == player.EventError {
			log.Printf("[%s] %s: %v", ev.Type, ev.Message, ev.Err)
			return
		}
		if ev.TrackPath != "" {
			log.Printf("[%s] #%d %s - %s", ev.Type, ev.TrackIndex, ev.TrackPath, ev.Message)
			return
		}

	}
	Cfg.Callbacks.OnPausedChanged = func(s player.State) {
		if Io == nil {
			return
		}
		Io.Emit("PausedChanged", map[string]any{
			"paused":   s.Paused,
			"position": s.Position.Milliseconds(),
			"duration": s.Duration.Milliseconds(),
			//"id":       s.Track.Meta["id"],
		})
		if s.Paused {
			Clientws.SendPaused()
		} else {
			Clientws.SendResumed()
		}
	}
	Cfg.Callbacks.OnTrackChanged = func(s player.State) {
		if Io == nil {
			return
		}
		Io.Emit("OnTrackChanged", map[string]any{
			"paused":   s.Paused,
			"position": s.Position.Milliseconds(),
			"duration": s.Duration.Milliseconds(),
			"id":       s.Track.Meta["id"],
		})
		Clientws.SendMusic(amllwsclient.MusicInfo{
			MusicID:   s.Track.Meta["id"].(string),
			Artists:   []amllwsclient.Artist{{Name: s.Track.Album}},
			AlbumName: s.Track.Album,
			MusicName: s.Track.Title,
			Duration:  uint64(s.Duration.Milliseconds()),
		})
	}
	Cfg.Callbacks.OnState = func(s player.State) {
		if Io == nil {
			return
		}
		Io.Emit("OnState", map[string]any{
			"paused":   s.Paused,
			"position": s.Position.Milliseconds(),
			"duration": s.Duration.Milliseconds(),
			//"id":       s.Track.Meta["id"].(string),
		})
		Clientws.SendProgress(uint64(s.Position.Milliseconds()))

		if !s.Paused {
			log.Println(s.Position.Milliseconds())
		}

	}
	Cfg.Callbacks.OnVolumeChanged = func(s player.State) {
		if Io == nil {
			return
		}
		Io.Emit("OnVolumeChanged", map[string]any{
			"volume": s.Volume,
		})
	}
	Cfg.Callbacks.OnVocalChanged = func(s player.State) {
		if Io == nil {
			return
		}
		Io.Emit("OnVocalChanged", map[string]any{
			"vocalGain": s.VocalGain,
		})
	}
	Player = player.New(Cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Player.Start(ctx); err != nil {
		panic(err)
	}
	go StartServer()

	Clientws = amllwsclient.New(amllwsclient.Config{
		URL:               "ws://localhost:11444",
		ReconnectInterval: 1 * time.Second,
		OnConnected: func() {
			s := Player.Snapshot()
			if s.Track != nil {
				var musicID string
				if id, ok := s.Track.Meta["id"].(string); ok {
					musicID = id
				}
				Clientws.SendMusic(amllwsclient.MusicInfo{
					MusicID:   musicID,
					MusicName: s.Track.Title,
					AlbumName: s.Track.Album,
					Artists:   []amllwsclient.Artist{{Name: s.Track.Artist}},
					Duration:  uint64(s.Duration.Milliseconds()),
				})
				Clientws.SendProgress(uint64(s.Position.Milliseconds()))
				if !s.Paused {
					Clientws.SendPaused()
				} else {
					Clientws.SendResumed()
				}
			}
			Clientws.SendVolume(Cfg.MasterVolume)
		},
		OnCommand: func(cmd *amllwsclient.Command) {
			switch cmd.Type {
			case amllwsclient.CmdPause:
				Player.SetPaused(true)
			case amllwsclient.CmdResume:
				Player.SetPaused(false)
			case amllwsclient.CmdForwardSong:
				Player.Next()
				// 下一首
			case amllwsclient.CmdBackwardSong:
				Player.Previous()
				// 上一首
			case amllwsclient.CmdSetVolume:
				Player.SetMasterVolume(cmd.Volume)
				log.Printf("设置音量: %.2f", cmd.Volume)
				// cmd.Volume 新音量 (0.0-1.0)
			case amllwsclient.CmdSeekPlayProgress:
				Player.SeekTo((time.Duration(cmd.Progress) / 1000) * time.Second)
				// cmd.Progress 跳转位置 (ms)
			case amllwsclient.CmdSetRepeatMode:
				// cmd.Mode 重复模式 ("off"/"all"/"one")
			case amllwsclient.CmdSetShuffleMode:
				// cmd.Enabled 随机播放开关
			}
		},
		OnStatusChange: func(st amllwsclient.Status) {
			// 连接状态变更
			switch st {
			case amllwsclient.StatusConnecting:
				log.Printf("正在连接")
			case amllwsclient.StatusConnected:
				log.Printf("已连接")

			case amllwsclient.StatusDisconnected:
				log.Printf("已断开")
			}
		},
	})
	Clientws.Connect()     // 异步连接，立即返回
	defer Clientws.Close() // 优雅关闭

	if err := Player.Wait(); err != nil {
		panic(err)
	}
	StopServer()
}
