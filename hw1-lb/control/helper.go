package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
)

const (
	cmdTypeRegister   = 1
	cmdTypeUnregister = 2
)

// envParseAddress parses environment variables and retrieves address to listen control server on
// - LB_LISTEN_ADDR: the IP address to listen control server on (defaults 0.0.0.0)
// - LB_LISTEN_PORT: the port number to listen control server on (defaults 8080)
func envParseAddress() string {
	// Retrieve listen address information from environment variable
	envAddr := os.Getenv("LB_LISTEN_ADDR")
	ipv4Addr, _, err := net.ParseCIDR(envAddr)
	if err != nil {
		log.Printf("Could not parse environment variable $LB_LISTEN_ADDR in IP address format: %v", err)
		log.Printf("Defaulting back to 0.0.0.0")
		ipv4Addr = net.IPv4(0, 0, 0, 0) // 0.0.0.0
	}

	// Retrieve port information from environment variable
	envPort := os.Getenv("LB_LISTEN_PORT")
	portVal, err := strconv.Atoi(envPort)
	if err != nil {
		log.Printf("Could not parse environment variable $LB_LISTEN_PORT as integer: %v", err)
		log.Printf("Defaulting back to 8080")
		portVal = 8080
	}

	// Return 0.0.0.0:8080 format IP listen address
	return fmt.Sprintf("%s:%d", ipv4Addr.String(), portVal)
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
	} else {
		return 0, nil
	}
}
