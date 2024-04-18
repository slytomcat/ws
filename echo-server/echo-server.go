package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/slytomcat/ws/server"
)

type msg struct {
	Type          string `json:"type,omitempty"`
	Payload       string `json:"payload,omitempty"`
	ListenerCount *int   `json:"listenerCount,omitempty"`
}

func sendMsg(m msg, c *websocket.Conn) {
	response, _ := json.Marshal(m)
	c.WriteMessage(websocket.TextMessage, response)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Printf("Usage: %s ws://host[:port][/path]", os.Args[0])
		os.Exit(1)
	}
	u, err := url.Parse(os.Args[1])
	if err != nil {
		fmt.Printf("url parsing error: %v", err)
		os.Exit(1)
	}
	srv := server.NewServer(u.Host)
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}
	srv.WSHandleFunc(u.Path, func(conn *websocket.Conn) {
		in := msg{}
		id := conn.RemoteAddr().String()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseServiceRestart) {
					fmt.Printf("echoHandler for %s: websocket reading error: %v\n", id, err)
				} else {
					fmt.Printf("echoHandler for %s: %s\n", id, err)
				}
				return
			}
			fmt.Printf("echoHandler for %s: handle message: %s\n", id, message)
			if err := json.Unmarshal(message, &in); err != nil {
				sendMsg(msg{"error", fmt.Sprintf("message parsing error: %v", err), nil}, conn)
				continue
			}
			switch in.Type {
			case "echo":
				sendMsg(msg{in.Type, in.Payload, nil}, conn)
			case "broadcast":
				count := 0
				out, _ := json.Marshal(msg{in.Type, in.Payload, nil})
				srv.ForEachConnection(func(c *websocket.Conn) bool {
					c.WriteMessage(websocket.TextMessage, out)
					count++
					return true
				})
				sendMsg(msg{"broadcastResult", in.Payload, &count}, conn)
			default:
				sendMsg(msg{"error", "unknown type", nil}, conn)
			}
		}
	})
	go func() {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, os.Interrupt, os.Kill)
		fmt.Printf("\n%s signal received, exiting...\n", <-sig)
		srv.Close()
	}()
	fmt.Printf("starting echo server on %s...\n", u.Host)
	err = srv.ListenAndServe()
	if err.Error() != "http: Server closed" {
		fmt.Println(err)
		os.Exit(1)
	}
}