package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"lb/common"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
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
func jsonParser(payload []byte) (map[string]interface{}, error) {
	var ret map[string]interface{}

	b := bytes.NewBuffer(payload)
	d := json.NewDecoder(b)

	err := d.Decode(&ret)
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
		return common.CmdTypeRegister, nil
	} else if strings.Compare(cmd, "unregister") == 0 { // Unregister command
		return common.CmdTypeUnregister, nil
	} else if strings.Compare(cmd, "hello") == 0 { // Hello command (health check)
		return common.CmdTypeHello, nil
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

// returnResult returns result for the connection
func returnResult(conn net.Conn, err error, commandType uint8) {
	typeString := ""
	switch commandType {
	case common.CmdTypeRegister:
		typeString = "register"
		break
	case common.CmdTypeUnregister:
		typeString = "unregister"
		break
	default:
		typeString = "unknown"
	}

	if err != nil {
		log.Printf("%s Controller could not %s replica [src=%s]: %v",
			common.ColoredError, typeString, conn.RemoteAddr(), err)

		// Registration failed, send acknowledgment with error message to the client
		failureResponse := map[string]string{"ack": "failed", "msg": err.Error()}
		failureResponseJSON, err := json.Marshal(failureResponse)
		if err != nil {
			log.Printf("%s Controller Error encoding failure response [src=%s]: %v\n",
				common.ColoredError, conn.RemoteAddr(), err)
			return
		}

		_, err = conn.Write(failureResponseJSON)
		if err != nil {
			log.Printf("%s Error writing failure response to client [src=%s]: %v\n",
				common.ColoredError, conn.RemoteAddr(), err)
			return
		}
	} else {
		// Registration successful, send acknowledgment to the client
		successResponse := map[string]string{"ack": "successful"}
		successResponseJSON, err := json.Marshal(successResponse)
		if err != nil {
			log.Printf("%s Error encoding success response [src=%s]: %v\n",
				common.ColoredError, conn.RemoteAddr(), err)
			return
		}

		_, err = conn.Write(successResponseJSON)
		if err != nil {
			log.Printf("%s Error writing success response to client [src=%s]: %v\n",
				common.ColoredError, conn.RemoteAddr(), err)
			return
		}
	}
}

// closeConnectionWithTimeout closes net.Conn connection with given timeout
// The code was generated by ChatGPT
func closeConnectionWithTimeout(conn net.Conn, timeout time.Duration) error {
	// Set a deadline for the connection to complete the close operation
	err := conn.SetDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}

	// Perform the close operation in a goroutine
	ch := make(chan error, 1)
	go func() {
		ch <- conn.Close()
	}()

	// Wait for the close operation to complete or for the deadline to expire
	select {
	case <-time.After(timeout):
		return errors.New("close operation timed out")
	case err := <-ch:
		return err
	}
}

// forwardTraffic forwards traffic from srcConn to target address
// The code was generated by ChatGPT
func forwardTraffic(srcConn net.Conn, targetAddr, targetProto string) error {
	// Establish a connection to the target server
	targetConn, err := net.Dial(targetProto, targetAddr)
	if err != nil {
		return err
	}
	defer targetConn.Close()

	// Use goroutines to forward traffic in both directions
	go func() {
		// Copy data from srcConn to targetConn
		_, err := io.Copy(targetConn, srcConn)
		if err != nil {
			// Handle the error as needed
			fmt.Println("Error copying data to targetConn:", err)
		}
	}()

	go func() {
		// Copy data from targetConn to srcConn
		_, err := io.Copy(srcConn, targetConn)
		if err != nil {
			// Handle the error as needed
			fmt.Println("Error copying data to srcConn:", err)
		}
	}()

	// Block until both goroutines complete
	select {}
}
