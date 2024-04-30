package main

import (
	"context"
	"io"
	"os"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chzyer/readline"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
)

func TestGetPrefix(t *testing.T) {
	options.timestamp = true
	defer func() { options.timestamp = false }()
	prefix := getPrefix()
	require.Contains(t, prefix, " ")
	require.Len(t, prefix, 20)
	options.timestamp = false
	prefix = getPrefix()
	require.Empty(t, prefix)
}

func TestSession(t *testing.T) {
	srv := newMockServer(0)
	defer srv.Close()
	conn := newMockConn()
	defer TryCloseNormally(conn, "test finished")
	ctx, cancel := context.WithCancel(context.Background())
	outR, outW, _ := os.Pipe()
	rl, err := readline.NewEx(&readline.Config{Prompt: "> ", Stdout: outW})
	require.NoError(t, err)
	s := Session{
		ws:      conn,
		rl:      rl,
		cancel:  cancel,
		ctx:     ctx,
		errors:  []error{},
		errLock: sync.Mutex{},
	}
	sent := "test message"
	// test sendMsg
	options.timestamp = true
	defer func() { options.timestamp = false }()
	err = s.sendMsg(sent)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(srv.Received) > 0 }, 30*time.Millisecond, 3*time.Millisecond)
	require.Equal(t, sent, <-srv.Received)
	// test typing
	typed := "typed"
	_, err = rl.WriteStdin([]byte(typed + "\n"))
	require.NoError(t, err)
	go func() {
		s.readConsole()
	}()
	require.Eventually(t, func() bool { return len(srv.Received) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, typed, <-srv.Received)
	// test readWebsocket
	go func() {
		s.readWebsocket()
	}()
	// text message
	srv.ToSend <- sent
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	// binary message
	atomic.StoreInt64(&srv.Mode, websocket.BinaryMessage)
	srv.ToSend <- sent
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	// binary as text
	options.binAsText = true
	defer func() { options.binAsText = false }()
	srv.ToSend <- "binary"
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	// filtered
	toBeFiltered := "must be filtered"
	options.filter = regexp.MustCompile("^.*not filtered.*$")
	defer func() { options.filter = nil }()
	require.False(t, options.filter.MatchString(toBeFiltered))
	srv.ToSend <- toBeFiltered
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	// unknown mode
	atomic.StoreInt64(&srv.Mode, 0)
	srv.ToSend <- "unknown"
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	cancel()
	outW.Close()
	output, err := io.ReadAll(outR)
	out := string(output)
	require.NoError(t, err)
	require.Contains(t, out, " > test message")
	require.Contains(t, out, " > typed")
	require.Contains(t, out, " < test message")
	require.Contains(t, out, " < \n00000000  74 65 73 74 20 6d 65 73  73 61 67 65              |test message|")
	require.Contains(t, out, " < binary")
	require.NotContains(t, out, toBeFiltered)
	require.NotContains(t, out, "unknown")
	// t.Log(out)
}

func TestPingPong(t *testing.T) {
	srv := newMockServer(2 * time.Millisecond)
	defer srv.Close()
	options.pingPong = true
	options.pingInterval = 2 * time.Millisecond
	outR, outW, _ := os.Pipe()
	errs := make(chan []error, 1)
	rl, err := readline.NewEx(&readline.Config{Prompt: "> ", Stdout: outW, UniqueEditLine: true})
	require.NoError(t, err)
	// the only way I found to keep redline working for a while
	go func() {
		for i := 0; i < 400; i++ {
			_, err = rl.WriteStdin([]byte("typed"))
		}
	}()
	rl.Write([]byte("typed"))
	require.NoError(t, err)
	go func() {
		errs <- connect(mockURL, rl)
	}()
	time.Sleep(200 * time.Millisecond)
	session.cancel()
	require.Eventually(t, func() bool { return len(errs) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	outW.Close()
	output, err := io.ReadAll(outR)
	out := string(output)
	require.NoError(t, err)
	require.Contains(t, out, "> ping")
	require.Contains(t, out, "< pong")
	require.Contains(t, out, "< ping:")
	require.Contains(t, out, "> pong:")
}

func TestInitMsg(t *testing.T) {
	s := newMockServer(0)
	defer s.Close()
	message := "test message"
	options.initMsg = message
	defer func() {
		options.initMsg = ""
	}()
	rl, err := readline.New(" >")
	require.NoError(t, err)
	time.AfterFunc(500*time.Millisecond, func() { session.cancel() })
	errs := connect(mockURL, rl)
	require.Empty(t, errs)
	require.Eventually(t, func() bool { return len(s.Received) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, message, <-s.Received)
}
