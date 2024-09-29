package main

import (
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
	outR, outW, _ := os.Pipe()
	rl, err := readline.NewEx(&readline.Config{Prompt: "> ", Stdout: outW})
	require.NoError(t, err)
	s := Session{
		ws:      conn,
		rl:      rl,
		cancel:  func() {},
		errors:  []error{},
		errLock: sync.Mutex{},
	}
	sent := "test message"
	typed := "typed"
	binary := "binary"
	unknown := "unknown"
	toBeFiltered := "must be filtered"
	// test sendMsg
	options.timestamp = true
	defer func() { options.timestamp = false }()
	err = s.sendMsg(sent)
	require.NoError(t, err)
	require.Eventually(t, func() bool { return len(srv.Received) > 0 }, 100*time.Millisecond, 3*time.Millisecond)
	require.Equal(t, sent, <-srv.Received)
	// test typing
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
	srv.ToSend <- binary
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	// filtered
	options.filter = regexp.MustCompile("^.*not filtered.*$")
	defer func() { options.filter = nil }()
	require.False(t, options.filter.MatchString(toBeFiltered))
	srv.ToSend <- toBeFiltered
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	// unknown mode
	atomic.StoreInt64(&srv.Mode, 0)
	srv.ToSend <- unknown
	require.Eventually(t, func() bool { return len(srv.ToSend) == 0 }, 20*time.Millisecond, 2*time.Millisecond)
	time.Sleep(20 * time.Millisecond)
	outW.Close()
	output, err := io.ReadAll(outR)
	out := string(output)
	require.NoError(t, err)
	require.Contains(t, out, " > "+sent)
	require.Contains(t, out, " > "+typed)
	require.Contains(t, out, " < "+sent)
	require.Contains(t, out, " < \n00000000  74 65 73 74 20 6d 65 73  73 61 67 65              |"+sent+"|")
	require.Contains(t, out, " < "+binary)
	require.NotContains(t, out, toBeFiltered)
	require.NotContains(t, out, unknown)
	// t.Log(out)
}

func TestPingPong(t *testing.T) {
	srv := newMockServer(5 * time.Millisecond)
	options.pingPong = true
	options.pingInterval = 5 * time.Millisecond
	outR, outW, _ := os.Pipe()
	inR, inW, _ := os.Pipe()
	defer func() {
		inW.Close()
		outW.Close()
		options.pingPong = false
		options.pingInterval = 0
		srv.Close()
	}()
	errs := make(chan []error, 1)
	// substitute FuncMakeRaw and FuncExitRaw to empty func to use open pipe as Stdin
	// switching to raw file descriptor 0 will cause immediately closure of rl due to EOF
	success := func() error { return nil }
	rl, err := readline.NewEx(&readline.Config{Prompt: "> ", Stdin: inR, Stdout: outW, FuncMakeRaw: success, FuncExitRaw: success})
	require.NoError(t, err)
	s := &Session{rl: rl}
	go func() {
		errs <- s.connect(mockURL)
	}()
	time.Sleep(20 * time.Millisecond)
	require.Eventually(t, func() bool { return s.cancel != nil }, 100*time.Millisecond, 2*time.Millisecond)
	s.cancel()
	inW.Close()
	outW.Close()
	require.Eventually(t, func() bool { return len(errs) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	output, err := io.ReadAll(outR)
	out := string(output)
	// t.Log(out)
	require.NoError(t, err)
	require.Contains(t, out, "> ping")
	require.Contains(t, out, "< pong")
	require.Contains(t, out, "< ping")
	require.Contains(t, out, "> pong")
}

func TestInitMsg(t *testing.T) {
	m := newMockServer(0)
	defer m.Close()
	message := "test message"
	options.initMsg = message
	defer func() {
		options.initMsg = ""
	}()
	rl, err := readline.New(" >")
	require.NoError(t, err)
	s := &Session{rl: rl}
	time.AfterFunc(500*time.Millisecond, func() { s.cancel() })
	errs := s.connect(mockURL)
	require.Empty(t, errs)
	require.Eventually(t, func() bool { return len(m.Received) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, message, <-m.Received)
}
