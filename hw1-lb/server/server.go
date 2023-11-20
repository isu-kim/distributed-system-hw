package server

import (
	"errors"
	"fmt"
	"lb/common"
	"log"
	"net"
	"strings"
	"sync"
)

// Server represents a single server
type Server struct {
	listener net.Listener
	address  string
	port     int
	proto    uint8
	alias    string
}

// ConnectionHandler is a callback function for handling connections
type ConnectionHandler func(conn net.Conn)

// New creates a new Server
func New(addr string, port int, proto string, alias string) (*Server, error) {
	listenAddr := fmt.Sprintf("%s:%d", addr, port)
	listener, err := net.Listen(proto, listenAddr)
	if err != nil {
		msg := fmt.Sprintf("could not start server %s/%s: %v", proto, addr, err)
		return nil, errors.New(msg)
	}

	// Convert proto from string to uint8
	var protoConverted uint8
	if strings.Compare(proto, "tcp") == 0 {
		protoConverted = common.TypeProtoTCP
	} else if strings.Compare(proto, "udp") == 0 {
		protoConverted = common.TypeProtoUDP
	}

	// Return Server
	return &Server{
		listener: listener,
		address:  addr,
		port:     port,
		proto:    protoConverted,
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
				log.Printf("%s \"%s(%s/%s)\" Temporary error accepting connection: %v",
					common.ColoredWarn, s.alias, s.proto, s.address, err)
				continue
			}

			// Listener is closed or other non-temporary error
			log.Printf("%s \"%s(%s/%s)\" Error accepting connection: %v",
				common.ColoredWarn, s.alias, s.proto, s.address, err)
			return
		}

		// Handle the connection in a new goroutine
		log.Printf("%s \"%s(%s/%s)\" Request from: %s",
			common.ColoredInfo, s.alias, s.proto, s.address, conn.RemoteAddr())
		go handler(conn)
	}
}

// IsSpec returns if given spec was the one running this Server
func (s *Server) IsSpec(port int, proto uint8) bool {
	return s.port == port && s.proto == proto
}

// GetInfo returns a string describing the listen address
func (s *Server) GetInfo() string {
	return fmt.Sprintf("%s:%d", s.address, s.port)
}
