package wsApi

import (
	"context"
	"fmt"
	"github.com/Bitspark/go-bitnode/bitnode"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"sync"
)

type Server struct {
	addr       string
	httpServer *http.Server
	wsUpgrader websocket.Upgrader
	errors     chan error
	wsFactory  *WSFactory
	mux        sync.Mutex
}

func NewServer(wsFactory *WSFactory, localAddr string) *Server {
	s := &Server{
		addr: localAddr,
		httpServer: &http.Server{
			Addr: localAddr,
		},
		wsUpgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		wsFactory: wsFactory,
	}
	handler := mux.NewRouter()
	handler.HandleFunc(wsPath, func(w http.ResponseWriter, r *http.Request) {
		if conn, err := s.acceptNode(w, r); err != nil {
			s.Log(bitnode.LogError, err.Error())
			return
		} else {
			go conn.Handle()
			go conn.Heartbeat(heartbeatInterval)
			s.Log(bitnode.LogInfo, fmt.Sprintf("Accepted node from %s", conn.ws.RemoteAddr().String()))
		}
	})
	s.httpServer.Handler = handler
	return s
}

func (s *Server) Address() string {
	return s.addr
}

func (s *Server) Listen() error {
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	if err := s.wsFactory.Shutdown(); err != nil {
		log.Println(err)
	}
	return s.httpServer.Shutdown(ctx)
}

func (s *Server) Clients() []*Client {
	clients := []*Client{}
	s.wsFactory.connsMux.Lock()
	for _, hc := range s.wsFactory.conns {
		hc.clientsMux.Lock()
		for _, c := range hc.clients {
			clients = append(clients, c)
		}
		hc.clientsMux.Unlock()
	}
	s.wsFactory.connsMux.Unlock()
	return clients
}

func (s *Server) Log(code int, msg string) {
	log.Printf("[SERVER-%d] %s", code, msg)
}

// Private

func (s *Server) acceptNode(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	conn, err := s.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	if conn, err := s.wsFactory.AcceptNode(conn); err != nil {
		return nil, err
	} else {
		return conn, nil
	}
}
