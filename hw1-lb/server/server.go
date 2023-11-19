package server

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sync"
)

// Server represents a single server
type Server struct {
	listener net.Listener
	address  string
	proto    string
	alias    string
}

// ConnectionHandler is a callback function for handling connections
type ConnectionHandler func(conn net.Conn)

// New creates a new Server
func New(addr string, proto string, alias string) (*Server, error) {
	listener, err := net.Listen(proto, addr)
	if err != nil {
		msg := fmt.Sprintf("could not start server %s/%s", proto, addr)
		return nil, errors.New(msg)
	}

	// Return Server
	return &Server{
		listener: listener,
		address:  addr,
		proto:    proto,
		alias:    alias,
	}, nil
}

// Close stops listening a Server
func (s *Server) Close() error {
	return s.listener.Close()
}

// DoMainLoop loops forever and accepts connections
// The sync.WaitGroup will tell this goroutine when to stop listening
// The ConnectionHandler will tell which function to call upon a connection
func (s *Server) DoMainLoop(wg *sync.WaitGroup, handler ConnectionHandler) {
	defer wg.Done()

	for {
		// Accept a new connection
		conn, err := s.listener.Accept()
		if err != nil {
			// Check if the error is due to the listener being closed
			_, ok := err.(net.Error)
			if ok {
				log.Printf("[%s(%s/%s)] Temporary error accepting connection: %v",
					s.alias, s.proto, s.address, err)
				continue
			}

			// Listener is closed or other non-temporary error
			log.Printf("[%s(%s/%s)] Error accepting connection: %v",
				s.alias, s.proto, s.address, err)
			return
		}

		// Handle the connection in a new goroutine
		log.Printf("[%s(%s/%s)] Request from: %s",
			s.alias, s.proto, s.address, conn.RemoteAddr())
		go handler(conn)
	}
}
