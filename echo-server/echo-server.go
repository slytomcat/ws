package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/gorilla/websocket"
	"github.com/slytomcat/ws/server"
)

const defaultUrl = "ws://localhost:8080/ws"

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
	DoMain(os.Args)
}

var srv *server.Server

// DoMain is main with os.Args as parameter
func DoMain(args []string) {
	raw := defaultUrl
	if len(args) > 1 {
		raw = args[1]
	}
	u, err := url.Parse(raw)
	if err != nil {
		fmt.Printf("url parsing error: %v", err)
		os.Exit(1)
	}
	srv = server.NewServer(u.Host)
	if !strings.HasPrefix(u.Path, "/") {
		u.Path = "/" + u.Path
	}
	srv.WSHandleFunc(u.Path, func(conn *websocket.Conn) {
		in := msg{}
		addr := conn.RemoteAddr().String()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseServiceRestart) {
					fmt.Printf("echoHandler for %s: websocket reading error: %v\n", addr, err)
				} else {
					fmt.Printf("echoHandler for %s: %s\n", addr, err)
				}
				return
			}
			fmt.Printf("echoHandler for %s: handle message: %s\n", addr, message)
			if err := json.Unmarshal(message, &in); err != nil {
				sendMsg(msg{"error", fmt.Sprintf("message parsing error: %v", err), nil}, conn)
				continue
			}
			switch in.Type {
			case "echo":
				sendMsg(in, conn)
			case "broadcast":
				count := 0
				out, _ := json.Marshal(in)
				srv.ForEachConnection(func(c *websocket.Conn) {
					c.WriteMessage(websocket.TextMessage, out)
					count++
				})
				sendMsg(msg{"broadcastResult", in.Payload, &count}, conn)
			default:
				sendMsg(msg{"error", "unknown type: " + in.Type, nil}, conn)
			}
		}
	})
	go func() {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		fmt.Printf("\n%s signal received, exiting...\n", <-sig)
		srv.Close()
	}()
	fmt.Printf("starting echo server on %s...\n", u.Host)
	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		fmt.Println(err)
		os.Exit(1)
	}
}
