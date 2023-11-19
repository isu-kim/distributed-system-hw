package control

import (
	"lb/server"
	"log"
	"net"
	"sync"
)

// Handler represents a single control server
type Handler struct {
	server *server.Server
	addr   string
}

// New creates a new control server handler
func New() *Handler {
	// Parse control server address from environment variables
	addr := envParseAddress()
	controlServer, err := server.New(addr, "tcp", "controller")
	if err != nil {
		log.Fatalf("Could not start control server: %v", err)
		return nil
	}

	// Return new server
	return &Handler{
		server: controlServer,
		addr:   addr,
	}
}

// Run starts listening control server
func (h *Handler) Run(wg *sync.WaitGroup) error {
	h.server.DoMainLoop(wg, h.tempHandler)
	return nil
}

// Stop stops listening control server
func (h *Handler) Stop() error {
	return h.server.Close()
}

// tempHandler is a temp connection handler function for connection callbacks
func (h *Handler) tempHandler(conn net.Conn) {
	buffer := make([]byte, 1024)
	for {
		// Read data from the connection
		n, err := conn.Read(buffer)
		if err != nil {
			log.Printf("%s Error reading: %v", conn.RemoteAddr(), err)
			return
		}

		// Print the received data
		log.Printf("%s Sent: %s", conn.RemoteAddr(), buffer[:n])

		// Echo the data back to the client
		_, _ = conn.Write(buffer[:n])

		// Parse json from user's transmission
		userPayload, err := jsonParser(buffer[:n])
		if err != nil {
			log.Printf("%s Could not parse Json: %v", conn.RemoteAddr(), err)
		} else {
			log.Printf("%s Command Type: %d", conn.RemoteAddr(), userPayload)
		}
	}
}
