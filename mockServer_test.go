package main

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestMockServer(t *testing.T) {
	s := newMockServer(0)
	defer s.Close()
	conn := newMockConn()
	defer TryCloseNormally(conn, "test finished")
	sent := "test"
	// test client -> server message
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(sent)))
	require.Eventually(t, func() bool { return len(s.Received) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, sent, <-s.Received)
	// Repeat
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(sent)))
	require.Eventually(t, func() bool { return len(s.Received) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, sent, <-s.Received)
	// test server -> client message
	s.ToSend <- sent
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Millisecond)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, sent, string(data))
}

func TestMockServerPing(t *testing.T) {
	s := newMockServer(5 * time.Millisecond)
	defer s.Close()
	conn := newMockConn()
	defer TryCloseNormally(conn, "test finished")
	var pingCount int64
	conn.SetPingHandler(func(appData string) error {
		atomic.AddInt64(&pingCount, 1)
		return conn.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(10*time.Millisecond))
	})
	conn.SetReadDeadline(time.Now().Add(30 * time.Millisecond))
	conn.ReadMessage()
	require.Greater(t, atomic.LoadInt64(&pingCount), int64(3))
	s.Close()
}

func TestMockServerDoubleStart(t *testing.T) {
	s := newMockServer(0)
	defer s.Close()
	require.Panics(t, func() { newMockServer(0) })
}

func TestMockConnPanic(t *testing.T) {
	require.Panics(t, func() { newMockConn() })
}
