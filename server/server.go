package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Server is a websocket/http server. It is wrapper for standard http.Server with additional functionality for websocket request handling.
type Server struct {
	http.Server
	connections sync.Map
	handle      func(*websocket.Conn)
	mux         *http.ServeMux
	Upgrader    websocket.Upgrader
}

// NewServer creates new websocket/http server. It is configured to start on provided addr and creates the standard serve mux and sets it as server handler.
// Use WSHandleFunc and HandleFunc for setting handlers on desired paths and start Server via ListenAndServe/ListenAndServeTLS.
// ForEachConnection allows to iterate over currently active WS connections.
// For example you can send some broadcast message via s.ForEachConnection(func(c *websocket.Conn){c.WriteMessage(websocket.TextMessage, message)})
func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	return &Server{
		mux: mux,
		Upgrader: websocket.Upgrader{
			HandshakeTimeout: time.Second,
			Subprotocols:     []string{},
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Server: http.Server{
			Addr:    addr,
			Handler: mux,
		},
	}
}

// WSHandleFunc setups new WS handler for path
func (s *Server) WSHandleFunc(path string, handler func(*websocket.Conn)) {
	s.mux.HandleFunc(path, s.serve(handler))
}

// HandleFunc setups new regular http handler for path
func (s *Server) HandleFunc(path string, handler func(w http.ResponseWriter, r *http.Request)) {
	s.mux.HandleFunc(path, handler)
}

// Close correctly closes all active ws connections and shutdown the server
func (s *Server) Close() error {
	s.ForEachConnection(func(c *websocket.Conn) {
		if err := TryCloseNormally(c, "server going down"); err != nil {
			fmt.Printf("server: closing connection from %s error: %v\n", c.RemoteAddr(), err)
		}
		c.Close()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	return s.Server.Shutdown(ctx)
}

// TryCloseNormally tries to close websocket connection normally i.e. according to RFC
// NOTE It doesn't close underlying connection as socket reader have to read and handle close response.
func TryCloseNormally(conn *websocket.Conn, message string) error {
	closeMessage := websocket.FormatCloseMessage(websocket.CloseNormalClosure, message)
	if err := conn.WriteControl(websocket.CloseMessage, closeMessage, time.Now().Add(time.Second)); err != nil {
		if strings.Contains(err.Error(), "close sent") {
			return nil
		}
		return err
	}
	return nil
}

func (s *Server) serve(handler func(*websocket.Conn)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		connection, err := s.Upgrader.Upgrade(w, r, nil)
		if err != nil {
			fmt.Printf("server: connection upgrade error: %v\n", err)
			return
		}
		addr := connection.RemoteAddr().String()
		fmt.Printf("server: new WS connection from %s\n", addr)
		s.connections.Store(connection, nil)
		handler(connection)
		s.connections.Delete(connection)
		fmt.Printf("server: WS connection from %s closed\n", addr)
		connection.Close()
	}
}

// ForEachConnection allow to iterate over all active connections
func (s *Server) ForEachConnection(f func(*websocket.Conn)) {
	s.connections.Range(func(conn, _ any) bool {
		f(conn.(*websocket.Conn))
		return true
	})
}
