package main

import (
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
	ws      *websocket.Conn
	rl      *readline.Instance
	errChan chan error
}

var (
	rxSprintf = color.New(color.FgGreen).SprintfFunc()
	txSprintf = color.New(color.FgBlue).SprintfFunc()
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
		ws:      ws,
		rl:      rl,
		errChan: make(chan error),
	}

	go session.readConsole()
	go session.readWebsocket()

	return <-session.errChan
}

func (s *Session) readConsole() {
	for {
		line, err := s.rl.Readline()
		if err != nil {
			s.errChan <- err
			return
		}

		err = s.ws.WriteMessage(websocket.TextMessage, []byte(line))
		if err != nil {
			s.errChan <- err
			return
		}
		if options.timestamp { // repeat sent massage only if timestamp is required
			fmt.Fprint(s.rl.Stdout(), txSprintf("%s> %s\n", getPrefix(), line))
		}
	}
}

func (s *Session) readWebsocket() {
	for {
		msgType, buf, err := s.ws.ReadMessage()
		if err != nil {
			s.errChan <- err
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
			s.errChan <- fmt.Errorf("unknown websocket frame type: %d", msgType)
			return
		}
		fmt.Fprint(s.rl.Stdout(), rxSprintf("%s< %s\n", getPrefix(), text))
	}
}
