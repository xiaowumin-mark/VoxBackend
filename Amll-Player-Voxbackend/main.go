package main

import (
	"context"
	"fmt"
	"time"

	amllwsclient "github.com/xiaowumin-mark/VoxBackend/amll-ws-client"
	"github.com/xiaowumin-mark/VoxBackend/console"
	"github.com/xiaowumin-mark/VoxBackend/player"
	"github.com/xiaowumin-mark/VoxBackend/separator"
)

var Cfg = player.DefaultConfig()
var Player *player.Player
var Clientws *amllwsclient.Client
var Con *console.Console

func main() {
	Con = console.New()
	defer Con.Close()

	Cfg.SeparatorMode = player.SeparatorONNX
	Cfg.ONNX.ModelPath = "./onnx/umxl_vocals.onnx"
	Cfg.ONNX.RuntimeLibraryPath = "./onnxruntime/lib/onnxruntime.dll"

	Con.Log("🔧 分离器: ONNX (umxl_vocals)")
	Con.Log("🔊 缓冲: %d", Cfg.SpeakerBufferSize)
	Con.Log("🎚️ DSP: %s | Crossfade %.0fs | Prewarm %.0fs", Cfg.DSP.Mode, Cfg.Crossfade.Seconds(), Cfg.Prewarm.Seconds())

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
		if ev.Type == player.EventError {
			Con.Log("❌ [%s] %s: %v", ev.Type, ev.Message, ev.Err)
			return
		}
		if ev.Type == player.EventTrackChanged {
			if ev.Track != nil {
				Con.Log("🎵 正在播放: %s — %s", ev.Track.Title, ev.Track.Artist)
			}
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
		})
		Clientws.SendProgress(uint64(s.Position.Milliseconds()))

		if s.Track != nil {
			status := fmt.Sprintf("🎵 %s — %s  %s / %s   🔊 %.0f%% 🎤 %.0f%%",
				truncateStr(s.Track.Title, 30),
				truncateStr(s.Track.Artist, 20),
				formatDuration(s.Position),
				formatDuration(s.Duration),
				s.Volume*100,
				s.VocalGain*100,
			)
			if Io != nil {
				status += "  🔗 ●"
			}
			if Clientws != nil {
				status += "  🌐 ●"
			}
			if s.Paused {
				status += "  ⏸"
			}
			Con.SetStatus(status)
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
	Cfg.Callbacks.OnPlaylistChanged = func(s player.State) {
		if Io == nil {
			return
		}
		tracks := Player.Playlist()
		list := make([]map[string]any, 0, len(tracks))
		for _, t := range tracks {
			list = append(list, map[string]any{
				"id":          t.Meta["id"],
				"duration":    t.Meta["duration"],
				"filePath":    t.Path,
				"songAlbum":   t.Album,
				"songArtists": t.Artist,
				"songName":    t.Title,
			})
		}
		Io.Emit("playlist", list)
	}
	Player = player.New(Cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := Player.Start(ctx); err != nil {
		panic(err)
	}
	go StartServer()

	Con.Log("🌐 AMLL WS: localhost:11444")
	Con.Log("🔌 插件端口: :54199")

	Clientws = amllwsclient.New(amllwsclient.Config{
		URL:               "ws://localhost:11444",
		ReconnectInterval: 1 * time.Second,
		OnConnected: func() {
			Con.Log("🌐 AMLL Player 已连接")
			Clientws.SendVolume(Cfg.MasterVolume)
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
		},
		OnCommand: func(cmd *amllwsclient.Command) {
			switch cmd.Type {
			case amllwsclient.CmdPause:
				Player.SetPaused(true)
			case amllwsclient.CmdResume:
				Player.SetPaused(false)
			case amllwsclient.CmdForwardSong:
				Player.Next()
			case amllwsclient.CmdBackwardSong:
				Player.Previous()
			case amllwsclient.CmdSetVolume:
				Player.SetMasterVolume(cmd.Volume)
				Con.Log("🔊 音量: %.0f%%", cmd.Volume*100)
			case amllwsclient.CmdSeekPlayProgress:
				Player.SeekTo((time.Duration(cmd.Progress) / 1000) * time.Second)
			}
		},
		OnStatusChange: func(st amllwsclient.Status) {
			switch st {
			case amllwsclient.StatusConnecting:
				Con.Log("🌐 正在连接 AMLL Player...")
			case amllwsclient.StatusConnected:
				Con.Log("🌐 AMLL Player 已连接")
			case amllwsclient.StatusDisconnected:
				Con.Log("🌐 AMLL Player 已断开")
			}
		},
	})
	Clientws.Connect()
	defer Clientws.Close()

	Con.Log("")
	Con.Log("────────────────────────────────")

	if err := Player.Wait(); err != nil {
		panic(err)
	}
	StopServer()
}

func formatDuration(d time.Duration) string {
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%02d:%02d", m, s)
}

func truncateStr(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n-1]) + "…"
	}
	return s
}
