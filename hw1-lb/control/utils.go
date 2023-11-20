package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"lb/common"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
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
