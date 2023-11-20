package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"lb/common"
	"lb/misc"
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

// replica represents a single replica for load balancing
type replica struct {
	addr            string
	port            int
	proto           uint8
	lastHealthCheck time.Time
	healthCheckConn net.Conn
}

// StartHealthCheckRoutine starts loop for health check for given replica forever
func (s *replica) StartHealthCheckRoutine() {
	go func() {
		curFailure := 0
		maxFailure := 5
		healthCheckInterval := 2

		// Retrieve HEALTH_CHECK_MAX_FAILURE for max retrials before considering a service dead
		// If not set, defaults to 5 times
		maxFailureString := os.Getenv("HEALTH_CHECK_MAX_FAILURE")
		if len(maxFailureString) == 0 {
			maxFailure = 5
		} else {
			val, err := strconv.Atoi(maxFailureString)
			if err != nil {
				maxFailure = 5
			} else {
				maxFailure = val
			}
		}

		// Retrieve HEALTH_CHECK_INTERVAL for duration in seconds between each health checks
		// If not set, defaults to 2 seconds
		healthCheckIntervalString := os.Getenv("HEALTH_CHECK_INTERVAL")
		if len(healthCheckIntervalString) == 0 {
			healthCheckInterval = 2
		} else {
			val, err := strconv.Atoi(healthCheckIntervalString)
			if err != nil {
				healthCheckInterval = 2
			} else {
				healthCheckInterval = val
			}
		}

		for {
			err := performHealthCheck(s.healthCheckConn)
			if err != nil {
				// Health check failed, warn user
				log.Printf("%s Health check failed for %s/%s:%d (%d/%d), last reported: %s",
					common.ColoredWarn, misc.ConvertProtoToString(s.proto), s.addr, s.port, curFailure, maxFailure,
					s.lastHealthCheck.String())
				curFailure++
			} else {
				// Health check successfully finished, reset failure count and set last health check time
				curFailure = 0
				s.lastHealthCheck = time.Now()
				log.Printf("%s Health check finished for %s/%s:%d (%d/%d), last reported: %s",
					common.ColoredInfo, misc.ConvertProtoToString(s.proto), s.addr, s.port, curFailure, maxFailure,
					s.lastHealthCheck.String())
			}

			// Reached max health check failures
			if curFailure >= maxFailure {
				log.Printf("%s Max health check failure count reached for %s/%s:%d (%d/%d)",
					common.ColoredError, misc.ConvertProtoToString(s.proto), s.addr, s.port, curFailure, maxFailure)
				break
			}

			// Sleep duration until next health check
			time.Sleep(time.Duration(healthCheckInterval) * time.Second)
		}

		log.Printf("%s Controller triggered garbage collection for %s/%s:%d due to max health check failure",
			common.ColoredInfo, misc.ConvertProtoToString(s.proto), s.addr, s.port)

		// @todo trigger garbage collection for the given service
		// perhaps create a new channel for garbage collection?
	}()
}

// performHealthCheck checks health for the target server
// This will send {"cmd":"hello"} and will expect result {"ack":"hello"}
// If the response was something else, the replica will be regarded as a failed health check
// Also, this will have a 5 sec timeout until the controller considers the replica to be "dead"
func performHealthCheck(conn net.Conn) error {
	// Retrieve HEALTH_CHECK_TIMEOUT for max time out for health check
	// If not set, defaults to 5 seconds
	timeout := 5
	maxTimeoutString := os.Getenv("HEALTH_CHECK_TIMEOUT")
	if len(maxTimeoutString) == 0 {
		timeout = 5
	} else {
		val, err := strconv.Atoi(maxTimeoutString)
		if err != nil {
			timeout = 5
		} else {
			timeout = val
		}
	}

	// Define the health check request
	request := map[string]string{"cmd": "hello"}
	requestJSON, err := json.Marshal(request)
	if err != nil {
		msg := fmt.Sprintf("error encoding health check request: %v", err)
		return errors.New(msg)
	}

	// Send the health check request
	_, err = conn.Write(requestJSON)
	if err != nil {
		return errors.New(fmt.Sprintf("error sending health check request: %v", err))
	}

	// Set a deadline for reading
	err = conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	if err != nil {
		return errors.New(fmt.Sprintf("error setting read deadline: %v", err))
	}

	// Read the response
	responseBuffer := make([]byte, 1024)
	n, err := conn.Read(responseBuffer)
	if err != nil {
		// Check if the error is due to a timeout
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			msg := fmt.Sprintf("health check timed out (%ds)", timeout)
			return errors.New(msg)
		}
		msg := fmt.Sprintf("error reading health check response: %v", err)
		return errors.New(msg)
	}

	// Parse the response JSON
	var response map[string]string
	err = json.Unmarshal(responseBuffer[:n], &response)
	if err != nil {
		msg := fmt.Sprintf("error decoding health check response: %v", err)
		return errors.New(msg)
	}

	// Check if the response is as expected
	expectedResponse := map[string]string{"ack": "hello"}
	if !misc.AreMapsEqual(response, expectedResponse) {
		msg := fmt.Sprintf("unexpected health check response: %v", response)
		return errors.New(msg)
	}

	return nil
}
