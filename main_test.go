package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWSinitMsg(t *testing.T) {
	s := newMockServer(0)
	defer s.Close()
	message := "test message"
	options.initMsg = message
	options.authHeader = "Bearer the_token_is_here"
	defer func() {
		options.initMsg = ""
		options.authHeader = ""
	}()
	cmd := &cobra.Command{}
	time.AfterFunc(100*time.Millisecond, func() { session.cancel() })
	root(cmd, []string{mockURL})
	require.Eventually(t, func() bool { return len(s.Received) > 0 }, 20*time.Millisecond, 2*time.Millisecond)
	require.Equal(t, message, <-s.Received)
}

func TestWSconnectFail(t *testing.T) {
	envName := fmt.Sprintf("BE_%s", t.Name())
	if os.Getenv(envName) == "1" {
		root(&cobra.Command{}, []string{"wss://127.0.0.1:8080"})
		time.Sleep(300 * time.Millisecond)
		session.cancel()
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
