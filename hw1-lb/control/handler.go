package control

import (
	"errors"
	"fmt"
	"lb/common"
	"lb/misc"
	"lb/server"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Handler represents a single control server
type Handler struct {
	server         *server.Server
	addr           string
	lock           sync.Mutex
	childServers   []*server.Server
	healthCheckWg  sync.WaitGroup
	childServersWg sync.WaitGroup
	services       []*service
}

// garbageCollectionRequest represents a single garbage collection request
// this is meant to trigger cleaning up unused servers (services) which do not have any replicas
type garbageCollectionRequest struct {
	addr  string
	port  int
	proto uint8
}

var gcChannel chan garbageCollectionRequest

// New creates a new control server handler
func New() *Handler {
	// Parse control server address from environment variables
	addr, port := envParseAddress()
	controlServer, err := server.New(addr, port, "tcp", "controller")
	if err != nil {
		log.Fatalf("%s Could not start control server: %v", common.ColoredError, err)
		return nil
	}

	// Return new server
	return &Handler{
		server:         controlServer,
		addr:           addr,
		lock:           sync.Mutex{},
		childServers:   make([]*server.Server, 0),
		healthCheckWg:  sync.WaitGroup{},
		childServersWg: sync.WaitGroup{},
		services:       make([]*service, 0),
	}
}

// Run starts listening control server
func (h *Handler) Run(wg *sync.WaitGroup) error {
	h.setupSignalHandling()
	gcChannel = make(chan garbageCollectionRequest)
	// h.garbageCollectorRoutine()
	h.server.DoMainLoop(wg, h.tempHandler)
	return nil
}

// Stop stops listening control server
func (h *Handler) Stop() error {
	return h.server.Close()
}

func (h *Handler) setupSignalHandling() {
	stopper := make(chan os.Signal)
	signal.Notify(stopper, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	go func() {
		sig := <-stopper
		log.Printf("%s Controller received signal: %v. Shutting down...", common.ColoredInfo, sig)
		os.Exit(0)
	}()
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

		// If this command was register, start up a new server
		switch commandType {
		case common.CmdTypeRegister:
			err := h.processRegister(conn, userPayload)
			if err != nil {
				log.Printf("%s Controller could not add a new replica [src=%s]: %v",
					common.ColoredError, conn.RemoteAddr(), err)
			}
		default:
			log.Printf("%s Controller received unknown command type [src=%s]",
				common.ColoredWarn, conn.RemoteAddr())
		}
		log.Printf("%s Controller received command %d [src=%s]",
			common.ColoredInfo, commandType, conn.RemoteAddr())
	}
}

// processRegister processes a register command
func (h *Handler) processRegister(conn net.Conn, mapData map[string]interface{}) error {
	// Parse raw IP address from the remote Addr, simply retrieve the part before port binding
	addrParts := strings.Split(conn.RemoteAddr().String(), ":")
	if len(addrParts) != 2 {
		msg := fmt.Sprintf("could not parse remote address: %s", conn.RemoteAddr().String())
		return errors.New(msg)
	}

	// Parse management command received
	protocol, port, err := parseManagementCommand(mapData)
	if err != nil {
		msg := fmt.Sprintf("could not parse management command: %v", err)
		return errors.New(msg)
	}

	// Check if service already exists
	targetService := h.getExistingService(port, protocol)
	if targetService != nil {
		if targetService.isLive {
			// This means we are registering a new replica for the service
			log.Printf("%s Controller: %s/%d is existing service, adding a new replica (total %d availble replicas)",
				common.ColoredInfo, misc.ConvertProtoToString(protocol), port, len(targetService.getReplicas()))
		} else {
			// This means that the service has no replica, thus had its server terminated, we need to restart server
			log.Printf("%s Controller: %s/%d is existing service, however had its server terminated, restarting server",
				common.ColoredInfo, misc.ConvertProtoToString(protocol), port)
			targetService, err = h.restartService(port, protocol, targetService)
			if err != nil {
				return err
			}
		}
	} else {
		// This means we are registering a new service
		log.Printf("%s Controller: %s/%d is a new service",
			common.ColoredInfo, misc.ConvertProtoToString(protocol), port)

		// Create a new service
		targetService, err = h.createNewService(port, protocol)
		if err != nil {
			return err
		}
	}

	// Create a new replica
	newReplica := Replica{
		addr:            addrParts[0],
		port:            port,
		proto:           protocol,
		healthCheckConn: conn,
		lastHealthCheck: time.Time{},
		ownerService:    targetService,
	}

	// Now add replica to the service, also start health checking the replica
	targetService.addReplica(&newReplica)
	newReplica.StartHealthCheckRoutine()

	return nil
}

// getExistingService retrieves existing service by given address and port with protocol information
// If no such service with given spec was found, nil will be returned
func (h *Handler) getExistingService(port int, proto uint8) *service {
	for _, s := range h.services {
		if s.isGivenSpec(port, proto) {
			return s
		}
	}

	return nil
}

// createNewService creates a new service with given port and protocol
// This will also start up the Server for that service as well
func (h *Handler) createNewService(port int, proto uint8) (*service, error) {
	// Retrieve LB_LISTEN_ADDR as load balancer listen address
	lbIPAddr := os.Getenv("LB_LISTEN_ADDR")
	if len(lbIPAddr) == 0 {
		log.Printf("%s $LB_LISTEN_ADDR not set, defaulting to 0.0.0.0", common.ColoredWarn)
		lbIPAddr = "0.0.0.0"
	}

	// Convert protocol as string
	protoString := misc.ConvertProtoToString(proto)

	// Start listening a new server
	newServer, err := server.New(lbIPAddr, port, protoString, "")
	if err != nil {
		log.Printf("%s Controller failed to start a new service at %s/%s:%d: %v",
			common.ColoredError, protoString, lbIPAddr, port, err)
		msg := fmt.Sprintf("failed to start server at %s/%s:%d: %v",
			protoString, lbIPAddr, port, err)
		return nil, errors.New(msg)
	}

	// Create a new service information
	newService := service{
		addr:               lbIPAddr,
		port:               port,
		proto:              proto,
		server:             newServer,
		replicas:           make([]*Replica, 0),
		lock:               sync.Mutex{},
		lastScheduledIndex: 0,
		isLive:             true,
	}

	// Register service to server
	// This is a critical section, so lock with mutex
	h.lock.Lock()
	h.services = append(h.services, &newService)
	h.lock.Unlock()

	// Everything went on properly
	return &newService, nil
}

// restartService will restart the server for the service
func (h *Handler) restartService(port int, proto uint8, existingService *service) (*service, error) {
	// Retrieve LB_LISTEN_ADDR as load balancer listen address
	lbIPAddr := os.Getenv("LB_LISTEN_ADDR")
	if len(lbIPAddr) == 0 {
		log.Printf("%s $LB_LISTEN_ADDR not set, defaulting to 0.0.0.0", common.ColoredWarn)
		lbIPAddr = "0.0.0.0"
	}

	// Convert protocol as string
	protoString := misc.ConvertProtoToString(proto)

	// Start listening a new server
	newServer, err := server.New(lbIPAddr, port, protoString, "")
	if err != nil {
		log.Printf("%s Controller failed to start a new service at %s/%s:%d: %v",
			common.ColoredError, protoString, lbIPAddr, port, err)
		msg := fmt.Sprintf("failed to start server at %s/%s:%d: %v",
			protoString, lbIPAddr, port, err)
		return nil, errors.New(msg)
	}

	existingService.isLive = true
	existingService.server = newServer
	return existingService, nil
}
