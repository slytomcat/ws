package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

// Session is the WS session
type Session struct {
	ws      *websocket.Conn
	rl      *readline.Instance
	cancel  func()
	errors  []error
	errLock sync.Mutex
}

func (s *Session) setErr(err error) {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	s.errors = append(s.errors, err)
}

func (s *Session) getErr() []error {
	s.errLock.Lock()
	defer s.errLock.Unlock()
	res := make([]error, len(s.errors))
	copy(res, s.errors)
	return res
}

var (
	rxSprintf = color.New(color.FgGreen).SprintfFunc()
	txSprintf = color.New(color.FgBlue).SprintfFunc()
	ctSprintf = color.New(color.FgRed).SprintfFunc()
)

const tsFormat = "20060102T150405.999"

func getPrefix() string {
	if options.timestamp {
		prefix := time.Now().UTC().Format(tsFormat)
		if len(prefix) < 19 {
			prefix += strings.Repeat("0", 19-len(prefix))
		}
		return prefix + " "
	}
	return ""
}

func (s *Session) connect(url string) []error {
	headers := make(http.Header)
	headers.Add("Origin", options.origin)
	if options.authHeader != "" {
		headers.Add("Authorization", options.authHeader)
	}
	dialer := websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: options.insecure,
		},
		EnableCompression: options.compression,
		Subprotocols:      []string{options.subProtocals},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ws, _, err := dialer.DialContext(ctx, url, headers)
	if err != nil {
		return []error{err}
	}
	defer func() {
		s.rl.Close()
		TryCloseNormally(ws, "client disconnection")
		ws.Close()
	}()
	s.ws = ws
	s.cancel =  cancel
	s.errors =  []error{}
	if options.pingPong {
		ws.SetPingHandler(func(appData string) error {
			fmt.Fprint(s.rl.Stdout(), ctSprintf("%s < ping: %s\n", getPrefix(), appData))
			err := ws.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
			if err != nil {
				return err
			}
			fmt.Fprint(s.rl.Stdout(), ctSprintf("%s > pong: %s\n", getPrefix(), appData))
			return nil
		})
		ws.SetPongHandler(func(appData string) error {
			fmt.Fprint(s.rl.Stdout(), ctSprintf("%s < pong: %s\n", getPrefix(), appData))
			return nil
		})
	}
	if options.pingInterval != 0 {
		go s.pingHandler(ctx)
	}
	if options.initMsg != "" {
		if err = s.sendMsg(options.initMsg); err != nil {
			return []error{err}
		}
	}
	go func() {
		sig := make(chan os.Signal, 2)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		fmt.Printf("\n%s signal received, exiting...\n", <-sig)
		s.rl.Close()
		s.cancel()
	}()

	go s.readConsole()
	go s.readWebsocket()
	<-ctx.Done()
	return s.getErr()
}

func (s *Session) pingHandler(ctx context.Context) {
	ticker := time.NewTicker(options.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := s.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				fmt.Printf("ping sending error: `%v`", err)
				s.setErr(err)
				return
			}
			if options.pingPong {
				fmt.Fprint(s.rl.Stdout(), ctSprintf("%s > ping: \n", getPrefix()))
			}
		}
	}
}

func (s *Session) sendMsg(msg string) error {
	err := s.ws.WriteMessage(websocket.TextMessage, []byte(msg))
	if err != nil {
		return fmt.Errorf("writing error: `%w`", err)
	}
	if options.timestamp { // repeat sent massage only if timestamp is required
		fmt.Fprint(s.rl.Stdout(), txSprintf("%s> %s\n", getPrefix(), msg))
	}
	return nil
}

func (s *Session) readConsole() {
	defer s.cancel()
	for {
		line, err := s.rl.Readline()
		if err != nil {
			if !(errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF)) {
				s.setErr(err)
			}
			return
		}
		if err = s.sendMsg(line); err != nil {
			s.setErr(err)
			return
		}
	}
}

func (s *Session) readWebsocket() {
	defer s.cancel()
	defer s.rl.Close()
	for {
		msgType, buf, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure, websocket.CloseServiceRestart) {
				s.setErr(fmt.Errorf("reading error: `%v`", err))
			} else {
				s.setErr(fmt.Errorf("connection closed: %s", err))
			}
			return
		}
		var text string
		switch msgType {
		case websocket.TextMessage:
			text = string(buf)
		case websocket.BinaryMessage:
			if options.binAsText {
				text = string(buf)
			} else {
				text = "\n" + hex.Dump(buf)
			}
		default:
			s.setErr(fmt.Errorf("unknown websocket frame type: %d", msgType))
			return
		}
		if options.filter != nil && !options.filter.MatchString(text) {
			continue
		}
		fmt.Fprint(s.rl.Stdout(), rxSprintf("%s< %s\n", getPrefix(), text))
	}
}

// TryCloseNormally tries to close websocket connection normally i.e. according to RFC
// NOTE It doesn't close underlying connection as socket reader have to read and handle close response.
func TryCloseNormally(conn *websocket.Conn, message string) error {
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, message)
	if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
		if !strings.Contains(err.Error(), "close sent") {
			return err
		}
	}
	return nil
}
