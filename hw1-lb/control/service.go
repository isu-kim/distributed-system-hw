package control

import (
	"fmt"
	"lb/common"
	"lb/misc"
	"lb/server"
	"log"
	"net"
	"strings"
	"sync"
)

// service represents a single service exposed by load balancer
type service struct {
	addr               string
	port               int
	proto              uint8
	server             *server.Server
	replicas           []*Replica
	lock               sync.Mutex
	lastScheduledIndex int
	isLive             bool
}

// isGivenSpec returns if given spec matches current service, if we are looking at address as well, use isExactGivenSpec
func (s *service) isGivenSpec(port int, proto uint8) bool {
	return s.port == port && s.proto == proto
}

// isExactGivenSpec returns if given spec matches current service
func (s *service) isExactGivenSpec(address string, port int, proto uint8) bool {
	return s.port == port && s.proto == proto && strings.Compare(s.addr, address) == 0
}

// addReplica adds a new Replica into the service
// Since this might be used in multiple goroutines, the function is thread safe by using mutex
func (s *service) addReplica(r *Replica) {
	s.lock.Lock()
	s.replicas = append(s.replicas, r)
	s.lock.Unlock()
}

// removeReplica removes a Replica from service
// If there was any removed replica from given service, this will return true
// Removing Replica will trigger if this service shall be terminated or not
func (s *service) removeReplica(target Replica) bool {
	ret := false

	// Slice of Replicas which are being kept
	var updatedReplicas []*Replica

	s.lock.Lock()
	// Iterate over the existing replicas
	for _, r := range s.replicas {
		// Check if the replica matches the specified criteria
		if !r.Equals(target) {
			// If it doesn't match, add it to the updatedReplicas slice
			updatedReplicas = append(updatedReplicas, r)
		} else {
			ret = true
		}
	}
	s.replicas = updatedReplicas
	s.lock.Unlock()

	// Check if this service shall be terminated or not
	if s.shouldBeTerminated() {
		log.Printf("%s Service %s/%s:%d has no more replica left, terminating server",
			common.ColoredInfo, misc.ConvertProtoToString(s.proto), s.addr, s.port)

		err := s.terminateService()
		if err != nil {
			log.Printf("%s Service %s/%s:%d cannot terminate server: %v",
				common.ColoredError, misc.ConvertProtoToString(s.proto), s.addr, s.port, err)
		}

		// Set current service as dead
		s.isLive = false
	}

	return ret
}

// getReplicas returns slice of all Replica for this service
// Since retrieving the Replica might result in race condition, this is thread safe by using mutex
func (s *service) getReplicas() []*Replica {
	var tmp []*Replica
	s.lock.Lock()
	tmp = s.replicas
	s.lock.Unlock()

	return tmp
}

// shouldBeTerminated returns if this service shall be terminated or not
// If the service has no live replicas, the server for this service shall be terminated
func (s *service) shouldBeTerminated() bool {
	replicas := s.getReplicas()
	return len(replicas) == 0
}

// terminateService terminates the server running for this service
func (s *service) terminateService() error {
	return s.server.Close()
}

// doLB picks a replica and sends the traffic from conn to the target replica server
// We can of course ignore race conditions and do a bit more dangerous connections, but
// for stability, we are going to use mutex and avoid any possible race conditions
// When autoTry was set on, the function will pick next replica when current replica fails
func (s *service) doLB(srcConn net.Conn) {

	// The replicas that are possible to be scheduled
	replicas := s.getReplicas()
	replicaLen := len(replicas)

	log.Printf("REPLICAS: %v (%d)", replicas, replicaLen)

	s.lock.Lock()
	// Perform simple round-robin algorithm
	// If this was the last element or exceeds it, send it to the beginning
	// Golang supports short circuit evaluation, meaning that if we only had single replica for a single server
	// This will fire up the first part of evaluation expression and set the if statement as true
	// If we explicitly checked if our replica count was 0, there will be additional cost for evaluating
	// whether the replica count was 0. So this will be effective and faster.
	if s.lastScheduledIndex+1 == replicaLen || s.lastScheduledIndex+1 > replicaLen {
		s.lastScheduledIndex = 0
	} else {
		s.lastScheduledIndex = s.lastScheduledIndex + 1
	}

	// The target replica that was selected
	targetReplica := replicas[s.lastScheduledIndex]
	targetAddr := fmt.Sprintf("%s:%d", targetReplica.addr, targetReplica.port)
	targetProto := misc.ConvertProtoToString(targetReplica.proto)

	s.lock.Unlock()

	// For debugging purpose
	log.Printf("%s Forwarding %s -> %s proto=%s / index=%d",
		common.ColoredInfo, srcConn.RemoteAddr(), targetAddr, targetProto, s.lastScheduledIndex)

	// Forward traffic from srcConn to targetAddr
	err := forwardTraffic(srcConn, targetAddr, targetProto)
	if err != nil {
		log.Printf("%s Forwarding %s -> %s proto=%s / index=%d failed: %v ",
			common.ColoredWarn, srcConn.RemoteAddr(), targetAddr, targetProto, s.lastScheduledIndex, err)
	}
}
