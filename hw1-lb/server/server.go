package server

import (
	"errors"
	"fmt"
	"lb/common"
	"lb/misc"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
)

// Server represents a single server
type Server struct {
	tcpListener net.Listener
	udpConn     net.Conn
	address     string
	port        int
	proto       uint8
	alias       string
	stopper     chan os.Signal
}

// ConnectionHandler is a callback function for handling connections
type ConnectionHandler func(conn net.Conn)

// New creates a new Server
func New(addr string, port int, proto string, alias string) (*Server, error) {
	// Convert proto from string to uint8
	var protoConverted uint8
	if strings.Compare(proto, "tcp") == 0 {
		protoConverted = common.TypeProtoTCP
	} else if strings.Compare(proto, "udp") == 0 {
		protoConverted = common.TypeProtoUDP
	}

	listenAddr := fmt.Sprintf("%s:%d", addr, port)

	// If this is a TCP server
	if protoConverted == common.TypeProtoTCP {
		listener, err := net.Listen(proto, listenAddr)
		if err != nil {
			msg := fmt.Sprintf("could not start server %s/%s: %v", proto, addr, err)
			return nil, errors.New(msg)
		}

		// Return Server
		return &Server{
			tcpListener: listener,
			address:     addr,
			port:        port,
			proto:       protoConverted,
			alias:       alias,
			stopper:     make(chan os.Signal),
		}, nil
	} else if protoConverted == common.TypeProtoUDP {
		// This is a UDP server
		server, err := net.ListenUDP("udp", &net.UDPAddr{
			IP:   net.ParseIP(addr),
			Port: port,
		})

		if err != nil {
			msg := fmt.Sprintf("could not start server %s/%s: %v", proto, addr, err)
			return nil, errors.New(msg)
		}

		return &Server{
			udpConn: server,
			address: addr,
			port:    port,
			proto:   protoConverted,
			alias:   alias,
			stopper: make(chan os.Signal),
		}, nil
	} else {
		return nil, errors.New("unknown protocol")
	}
}

// Close stops listening a Server
func (s *Server) Close() error {
	signal.Notify(s.stopper, os.Interrupt)
	if s.proto == common.TypeProtoTCP {
		return s.tcpListener.Close()
	} else if s.proto == common.TypeProtoUDP {
		return s.udpConn.Close()
	} else {
		return errors.New("unknown protocol")
	}
}

// DoMainLoop loops forever and accepts connections
// The sync.WaitGroup will tell this goroutine when to stop listening
// The ConnectionHandler will tell which function to call upon a connection
func (s *Server) DoMainLoop(wg *sync.WaitGroup, handler ConnectionHandler) {
	if s.proto == common.TypeProtoTCP {
		for {
			select {
			case <-s.stopper:
				log.Printf("%s Server %s/%s:%d received interrupt",
					common.ColoredInfo, misc.ConvertProtoToString(s.proto), s.address, s.port)
				return
			default:
				// Accept a new connection
				conn, err := s.tcpListener.Accept()
				if err != nil {
					// Check if the error is due to the tcpListener being closed
					_, ok := err.(net.Error)
					if ok {
						continue
					}

					// Listener is closed or other non-temporary error
					log.Printf("%s \"%s(%s/%s)\" Error accepting connection: %v",
						common.ColoredWarn, s.alias, s.proto, s.address, err)
					return
				}

				// Handle the connection in a new goroutine
				/**
				log.Printf("%s \"%s(%s/%s)\" Request from: %s",
					common.ColoredInfo, s.alias, misc.ConvertProtoToString(s.proto), s.address, conn.RemoteAddr())
				*/
				go handler(conn)
			}
		}
	} else if s.proto == common.TypeProtoUDP {
		for {
			select {
			case <-s.stopper:
				log.Printf("%s Server %s/%s:%d received interrupt",
					common.ColoredInfo, misc.ConvertProtoToString(s.proto), s.address, s.port)
				return
			default:
				handler(s.udpConn)
			}
		}
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
