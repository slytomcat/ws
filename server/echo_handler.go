package server

import (
	"fmt"

	"github.com/gorilla/websocket"
)

// EchoHandler is a handler that sends back all received messages
func EchoHandler(conn *websocket.Conn) {
	id := conn.RemoteAddr().String()
	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseServiceRestart) {
				fmt.Printf("echoHandler for %s: websocket reading error: %v\n", id, err)
			} else {
				fmt.Printf("echoHandler for %s: %s\n", id, err)
			}
			return
		}
		fmt.Printf("echoHandler for %s: handle message: %s\n", id, message)
		conn.WriteMessage(mt, message)
	}
}
