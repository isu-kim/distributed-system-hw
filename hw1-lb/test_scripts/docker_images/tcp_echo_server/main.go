package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
)

func main() {
	// Get environment variables
	listenAddr := os.Getenv("LISTEN_ADDR")
	listenPortStr := os.Getenv("LISTEN_PORT")
	listenPort, err := strconv.Atoi(listenPortStr)
	if err != nil {
		log.Fatalf("Error converting listenPort to integer: %s\n", err)
		return
	}
	lbAddr := os.Getenv("LB_ADDR")
	lbPort := os.Getenv("LB_PORT")

	// Start TCP server
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%d", listenAddr, listenPort))
	if err != nil {
		log.Fatalf("Error starting server: %s\n", err)
		return
	}
	defer server.Close()

	log.Printf("Server listening on %s:%d\n", listenAddr, listenPort)

	// Send initialization message to LB_ADDR:LB_PORT
	initMessage := map[string]interface{}{
		"cmd":      "register",
		"protocol": "tcp",
		"port":     listenPort,
	}
	initMessageJSON, err := json.Marshal(initMessage)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %s\n", err)
		return
	}

	// Connect to LB_ADDR:LB_PORT and send the initialization message
	lbConn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", lbAddr, lbPort))
	if err != nil {
		log.Fatalf("Error connecting to %s:%s: %v\n", lbAddr, lbPort, err)
		return
	}
	defer lbConn.Close()

	_, err = lbConn.Write(initMessageJSON)
	if err != nil {
		log.Fatalf("Error sending registration message: %s\n", err)
		return
	}

	log.Printf("Registration message sent to %s:%s (%v)\n", lbAddr, lbPort, initMessage)

	// Start a goroutine for health check
	go func() {
		healthCheckMessage := map[string]interface{}{
			"cmd": "hello",
		}
		healthCheckJSON, err := json.Marshal(healthCheckMessage)
		if err != nil {
			log.Printf("Error marshaling health check JSON: %s\n", err)
			return
		}

		_, err = lbConn.Write(healthCheckJSON)
		if err != nil {
			log.Printf("Error sending health check message: %s\n", err)
			return
		}

		// Read the response for health check
		buffer := make([]byte, 1024)
		n, err := lbConn.Read(buffer)
		if err != nil {
			log.Printf("Error reading from connection during health check: %s\n", err)
			return
		}

		receivedMessage := string(buffer[:n])
		fmt.Printf("Received health check response: %s\n", receivedMessage)
	}()

	// Accept and handle incoming connections
	for {
		conn, err := server.Accept()
		if err != nil {
			log.Printf("[%s] Error accepting connection: %s\n", conn.RemoteAddr(), err)
			continue
		}

		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	// Log connection information
	remoteAddr := conn.RemoteAddr().String()

	// Read incoming message
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		log.Printf("[%s] Error reading from connection: %s\n", conn.RemoteAddr(), err)
		return
	}

	receivedMessage := string(buffer[:n])
	fmt.Printf("[%s] Received message: %s\n", remoteAddr, receivedMessage)
}
