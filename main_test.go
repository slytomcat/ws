package main

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/slytomcat/ws/server"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const mockURL = "ws://localhost:8080"

type mockServer struct {
	Close    func() error
	Received chan string
	ToSend   chan string
}

func newMockServer() *mockServer {
	received := make(chan string, 10)
	toSend := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	u, _ := url.Parse(mockURL)
	s := server.NewServer(u.Host)
	s.WSHandleFunc("/", func(conn *websocket.Conn) {
		go func() {
			for {
				_, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				received <- string(msg)
			}
		}()
		select {
		case <-ctx.Done():
			TryCloseNormally(conn, "server going down")
			return
		case data := <-toSend:
			conn.WriteMessage(websocket.TextMessage, []byte(data))
		}
	})
	go s.ListenAndServe()
	time.Sleep(50 * time.Millisecond)
	return &mockServer{
		Close: func() error {
			cancel()
			s.Shutdown(ctx)
			return nil
		},
		Received: received,
		ToSend:   toSend,
	}
}

func newMockConn() *websocket.Conn {
	dial := websocket.Dialer{}
	conn, _, err := dial.Dial(mockURL, nil)
	if err != nil {
		panic(err)
	}
	return conn
}

func TestMockServer(t *testing.T) {
	s := newMockServer()
	defer s.Close()
	conn := newMockConn()
	defer TryCloseNormally(conn, "test finished")
	sent := "test"
	// test client -> server message
	require.NoError(t, conn.WriteMessage(websocket.TextMessage, []byte(sent)))
	var received string
	require.Eventually(t, func() bool {
		select {
		case received = <-s.Received:
			return true
		default:
			return false
		}
	}, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, sent, received)
	// test server -> client message
	s.ToSend <- sent
	require.NoError(t, conn.SetReadDeadline(time.Now().Add(20*time.Millisecond)))
	_, data, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, sent, string(data))
}

func TestWSinitMsg(t *testing.T) {
	s := newMockServer()
	defer s.Close()
	message := "test message"
	options.initMsg = message
	options.authHeader += "Bearer ajshdkjhipuqofqldbclqwehqlieh;#kqnwe;ldk"
	defer func() { options.initMsg = "" }()
	cmd := &cobra.Command{}
	time.AfterFunc(100*time.Millisecond, func() { session.cancel() })
	root(cmd, []string{mockURL})
}

func TestWSconnectFail(t *testing.T) {
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		root(&cobra.Command{}, []string{"wss://127.0.0.1:8080"})
		return
	}
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	r, err := cmd.StderrPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	out, _ := io.ReadAll(r)
	err = cmd.Wait()
	require.EqualError(t, err, "exit status 1")
	require.Equal(t, "dial tcp 127.0.0.1:8080: connect: connection refused\n", string(out))
}

func TestWSincorrectUrl(t *testing.T) {
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		root(&cobra.Command{}, []string{"\n"})
		return
	}
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	r, err := cmd.StderrPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	out, _ := io.ReadAll(r)
	err = cmd.Wait()
	require.EqualError(t, err, "exit status 1")
	require.Equal(t, "parse \"\\n\": net/url: invalid control character in URL\n", string(out))
}

func TestWSnoArg(t *testing.T) {
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		main()
		return
	}
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	outR, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	time.Sleep(200 * time.Millisecond)
	stdOut, _ := io.ReadAll(outR)
	err = cmd.Wait()
	require.EqualError(t, err, "exit status 1")
	assert.Equal(t, "ws is a websocket client v.local build\n\nUsage:\n  ws URL [flags]\n\nFlags:\n  -a, --auth string          auth header value, like 'Bearer $TOKEN'\n  -b, --bin2text             print binary message as text\n  -c, --compression          enable compression\n  -f, --filter string        only messages that match regexp will be printed\n  -h, --help                 help for ws\n  -m, --init string          connection init message\n  -k, --insecure             skip ssl certificate check\n  -i, --interval duration    send ping each interval (ex: 20s)\n  -o, --origin string        websocket origin (default value is formed from URL)\n  -p, --pingPong             print out ping/pong messages\n  -s, --subprotocal string   sec-websocket-protocal field\n  -t, --timestamp            print timestamps for sent and received messages\n  -v, --version              print version\n", string(stdOut))
}

func TestWSversion(t *testing.T) {
	options.printVersion = true
	defer func() { options.printVersion = false }()
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		root(&cobra.Command{}, []string{})
		return
	}
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	outR, err := cmd.StdoutPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	time.Sleep(200 * time.Millisecond)
	stdOut, _ := io.ReadAll(outR)
	err = cmd.Wait()
	require.NoError(t, err)
	assert.Equal(t, "ws v.local build\n", string(stdOut))
}

func TestWSwrongFilter(t *testing.T) {
	filter = "}])^$jkh"
	defer func() { filter = "" }()
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		root(&cobra.Command{}, []string{mockURL})
		return
	}
	args := []string{"-test.run=" + t.Name()}
	for _, v := range os.Args {
		if strings.Contains(v, "cover") {
			args = append(args, v)
		}
	}
	cmd := exec.Command(os.Args[0], args...)
	errR, err := cmd.StderrPipe()
	require.NoError(t, err)
	cmd.Env = append(os.Environ(), envName+"=1")
	require.NoError(t, cmd.Start())
	time.Sleep(200 * time.Millisecond)
	stdErr, _ := io.ReadAll(errR)
	err = cmd.Wait()
	require.EqualError(t, err, "exit status 1")
	assert.Equal(t, "compiling regexp '}])^$jkh' error: error parsing regexp: unexpected ): `}])^$jkh`", string(stdErr))
}
