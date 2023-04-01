package main

import (
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"time"

	"github.com/chzyer/readline"
	"github.com/fatih/color"
	"github.com/gorilla/websocket"
)

type session struct {
	ws      *websocket.Conn
	rl      *readline.Instance
	errChan chan error
}

var rxSprintf = color.New(color.FgGreen).SprintfFunc()

const tcFormat = "20060102T150405.999"

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

	rl, err := readline.NewEx(rlConf)
	if err != nil {
		return err
	}
	defer rl.Close()

	sess := &session{
		ws:      ws,
		rl:      rl,
		errChan: make(chan error),
	}

	go sess.readConsole()
	go sess.readWebsocket()

	return <-sess.errChan
}

func (s *session) readConsole() {
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
		var prefix string
		if options.timestamp {
			prefix = time.Now().UTC().Format(tcFormat)
		}
		fmt.Fprint(s.rl.Stdout(), rxSprintf("%s > %s\n", prefix, line))
	}
}

func bytesToFormattedHex(bytes []byte) string {
	text := hex.EncodeToString(bytes)
	return regexp.MustCompile("(..)").ReplaceAllString(text, "$1 ")
}

func (s *session) readWebsocket() {
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
				text = bytesToFormattedHex(buf)
			}
		default:
			s.errChan <- fmt.Errorf("unknown websocket frame type: %d", msgType)
			return
		}
		var prefix string
		if options.timestamp {
			prefix = time.Now().UTC().Format(tcFormat)
		}
		fmt.Fprint(s.rl.Stdout(), rxSprintf("%s < %s\n", prefix, text))
	}
}
