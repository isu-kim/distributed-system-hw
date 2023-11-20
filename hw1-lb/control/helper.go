package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"lb/common"
	"lb/misc"
	"lb/relay"
	"lb/server"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// Predefined types of commands
const (
	cmdTypeRegister   = 1
	cmdTypeUnregister = 2
	cmdTypeHello      = 3
)

// envParseAddress parses environment variables and retrieves address to listen control server on
// - LB_LISTEN_ADDR: the IP address to listen control server on (defaults 0.0.0.0)
// - LB_LISTEN_PORT: the port number to listen control server on (defaults 8080)
func envParseAddress() (string, int) {
	// Retrieve listen address information from environment variable
	envAddr := os.Getenv("LB_LISTEN_ADDR")
	ipv4Addr, _, err := net.ParseCIDR(envAddr)
	if err != nil {
		log.Printf("%s Could not parse environment variable $LB_LISTEN_ADDR in IP address format: %v",
			common.ColoredWarn, err)
		log.Printf("%s Defaulting listen IP to 0.0.0.0", common.ColoredWarn)
		ipv4Addr = net.IPv4(0, 0, 0, 0) // 0.0.0.0
	}

	// Retrieve port information from environment variable
	envPort := os.Getenv("LB_LISTEN_PORT")
	portVal, err := strconv.Atoi(envPort)
	if err != nil {
		log.Printf("%s Could not parse environment variable $LB_LISTEN_PORT as integer: %v", common.ColoredWarn, err)
		log.Printf("%s Defaulting back to 8080", common.ColoredWarn)
		portVal = 8080
	}

	return ipv4Addr.String(), portVal
}

// jsonParser decodes raw bytes into JSON object
func jsonParser(bytes []byte) (map[string]interface{}, error) {
	var ret map[string]interface{}
	err := json.Unmarshal(bytes, &ret)
	return ret, err
}

// parseCommandType parses command type (register, unregister) from a map
func parseCommandType(mapData map[string]interface{}) (uint8, error) {
	// Try parsing value of "cmd" as string
	if mapData["cmd"] == nil {
		msg := fmt.Sprintf("Invalid cmd input: %v", mapData["cmd"])
		return 0, errors.New(msg)
	}

	cmd := mapData["cmd"].(string)
	cmd = strings.ToLower(cmd)

	if strings.Compare(cmd, "register") == 0 { // Register command
		return cmdTypeRegister, nil
	} else if strings.Compare(cmd, "unregister") == 0 { // Unregister command
		return cmdTypeUnregister, nil
	} else if strings.Compare(cmd, "hello") == 0 { // Hello command (health check)
		return cmdTypeHello, nil
	} else {
		return 0, nil
	}
}

// parseManagementCommand parses management commands which are register and unregister
func parseManagementCommand(mapData map[string]interface{}) (uint8, int, error) {
	// Check if protocol key is present
	protocol, ok := mapData["protocol"].(string)
	if !ok {
		return 0, 0, errors.New("missing or invalid 'protocol' key")
	}

	// Check if port key is present
	// We are intentionally converting into float since float is not possible port number
	// so that we can compare if the user's input was actually a float later by comparing its integer value
	port, ok := mapData["port"].(float64)
	if !ok {
		return 0, 0, errors.New("missing or invalid 'port' key")
	}

	// Check if port is a positive integer in the valid port range
	if port <= 0 || port > 65535 || port != float64(int(port)) {
		return 0, 0, errors.New("invalid port, must be a positive integer in the range [1, 65535]")
	}

	// Convert protocol type
	protoType := 0
	if strings.Compare(protocol, "tcp") == 0 {
		protoType = common.TypeProtoTCP
	} else if strings.Compare(protocol, "udp") == 0 {
		protoType = common.TypeProtoUDP
	}

	return uint8(protoType), int(port), nil
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

	// Create a new replica
	newReplica := replica{
		addr:            addrParts[0],
		port:            port,
		proto:           protocol,
		healthCheckConn: conn,
		lastHealthCheck: time.Time{},
	}

	// Check if lb shall create a new server or not
	if h.isExistingService(port, protocol) {
		// This means that current registration is just adding a new replica to service
		log.Printf("%s Controller: %s/%d is existing service",
			common.ColoredInfo, misc.ConvertProtoToString(protocol), port)
	} else {
		// This means that current registration is a new service, so fire up a new server
		log.Printf("%s Controller: %s/%d is a new service",
			common.ColoredInfo, misc.ConvertProtoToString(protocol), port)
		err := h.startServiceServer(port, protocol)
		if err != nil {
			return err
		}
	}

	// Add new replica
	// This is a critical section, so use mutex here
	h.lock.Lock()
	h.replicas = append(h.replicas, &newReplica)
	h.lock.Unlock()

	// Start health check routine for the replica
	newReplica.StartHealthCheckRoutine()

	return nil
}

// isExistingService checks if given port and protocol was being served by load balancer before
func (h *Handler) isExistingService(port int, proto uint8) bool {
	// This might be a critical section when multiple goroutines check for this port, use mutex to resolve this.

	log.Printf("isExistingService called, port:%d proto:%d, child server len:%d", port, proto, len(h.childServers))
	h.lock.Lock()
	for _, childServer := range h.childServers {
		if childServer.IsSpec(port, proto) {
			return true
		}
	}

	h.lock.Unlock()
	return false
}

// startServiceServer starts a new server with given protocol and port
func (h *Handler) startServiceServer(port int, proto uint8) error {
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
		log.Printf("%s Controller: Failed to start a new service at %s/%s:%d: %v",
			common.ColoredError, protoString, lbIPAddr, port, err)
		msg := fmt.Sprintf("failed to start server at %s/%s:%d: %v",
			protoString, lbIPAddr, port, err)
		return errors.New(msg)
	}

	// Add a new wait group for child server
	h.childServersWg.Add(1)
	go newServer.DoMainLoop(&h.childServersWg, relay.TempRelay)

	// Lock child servers, this is critical section
	// We are adding new server into child servers
	h.lock.Lock()
	h.childServers = append(h.childServers, newServer)
	h.lock.Unlock()

	log.Printf("%s Controller: Successfully started new service at %s/%s:%d",
		common.ColoredInfo, protoString, lbIPAddr, port)

	return nil
}

// stopServiceServer stops an existing server for a given protocol and port
// @todo implement a garbage collector!
func (h *Handler) stopServiceServer(port int, proto uint8) error {
	return nil
}
