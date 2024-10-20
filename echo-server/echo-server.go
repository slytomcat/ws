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

type echoServer struct {
	*server.Server
}

func New(addr string) (*echoServer, error) {
	u, err := url.Parse(addr)
	if err != nil {
		return nil, fmt.Errorf("url parsing error: %v", err)
	}
	srv := server.NewServer(u.Host)
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
	return &echoServer{srv}, nil
}

func sendMsg(m msg, c *websocket.Conn) {
	response, _ := json.Marshal(m)
	c.WriteMessage(websocket.TextMessage, response)
}

func main() {
	if err := DoMain(os.Args); err != nil {
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}
}

// DoMain is main with os.Args as parameter
func DoMain(args []string) error {
	addr := defaultUrl
	if len(args) > 1 {
		addr = args[1]
	}
	srv, err := New(addr)
	if err != nil {
		return err
	}
	go func() {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		fmt.Printf("\n%s signal received, exiting...\n", <-sig)
		srv.Close()
	}()
	fmt.Printf("starting echo server on %s...\n", addr)
	err = srv.ListenAndServe()
	if !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
