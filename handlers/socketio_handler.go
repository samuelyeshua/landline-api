package handlers

import (
	"fmt"
	"github.com/asm-products/landline-api/models"
	"github.com/gin-gonic/gin"
	"github.com/googollee/go-socket.io"
	"os"
)

var Socketio_Server *socketio.Server

type sioMessage struct {
	Body string
	Room string
}

type socketIOResponse struct {
	Success bool
	Message string
	Result interface{}
}

func newSuccessResponse(result interface{}) socketIOResponse {
	return socketIOResponse{true, "", result}
}

func newErrorResponse(err error) socketIOResponse {
	return socketIOResponse{false, err.Error(), nil}
}

func newAuthErrorResponse() socketIOResponse {
	return newErrorResponse(fmt.Errorf("Not Authenticated."))
}

func SetupSocketIOServer() {
	var err error
	Socketio_Server, err = socketio.NewServer(nil)
	if err != nil {
		panic(err)
	}
}

func SocketHandler(c *gin.Context) {
	Socketio_Server.On("connection", func(so socketio.Socket) {
		// Since this function is called for every connection,
		// so the user variable is caught in this closure, and is only accessible
		// to this connection.
		var user *models.User

		// The socket.io client doesn't support sending headers, so we can't use
		// the standard auth mechanism. To solve this, before doing anything else,
		// the client should emit an "auth" event, with their JWT as the message.
		so.On("auth", func(token string) socketIOResponse {
			res, err := getUserFromJwt(token, os.Getenv("SECRET"))
			user = res
			if err != nil {
				return newErrorResponse(err)
			}
			joinRoomMemberships(user.Id, so)
			return newSuccessResponse(nil)
		})

		// To receive notifications for a room, the client emits the 'join' event,
		// with the room slug as the message.
		so.On("join", func(roomSlug string) socketIOResponse {
			if user == nil {
				return newAuthErrorResponse()
			}
			membership, err := JoinRoom(user, roomSlug)
			if err != nil {
				newErrorResponse(err)
			}
			err = so.Join(membership.RoomId)
			if err != nil {
				return newErrorResponse(err)
			}
			return newSuccessResponse(membership)
		})

		// Works like the 'join' event.
		so.On("leave", func(roomSlug string) socketIOResponse {
			if user == nil {
				return newAuthErrorResponse()
			}
			rid, err := LeaveRoom(user, roomSlug)
			if err != nil {
				return newErrorResponse(err)
			}
			err = so.Leave(rid)
			if err != nil {
				return newErrorResponse(err)
			}
			return newSuccessResponse(rid)
		})

		so.On("message", func(m *sioMessage) socketIOResponse {
			if user == nil {
				return newAuthErrorResponse()
			}
			msg, err := SendMessage(user, m.Room, m.Body)
			if err != nil {
				return newErrorResponse(err)
			}
			return newSuccessResponse(msg)
		})

		so.On("disconnection", func() {
			fmt.Println("on disconnect")
		})
	})

	Socketio_Server.On("error", func(so socketio.Socket, err error) {
		fmt.Printf("[ WebSocket ] Error : %v", err.Error())
	})

	Socketio_Server.ServeHTTP(c.Writer, c.Request)
}

// A helper function to join all rooms the user is a member of.
func joinRoomMemberships(userId string, so socketio.Socket) error {
	ms, err := models.FindRoomMemberships(userId)
	for _, m := range ms {
		if err != nil {
			return err
		}
		err = so.Join(m)
	}
	return err
}

// Browsers complain when the allowed origin is *, and there are cookies being set, which socket.io requires.
func SocketIOCors(c *gin.Context) {
	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
	}
	c.Next()
}
