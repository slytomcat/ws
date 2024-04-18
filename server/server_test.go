package server

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestConnection(t *testing.T) {
	fullURL := "ws://localhost:8080/ws"
	server := NewServer("localhost:8080")
	server.WSHandleFunc("/ws", EchoHandler)
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()
	require.Never(t, func() bool {
		select {
		case <-errCh:
			return true
		default:
			return false
		}
	}, 100*time.Millisecond, 10*time.Millisecond)
	defer server.Close()
	dialer := websocket.Dialer{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	conn, _, err := dialer.DialContext(ctx, fullURL, nil)
	require.NoError(t, err)
	sentMsg := []byte("ping")
	err = conn.WriteMessage(websocket.TextMessage, sentMsg)
	require.NoError(t, err)
	_, msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, sentMsg, msg)
	t.Log("echo message received")
	TryCloseNormally(conn, "test finish")
	server.Close()
	err = <-errCh
	require.EqualError(t, err, "http: Server closed")
	TryCloseNormally(conn, "test finish")
}

func TestHandshakeServerError(t *testing.T) {
	fullURL := "ws://:8080"
	s := NewServer(":8080")
	require.NotNil(t, s)
	s.WSHandleFunc("/", EchoHandler)
	go func() { s.ListenAndServe() }()
	time.Sleep(50 * time.Millisecond)
	defer s.Close()
	orig := s.Upgrader
	upgrader.CheckOrigin = func(r *http.Request) bool { return false }
	defer func() {
		s.Upgrader = orig
	}()
	dialer := websocket.Dialer{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err := dialer.DialContext(ctx, fullURL, nil)
	require.EqualError(t, err, "websocket: bad handshake")
}

func TestHandshakeClientError(t *testing.T) {
	fullURL := "ws://:8080"
	s := NewServer(":8080")
	require.NotNil(t, s)
	s.WSHandleFunc("/", EchoHandler)
	go func() { s.ListenAndServe() }()
	time.Sleep(50 * time.Millisecond)
	defer s.Close()
	orig := s.Upgrader
	upgrader.CheckOrigin = func(r *http.Request) bool { return false }
	defer func() {
		s.Upgrader = orig
	}()
	dialer := websocket.Dialer{HandshakeTimeout: time.Nanosecond}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, _, err := dialer.DialContext(ctx, fullURL, nil)
	require.EqualError(t, err, "dial tcp :8080: i/o timeout")

}

func TestRegularHandler(t *testing.T) {
	s := NewServer("localhost:8080")
	require.NotNil(t, s)
	s.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	go func() { s.ListenAndServe() }()
	time.Sleep(50 * time.Millisecond)
	defer s.Close()
	resp, err := http.DefaultClient.Get("http://localhost:8080/ok")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

}
