package misc

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// Config represents the config of running the distributed storage
type Config struct {
	ServicePort int      `json:"servicePort"`
	Sync        string   `json:"sync"`
	Replicas    []string `json:"replicas"`
}

// Parse parses the designated config file and returns the Config struct
func Parse(filePath string) (error, Config) {
	// Read YAML file
	yamlFile, err := os.ReadFile(filePath)
	if err != nil {
		log.Fatalf("[ERROR] Unable to read the config file %s: %v", filePath, err)
		return err, Config{}
	}

	// Parse config file as struct
	var config Config
	err = json.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("[ERROR] Unable to unmarshall the config file %s: %v", filePath, err)
		return err, Config{}
	}

	// Now try validating the config file
	err = config.isValid()
	if err != nil {
		log.Fatalf("[ERROR] Unable to load config file properly: %v", err)
		return err, Config{}
	}

	// Everything was correct
	return nil, config
}

// isValid returns if this config is valid or not
func (c Config) isValid() error {
	// First check if sync method was correct or not
	// Sync method only supports "local-write" or "remote-write"
	if !strings.Contains(c.Sync, "local-write") && !strings.Contains(c.Sync, "remote-write") {
		msg := fmt.Sprintf("invalid sync type: %s, supported sync types: \"local-write\" or \"remote-write\"",
			c.Sync)
		return errors.New(msg)
	}

	// Then check if service port is valid or not
	if c.ServicePort <= 0 || c.ServicePort > 65535 {
		msg := fmt.Sprintf("invalid service port %d, range must be 0-65535", c.ServicePort)
		return errors.New(msg)
	}

	// We are not going to check if replica's ip address and ports are correct or not
	return nil
}

// PrintConfig prints out the contents of the config
func (c Config) PrintConfig() {
	log.Printf("- ServicePort: %d", c.ServicePort)
	log.Printf("- SyncType: %s", c.Sync)

	// Construct replicas and print this out
	replicaMsg := ""
	for i, rep := range c.Replicas {
		msg := fmt.Sprintf("%d) %s ", i, rep)
		if i != len(c.Replicas)-1 {
			msg = msg + ", "
		}

		replicaMsg = replicaMsg + msg
	}

	log.Printf("- Replicas: %s", replicaMsg)
}
