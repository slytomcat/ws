package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
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
	ctx     context.Context
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

func connect(url string, rlConf *readline.Config) []error {
	headers := make(http.Header)
	headers.Add("Origin", options.origin)

	dialer := websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: options.insecure,
		},
		Subprotocols: []string{options.subProtocals},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ws, _, err := dialer.DialContext(ctx, url, headers)
	if err != nil {
		return []error{err}
	}
	defer ws.Close()

	rl, err := readline.NewEx(rlConf)
	if err != nil {
		return []error{err}
	}
	defer rl.Close()

	session := &Session{
		ws:     ws,
		rl:     rl,
		ctx:    ctx,
		cancel: cancel,
		errors: []error{},
	}
	if options.pingPong {
		ws.SetPingHandler(func(appData string) error {
			fmt.Fprint(rl.Stdout(), ctSprintf("%s < ping: %s\n", getPrefix(), appData))
			err := ws.WriteControl(websocket.PongMessage, []byte(appData), time.Now().Add(time.Second))
			if err != nil {
				return err
			}
			fmt.Fprint(rl.Stdout(), ctSprintf("%s > pong: %s\n", getPrefix(), appData))
			return nil
		})
		ws.SetPongHandler(func(appData string) error {
			fmt.Fprint(rl.Stdout(), ctSprintf("%s < pong\n", getPrefix()))
			return nil
		})
	}
	if options.pingInterval != 0 {
		go session.pingHandler()
	}
	if options.initMsg != "" {
		if err = session.sendMsg(options.initMsg); err != nil {
			return []error{err}
		}
	}
	go session.readConsole()
	go session.readWebsocket()
	<-session.ctx.Done()
	return session.getErr()
}

func (s *Session) pingHandler() {
	ticker := time.NewTicker(options.pingInterval)
	defer ticker.Stop()
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			err := s.ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
			if err != nil {
				fmt.Printf("ping sending error: `%v`", err)
				s.setErr(err)
				return
			}
			if options.pingPong {
				fmt.Fprint(s.rl.Stdout(), ctSprintf("%s > ping\n", getPrefix()))
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
			s.setErr(err)
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
			s.setErr(fmt.Errorf("reading error: `%w`", err))
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
		fmt.Fprint(s.rl.Stdout(), rxSprintf("%s< %s\n", getPrefix(), text))
	}
}
