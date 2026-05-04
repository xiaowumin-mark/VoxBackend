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
		fmt.Println("OnConnection")

		socket.On("need-cfg", func(event *socketio.EventPayload) {
			fmt.Println("OnNeedCfg")
			Io.Emit("cfg", map[string]any{
				"VocalGain":        Cfg.MasterVolume,
				"MasterVolume":     Cfg.MasterVolume,
				"Crossfade":        Cfg.Crossfade.Seconds(),
				"Prewarm":          Cfg.Prewarm.Seconds(),
				"VocalGainRamp":    Cfg.VocalGainRamp.Milliseconds(),
				"SeparatorMode":    Cfg.SeparatorMode,
				"ONNXModel":        Cfg.ONNX.ModelPath,
				"ONNXProfile":      Cfg.ONNX.Profile,
				"ONNXCompensation": Cfg.ONNX.Compensation,
				"ONNXOtherModel":   Cfg.ONNX.OtherModelPath,
				"ONNXStepFrames":   Cfg.ONNX.StepFrames,
				"DSPMode":          Cfg.DSP.Mode,
			})
		})

		socket.On("songs", func(event *socketio.EventPayload) {
			fmt.Println("OnSongs")
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
			Player.AddTracks(tracks...)
		})

		socket.On("play", func(event *socketio.EventPayload) {
			fmt.Println("OnPlay")
			Player.SetPaused(false)
		})
		socket.On("pause", func(event *socketio.EventPayload) {
			fmt.Println("OnPause")
			Player.SetPaused(true)
		})

		socket.On("seek", func(event *socketio.EventPayload) {
			//fmt.Println("OnSeek")
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SeekTo(time.Duration(p * float64(time.Second)))
			fmt.Println("OnSeek", p)

		})
		socket.On("volume", func(event *socketio.EventPayload) {
			//fmt.Println("OnVolume")
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetMasterVolume(p)
			Clientws.SendVolume(p)
			fmt.Println("OnVolume", p)

		})
		socket.On("vocal-gain", func(event *socketio.EventPayload) {
			//fmt.Println("OnVocalGain")
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetVocalGain(p)
			fmt.Println("OnVocalGain", p)
		})
		socket.On("crossfade", func(event *socketio.EventPayload) {
			//fmt.Println("OnCrossfade")
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetCrossfade(time.Duration(p * float64(time.Second)))
			fmt.Println("OnCrossfade", p)
		})

		socket.On("dsp", func(event *socketio.EventPayload) {
			//fmt.Println("dsp")
			p, ok := event.Data[0].(string)
			if !ok {
				return
			}
			Player.SetDSPMode(player.DSPMode(p))
			fmt.Println("dsp", p)
		})

		//vocalGainRamp
		socket.On("vocal-gain-ramp", func(event *socketio.EventPayload) {
			//fmt.Println("OnVocalGainRamp")
			p, ok := event.Data[0].(float64)
			if !ok {
				return
			}
			Player.SetVocalRamp(time.Duration(p * float64(time.Millisecond)))
			fmt.Println("OnVocalGainRamp", p)
		})

		socket.On("rm-all-songs", func(event *socketio.EventPayload) {
			fmt.Println("OnRmAllSongs")
			Player.ClearPlaylist()
		})

		socket.On("disconnect", func(event *socketio.EventPayload) {
			fmt.Println("OnDisconnect")
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
