package main

import (
	"fmt"
	"time"

	"github.com/doquangtan/socketio/v4"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/xiaowumin-mark/VoxBackend/player"
)

var Io *socketio.Io

func StartServer() {
	Io = socketio.New()

	Io.OnConnection(func(socket *socketio.Socket) {
		Con.Log("🔗 插件已连接")

		socket.On("songs", func(event *socketio.EventPayload) {
			raw, ok := event.Data[0].([]interface{})
			if !ok {
				fmt.Println("invalid payload type:", event.Data)
				return
			}

			tracks := make([]player.Track, 0, len(raw))
			for _, v := range raw {
				song, ok := v.(map[string]interface{})
				if !ok {
					continue
				}
				tracks = append(tracks, player.Track{
					Path:   song["filePath"].(string),
					Title:  song["songName"].(string),
					Album:  song["songAlbum"].(string),
					Artist: song["songArtists"].(string),
					Meta:   song,
				})
			}
			Con.Log("📋 已加载 %d 首歌曲", len(tracks))
			Player.AddTracks(tracks...)
		})

		socket.On("play", func(event *socketio.EventPayload) {
			Player.SetPaused(false)
		})
		socket.On("pause", func(event *socketio.EventPayload) {
			Player.SetPaused(true)
		})

		socket.On("seek", func(event *socketio.EventPayload) {
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SeekTo(time.Duration(p * float64(time.Second)))
		})
		socket.On("volume", func(event *socketio.EventPayload) {
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetMasterVolume(p)
			Clientws.SendVolume(p)
		})
		socket.On("vocal-gain", func(event *socketio.EventPayload) {
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetVocalGain(p)
		})
		socket.On("crossfade", func(event *socketio.EventPayload) {
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetCrossfade(time.Duration(p * float64(time.Second)))
		})

		socket.On("dsp", func(event *socketio.EventPayload) {
			p, ok := event.Data[0].(string)
			if !ok {
				return
			}
			Player.SetDSPMode(player.DSPMode(p))
		})

		socket.On("vocal-gain-ramp", func(event *socketio.EventPayload) {
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetVocalRamp(time.Duration(p * float64(time.Millisecond)))
		})

		socket.On("rm-all-songs", func(event *socketio.EventPayload) {
			Player.ClearPlaylist()
		})

		socket.On("rm", func(event *socketio.EventPayload) {
			idx, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.RemoveTrack(int(idx))
		})

		socket.On("mv", func(event *socketio.EventPayload) {
			from, _ := event.Data[0].(float64)
			to, _ := event.Data[1].(float64)
			Player.MoveTrack(int(from), int(to))
		})

		socket.On("shuffle-upcoming", func(event *socketio.EventPayload) {
			Player.ShuffleUpcoming()
		})

		socket.On("disconnect", func(event *socketio.EventPayload) {
			Con.Log("💔 插件已断开")
		})
	})

	router := gin.Default()

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	router.GET("/socket.io/*any", gin.WrapH(Io.HttpHandler()))
	router.Run(":54199")

}

func StopServer() {
	Io.Close()
}
