package control

// Predefined types of commands
const (
	cmdTypeRegister   = 1
	cmdTypeUnregister = 2
	cmdTypeHello      = 3
)

/*

// processRegister processes a register command
func (h *Handler) processRegister1(conn net.Conn, mapData map[string]interface{}) error {
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
	service := h.getExistingService(port, protocol)
	if service != nil {
		// This means that current registration is just adding a new replica to service
		log.Printf("%s Controller: %s/%d is existing service, adding a new replica...",
			common.ColoredInfo, misc.ConvertProtoToString(protocol), port)
		log.Printf("%s Controller: %s/%d has following replica: ",
			common.ColoredInfo, misc.ConvertProtoToString(protocol), port)

		// Print replicas of this service
		fmt.Printf("\t")
		for _, r := range h.replicas {
			if r.IsSpec(port, protocol) {
				fmt.Printf("%s, ", r.GetInfo())
			}
		}
		fmt.Printf("\n")
		service.IncrementReplica()

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

// getExistingService returns the service which is running given port and protocol
func (h *Handler) getExistingService2(port int, proto uint8) *server.Server {
	// This might be a critical section when multiple goroutines check for this port, use mutex to resolve this.
	h.lock.Lock()
	for _, childServer := range h.childServers {
		if childServer.IsSpec(port, proto) {
			h.lock.Unlock()
			return childServer
		}
	}

	h.lock.Unlock()
	return nil

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

*/
