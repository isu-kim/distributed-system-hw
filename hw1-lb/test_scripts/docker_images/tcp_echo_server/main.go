package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	// Get environment variables
	listenAddr := os.Getenv("LISTEN_ADDR")
	listenPort := os.Getenv("LISTEN_PORT")
	lbAddr := os.Getenv("LB_ADDR")
	lbPort := os.Getenv("LB_PORT")

	// Start TCP server
	server, err := net.Listen("tcp", fmt.Sprintf("%s:%s", listenAddr, listenPort))
	if err != nil {
		log.Fatalf("Error Starting server: %s\n", err)
		return
	}
	defer server.Close()

	log.Printf("Server listening on %s:%s\n", listenAddr, listenPort)

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

	log.Printf("Registration message sent to %s:%s (%v)", lbAddr, lbPort, initMessage)

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
