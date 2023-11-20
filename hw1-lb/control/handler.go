package control

import (
	"lb/common"
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
		log.Fatalf("%s Could not start control server: %v", common.ColoredError, err)
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
			log.Printf("%s Controller: Src=%s error reading: %v",
				common.ColoredWarn, conn.RemoteAddr(), err)
			return
		}

		// Print the received data
		log.Printf("%s Controller received %s [src=%s]",
			common.ColoredInfo, buffer[:n], conn.RemoteAddr())

		// Echo the data back to the client
		_, _ = conn.Write(buffer[:n])

		// Parse json from user's transmission
		userPayload, err := jsonParser(buffer[:n])
		if err != nil {
			log.Printf("%s Controller could not parse JSON [src=%s]: %v",
				common.ColoredWarn, conn.RemoteAddr(), err)
		}

		// Parse command type
		commandType, err := parseCommandType(userPayload)
		if err != nil {
			log.Printf("%s Controller could not parse command type [src=%s]: %v",
				common.ColoredWarn, conn.RemoteAddr(), err)
		}

		log.Printf("%s Controller received command %d [src=%s]",
			common.ColoredInfo, commandType, conn.RemoteAddr())
	}
}
