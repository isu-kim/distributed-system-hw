package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
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

	log.Printf("Starting up simple API echo server")
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
			_, err = lbConn.Read(buffer)
			if err != nil {
				log.Printf("Error reading from connection during health check: %s\n", err)
				return
			}

			//receivedMessage := string(buffer[:n])
			//log.Printf("Received health check response: %s, sent back response\n", receivedMessage)
		}
	}()

	serverAddr := fmt.Sprintf("%s:%d", listenAddr, listenPort)
	log.Printf("Starting server on %s...\n", serverAddr)

	http.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
		// Read the request body
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		// Get the IP address of the server machine
		ip, err := getServerIP()
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting server IP: %s", err), http.StatusInternalServerError)
			return
		}

		// Print the received payload and server IP address
		log.Printf("Received payload: %s\n", string(body))
		log.Printf("Server IP address: %s\n", ip)

		// Send the payload and IP address back to the client
		w.Write([]byte(fmt.Sprintf("Received payload: %s\nServer IP address: %s\n", body, ip)))
	})

	err = http.ListenAndServe(serverAddr, nil)
	if err != nil {
		log.Fatalf("Error starting server: %s\n", err)
	}

}

// getServerIP returns the IP address of the server machine
func getServerIP() (string, error) {
	cmd := exec.Command("ifconfig", "eth0")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Extract the IP address from the ifconfig output
	ipStartIndex := strings.Index(string(output), "inet ") + 5
	ipEndIndex := ipStartIndex + strings.Index(string(output[ipStartIndex:]), " ")
	ip := string(output[ipStartIndex:ipEndIndex])

	return ip, nil
}
