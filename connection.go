package main

import (
	"context"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

// Session is the WS session
type Session struct {
	ws     *websocket.Conn
	rl     *readline.Instance
	cancel func()
	ctx    context.Context
	err    error
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

func connect(url string, rlConf *readline.Config) error {
	headers := make(http.Header)
	headers.Add("Origin", options.origin)

	dialer := websocket.Dialer{
		Proxy: http.ProxyFromEnvironment,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: options.insecure,
		},
		Subprotocols: []string{options.subProtocals},
	}
	ws, _, err := dialer.Dial(url, headers)
	if err != nil {
		return err
	}
	defer ws.Close()

	rl, err := readline.NewEx(rlConf)
	if err != nil {
		return err
	}
	defer rl.Close()

	session := &Session{
		ws: ws,
		rl: rl,
	}
	session.ctx, session.cancel = context.WithCancel(context.Background())
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
	}
	if options.pingInterval != 0 {
		ws.SetPongHandler(func(appData string) error {
			fmt.Fprint(rl.Stdout(), ctSprintf("%s < pong\n", getPrefix()))
			return nil
		})
		ticker := time.NewTicker(options.pingInterval)
		defer ticker.Stop()
		go func() {
			for {
				select {
				case <-session.ctx.Done():
					return
				case <-ticker.C:
					err := ws.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(time.Second))
					if err != nil {
						fmt.Printf("ping sending error: `%v`", err)
						session.err = err
						return
					}
					fmt.Fprint(rl.Stdout(), ctSprintf("%s > ping\n", getPrefix()))
				}
			}
		}()
	}
	go session.readConsole()
	go session.readWebsocket()
	<-session.ctx.Done()
	return session.err
}

func (s *Session) readConsole() {
	defer s.cancel()
	for {
		line, err := s.rl.Readline()
		if err != nil {
			s.err = err
			return
		}

		err = s.ws.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			s.err = fmt.Errorf("writing error: `%w`", err)
			return
		}
		if options.timestamp { // repeat sent massage only if timestamp is required
			fmt.Fprint(s.rl.Stdout(), txSprintf("%s> %s\n", getPrefix(), line))
		}
	}
}

func (s *Session) readWebsocket() {
	defer s.cancel()
	for {
		msgType, buf, err := s.ws.ReadMessage()
		if err != nil {
			s.err = fmt.Errorf("reading error: `%w`", err)
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
			s.err = fmt.Errorf("unknown websocket frame type: %d", msgType)
			return
		}
		fmt.Fprint(s.rl.Stdout(), rxSprintf("%s< %s\n", getPrefix(), text))
	}
}
