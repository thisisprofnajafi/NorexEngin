package handler

import (
	"golang.org/x/net/websocket"
	"net/http"
)

type Server struct {
	conns map[*websocket.Conn]bool
}

func NewServer() *Server {
	return &Server{conns: make(map[*websocket.Conn]bool)}
}
func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {

}
