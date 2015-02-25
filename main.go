package main

import (
	"os"
	"runtime"

	"github.com/asm-products/landline-api/handlers"
	"github.com/gin-gonic/gin"
	"github.com/tommy351/gin-cors"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	r := gin.Default()
	r.Use(cors.Middleware(cors.Options{
		AllowMethods: []string{"GET", "OPTIONS"},
	  AllowOrigins: []string{"*"},
	}))

	// Unauthenticated routes
	r.GET("/sessions/new", handlers.SessionsNew)
	r.GET("/sessions/sso", handlers.SessionsLoginSSO)
	r.POST("/teams", handlers.TeamsCreate)
	r.OPTIONS("", func (c *gin.Context) {
    c.JSON(200, gin.H{"ok": "ok"})
	})

	// authenticated routes
	a := r.Group("/")
	a.Use(handlers.Auth(os.Getenv("SECRET")))
	a.GET("rooms", handlers.RoomsIndex)
	a.GET("users/find", handlers.UsersFindOne)

	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "3000"
	}

	r.Run(":" + port)
}
