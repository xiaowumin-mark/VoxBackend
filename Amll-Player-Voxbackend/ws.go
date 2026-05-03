package main

import (
	"fmt"
	"time"

	"github.com/doquangtan/socketio/v4"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var Io *socketio.Io

func StartServer() {
	Io = socketio.New()

	Io.OnConnection(func(socket *socketio.Socket) {
		fmt.Println("OnConnection")

		socket.On("message", func(event *socketio.EventPayload) {
			socket.Emit("message", event.Data...)
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
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://github.com"
		},
		MaxAge: 12 * time.Hour,
	}))

	router.GET("/socket.io/*any", gin.WrapH(Io.HttpHandler()))
	router.Run(":54199")

}

func StopServer() {
	Io.Close()
}
