package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	amllwsclient "github.com/xiaowumin-mark/VoxBackend/amll-ws-client"
	"github.com/xiaowumin-mark/VoxBackend/player"
)

type amllClient = amllwsclient.Client

type startedMsg struct {
	Config      AppConfig
	Profile     string
	ProfileNote string
}

type appController struct {
	mu       sync.Mutex
	cfgPath  string
	store    *appStore
	player   *player.Player
	client   *amllwsclient.Client
	cancel   context.CancelFunc
	running  bool
	starting bool
}

func newAppController(cfgPath string, store *appStore) *appController {
	return &appController{
		cfgPath: cfgPath,
		store:   store,
	}
}

func (c *appController) Start(cfg AppConfig) {
	c.mu.Lock()
	if c.running || c.starting {
		c.mu.Unlock()
		return
	}
	c.starting = true
	c.mu.Unlock()

	if c.store != nil {
		c.store.SetStarting(true)
	}

	go func() {
		cfg.normalize()
		profile, note, err := resolveConfiguredProfile(cfg.ModelPath)
		if err != nil {
			c.startFailed(err)
			return
		}
		if err := cfg.validateForStart(); err != nil {
			c.startFailed(err)
			return
		}
		if err := saveAppConfig(c.cfgPath, cfg); err != nil {
			c.startFailed(fmt.Errorf("保存配置失败: %w", err))
			return
		}
		if err := c.startRuntime(cfg, profile); err != nil {
			c.startFailed(err)
			return
		}
		if c.store != nil {
			c.store.MarkStarted(startedMsg{Config: cfg, Profile: profile, ProfileNote: note})
		}
	}()
}

func (c *appController) startFailed(err error) {
	c.mu.Lock()
	c.starting = false
	c.mu.Unlock()
	if c.store != nil {
		c.store.MarkStartFailed(err)
	}
}

func (c *appController) startRuntime(appCfg AppConfig, profile string) error {
	c.mu.Lock()
	if c.running {
		c.starting = false
		c.mu.Unlock()
		return nil
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	c.running = true
	c.starting = false
	c.mu.Unlock()

	cfg := player.DefaultConfig()
	cfg.SeparatorMode = player.SeparatorONNX
	cfg.ONNX.ModelPath = appCfg.ModelPath
	cfg.ONNX.RuntimeLibraryPath = appCfg.RuntimeLibraryPath
	cfg.ONNX.Profile = profile
	cfg.Callbacks = c.callbacks()

	p := player.New(cfg)
	if err := p.Start(ctx); err != nil {
		cancel()
		c.markStopped()
		return err
	}

	c.mu.Lock()
	c.player = p
	c.mu.Unlock()
	Player = p

	go StartServer(appCfg.PluginAddr)

	client := c.newAMLLClient(appCfg)
	c.mu.Lock()
	c.client = client
	c.mu.Unlock()
	Clientws = client
	client.Connect()

	go func() {
		err := p.Wait()
		c.Stop()
		if c.store != nil {
			c.store.MarkStopped(err)
		}
	}()
	return nil
}

func (c *appController) callbacks() player.Callbacks {
	return player.Callbacks{
		OnEvent: func(ev player.Event) {
			if Io != nil {
				Io.Emit("Event", map[string]any{
					"type":    ev.Type,
					"message": ev.Message,
				})
			}
			if ev.Type == player.EventError {
				if ev.Err != nil {
					uiLog("[%s] %s: %v", ev.Type, ev.Message, ev.Err)
					return
				}
				uiLog("[%s] %s", ev.Type, ev.Message)
			}
		},
		OnPausedChanged: func(s player.State) {
			if c.store != nil {
				c.store.UpdatePlayerState(s)
			}
			if Io != nil {
				Io.Emit("PausedChanged", map[string]any{
					"paused":   s.Paused,
					"position": s.Position.Milliseconds(),
					"duration": s.Duration.Milliseconds(),
				})
			}
			c.withClient(func(client *amllwsclient.Client) {
				if s.Paused {
					client.SendPaused()
				} else {
					client.SendResumed()
				}
			})
		},
		OnTrackChanged: func(s player.State) {
			if c.store != nil {
				c.store.UpdatePlayerState(s)
			}
			if s.Track == nil {
				return
			}
			musicID := trackMetaString(s.Track, "id")
			if Io != nil {
				Io.Emit("OnTrackChanged", map[string]any{
					"paused":   s.Paused,
					"position": s.Position.Milliseconds(),
					"duration": s.Duration.Milliseconds(),
					"id":       musicID,
					"index":    s.TrackIndex,
				})
			}
			c.withClient(func(client *amllwsclient.Client) {
				client.SendMusic(amllwsclient.MusicInfo{
					MusicID:   musicID,
					Artists:   []amllwsclient.Artist{{Name: s.Track.Artist}},
					AlbumName: s.Track.Album,
					MusicName: s.Track.Title,
					Duration:  uint64(s.Duration.Milliseconds()),
				})
			})
			uiLog("正在播放: %s - %s", s.Track.Title, s.Track.Artist)
		},
		OnState: func(s player.State) {
			if c.store != nil {
				c.store.UpdatePlayerState(s)
			}
			if Io != nil {
				Io.Emit("OnState", map[string]any{
					"paused":   s.Paused,
					"position": s.Position.Milliseconds(),
					"duration": s.Duration.Milliseconds(),
				})
			}
			c.withClient(func(client *amllwsclient.Client) {
				client.SendProgress(uint64(s.Position.Milliseconds()))
			})
		},
		OnVolumeChanged: func(s player.State) {
			if c.store != nil {
				c.store.UpdatePlayerState(s)
			}
			if Io != nil {
				Io.Emit("OnVolumeChanged", map[string]any{"volume": s.Volume})
			}
		},
		OnVocalChanged: func(s player.State) {
			if c.store != nil {
				c.store.UpdatePlayerState(s)
			}
			if Io != nil {
				Io.Emit("OnVocalChanged", map[string]any{"vocalGain": s.VocalGain})
			}
		},
		OnPlaylistChanged: func(s player.State) {
			if c.store != nil {
				c.store.UpdatePlayerState(s)
			}
			if Io == nil || Player == nil {
				return
			}
			tracks := Player.Playlist()
			list := make([]map[string]any, 0, len(tracks))
			for i, t := range tracks {
				list = append(list, map[string]any{
					"id":          trackMetaString(&t, "id"),
					"index":       i,
					"isCurrent":   i == s.TrackIndex,
					"duration":    trackMetaValue(&t, "duration"),
					"filePath":    t.Path,
					"songAlbum":   t.Album,
					"songArtists": t.Artist,
					"songName":    t.Title,
				})
			}
			Io.Emit("playlist", list)
		},
	}
}

func (c *appController) newAMLLClient(appCfg AppConfig) *amllwsclient.Client {
	return amllwsclient.New(amllwsclient.Config{
		URL:               appCfg.AMLLURL,
		ReconnectInterval: time.Second,
		OnConnected: func() {
			uiLog("AMLL Player 已连接")
			c.withPlayer(func(p *player.Player) {
				s := p.Snapshot()
				c.withClient(func(client *amllwsclient.Client) {
					client.SendVolume(s.Volume)
					if s.Track != nil {
						client.SendMusic(amllwsclient.MusicInfo{
							MusicID:   trackMetaString(s.Track, "id"),
							MusicName: s.Track.Title,
							AlbumName: s.Track.Album,
							Artists:   []amllwsclient.Artist{{Name: s.Track.Artist}},
							Duration:  uint64(s.Duration.Milliseconds()),
						})
						client.SendProgress(uint64(s.Position.Milliseconds()))
					}
					if s.Paused {
						client.SendPaused()
					} else {
						client.SendResumed()
					}
				})
			})
		},
		OnCommand: func(cmd *amllwsclient.Command) {
			c.withPlayer(func(p *player.Player) {
				switch cmd.Type {
				case amllwsclient.CmdPause:
					p.SetPaused(true)
				case amllwsclient.CmdResume:
					p.SetPaused(false)
				case amllwsclient.CmdForwardSong:
					p.Next()
				case amllwsclient.CmdBackwardSong:
					p.Previous()
				case amllwsclient.CmdSetVolume:
					p.SetMasterVolume(cmd.Volume)
					uiLog("音量: %.0f%%", cmd.Volume*100)
				case amllwsclient.CmdSeekPlayProgress:
					p.SeekTo(time.Duration(cmd.Progress) * time.Millisecond)
				}
			})
		},
		OnStatusChange: func(st amllwsclient.Status) {
			if c.store != nil {
				c.store.SetAMLLStatus(st)
			}
		},
	})
}

func (c *appController) Stop() {
	c.mu.Lock()
	if !c.running && !c.starting {
		c.mu.Unlock()
		return
	}
	cancel := c.cancel
	playerRef := c.player
	client := c.client
	c.running = false
	c.starting = false
	c.cancel = nil
	c.player = nil
	c.client = nil
	c.mu.Unlock()

	if c.store != nil {
		c.store.SetStarting(false)
	}
	if cancel != nil {
		cancel()
	}
	if client != nil {
		client.Close()
	}
	StopServer()
	if playerRef != nil {
		playerRef.Stop()
	}
	Player = nil
	Clientws = nil
}

func (c *appController) markStopped() {
	c.mu.Lock()
	c.running = false
	c.starting = false
	c.cancel = nil
	c.player = nil
	c.client = nil
	c.mu.Unlock()
	Player = nil
	Clientws = nil
}

func (c *appController) withPlayer(fn func(*player.Player)) {
	c.mu.Lock()
	p := c.player
	c.mu.Unlock()
	if p != nil {
		fn(p)
	}
}

func (c *appController) withClient(fn func(*amllwsclient.Client)) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client != nil {
		fn(client)
	}
}

func (c *appController) TogglePause() {
	c.withPlayer(func(p *player.Player) { p.TogglePaused() })
}

func (c *appController) Next() {
	c.withPlayer(func(p *player.Player) { p.Next() })
}

func (c *appController) Previous() {
	c.withPlayer(func(p *player.Player) { p.Previous() })
}

func (c *appController) Seek(delta time.Duration) {
	c.withPlayer(func(p *player.Player) {
		s := p.Snapshot()
		p.SeekTo(s.Position + delta)
	})
}
