package main

import (
	"context"
	"net/url"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/slytomcat/ws/server"
)

const mockURL = "ws://localhost:8080/ws"

type mockServer struct {
	Close    func() error
	Received chan string
	ToSend   chan string
	Mode     int64
}

func newMockServer(interval time.Duration) *mockServer {
	received := make(chan string, 10)
	toSend := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	u, _ := url.Parse(mockURL)
	s := server.NewServer(u.Host)
	m := &mockServer{
		Close: func() error {
			cancel()
			s.Shutdown(ctx)
			return nil
		},
		Received: received,
		ToSend:   toSend,
		Mode:     websocket.TextMessage,
	}
	s.WSHandleFunc(u.Path, func(conn *websocket.Conn) {
		if interval != 0 {
			go func() {
				ticker := time.NewTicker(interval)
				for {
					select {
					case <-ticker.C:
						if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(20*time.Millisecond)); err != nil {
							return
						}
					case <-ctx.Done():
						return
					}
				}
			}()
		}
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				received <- string(msg)
			}
		}()
		for {
			select {
			case <-ctx.Done():
				TryCloseNormally(conn, "server going down")
				return
			case data := <-toSend:
				conn.WriteMessage(int(atomic.LoadInt64(&m.Mode)), []byte(data))
			}
		}
	})
	go s.ListenAndServe()
	time.Sleep(50 * time.Millisecond)
	return m
}

func newMockConn() *websocket.Conn {
	dial := websocket.Dialer{}
	conn, _, err := dial.Dial(mockURL, nil)
	if err != nil {
		panic(err)
	}
	return conn
}
