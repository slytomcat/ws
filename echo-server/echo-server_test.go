package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEchoServer(t *testing.T) {
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		go DoMain([]string{""})
		time.Sleep(30 * time.Millisecond)
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(defaultUrl, nil)
		require.NoError(t, err)
		testCases := []struct {
			name      string
			toSend    string
			toReceive []string
		}{
			{
				name:   "echo success",
				toSend: `{"type":"echo", "payload":"Hello world!"}`,
				toReceive: []string{
					`{"type":"echo","payload":"Hello world!"}`,
				},
			},
			{
				name:   "broadcast success",
				toSend: `{"type":"broadcast", "payload":"Hello world!"}`,
				toReceive: []string{
					`{"type":"broadcast","payload":"Hello world!"}`,
					`{"type":"broadcastResult","payload":"Hello world!","listenerCount":1}`,
				},
			},
			{
				name:   "wrong message type",
				toSend: `{"type":"wrong", "payload":"Hello world!"}`,
				toReceive: []string{
					`{"type":"error","payload":"unknown type"}`,
				},
			},
			{
				name:   "incorrect json",
				toSend: `}`,
				toReceive: []string{
					`{"type":"error","payload":"message parsing error: invalid character '}' looking for beginning of value"}`,
				},
			},
		}
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := conn.WriteMessage(websocket.TextMessage, []byte(tc.toSend))
				assert.NoError(t, err)
				for _, r := range tc.toReceive {
					mType, received, err := conn.ReadMessage()
					assert.NoError(t, err)
					assert.Equal(t, websocket.TextMessage, mType)
					assert.Equal(t, string(r), string(received))
				}
			})
		}
		// server.TryCloseNormally(conn, "tests finish")
		//require.NoError(t, srv.Close())
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		return
	}
	// collect child coverage artifacts in the parent files
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	r, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	out, _ := io.ReadAll(r)
	output := string(out)
	err = cmd.Wait()
	require.NoError(t, err)
	require.Contains(t, output, "starting echo server on localhost:8080...")
}

func TestDoMain(t *testing.T) {
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		DoMain([]string{"", ":"})
		return
	}
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	r, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	out, _ := io.ReadAll(r)
	err = cmd.Wait()
	require.EqualError(t, err, "exit status 1")
	require.Equal(t, "url parsing error: parse \":\": missing protocol scheme", string(out))
}

func TestEchoServer2(t *testing.T) {
	url := "ws://localhost:8080"
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		go DoMain([]string{"", url})
		time.Sleep(50 * time.Millisecond)
		dialer := websocket.Dialer{}
		conn, _, err := dialer.Dial(url, nil)
		require.NoError(t, err)
		conn.Close()
		time.Sleep(5 * time.Millisecond)
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		return
	}
	// collect child coverage artifacts in the parent files
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	r, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	out, _ := io.ReadAll(r)
	output := string(out)
	err = cmd.Wait()
	require.NoError(t, err)
	require.Contains(t, output, "starting echo server on localhost:8080...")
}
