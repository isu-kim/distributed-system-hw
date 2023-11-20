package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
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

	log.Printf("Starting up simple TCP echo server")
	log.Printf("Use CTRL+C (SIGINT) to send unregister command")

	stopper := make(chan os.Signal)
	signal.Notify(stopper, syscall.SIGINT, os.Interrupt)

	// signal handler for SIGINT, will send unregister command
	go func() {
		sig := <-stopper
		log.Printf("Received %v, Sending unregister command...", sig)

		// Send initialization message to LB_ADDR:LB_PORT
		initMessage := map[string]interface{}{
			"cmd":      "unregister",
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

		// Read the response from the server
		response := make([]byte, 512) // Adjust the buffer size based on your expected response size
		n, err := lbConn.Read(response)
		if err != nil {
			log.Fatalf("Error reading response: %s\n", err)
			return
		}

		// Parse the JSON response
		var jsonResponse map[string]interface{}
		err = json.Unmarshal(response[:n], &jsonResponse)
		if err != nil {
			log.Fatalf("Error parsing JSON response: %s\n", err)
			return
		}

		// Check the acknowledgment in the response
		acknowledgment, exists := jsonResponse["ack"].(string)
		if !exists {
			log.Fatalf("Invalid server response: %s\n", response[:n])
			return
		}

		// Handle the acknowledgment
		switch acknowledgment {
		case "successful":
			log.Println("Registration successful")
		case "failed":
			msg, exists := jsonResponse["msg"].(string)
			if !exists {
				log.Println("Registration failed without error message")
			} else {
				log.Printf("Registration failed: %s\n", msg)
			}
		default:
			log.Printf("Unexpected acknowledgment: %s\n", acknowledgment)
		}

		os.Exit(0)
	}()

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
		for {
			healthCheckMessage := map[string]interface{}{
				"ack": "hello",
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
			log.Printf("Received health check response: %s, sent back response\n", receivedMessage)
		}
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
