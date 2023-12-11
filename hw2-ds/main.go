package main

import (
	"hw2-ds/misc"
	"log"
	"os"
)

var config misc.Config

// main is the entry point of this program
func main() {
	// Check if a command-line argument is provided
	if len(os.Args) < 2 {
		log.Printf("Usage: ./%s <config.json>", os.Args[0])
		return
	}

	// Get the YAML file from the command-line argument
	configFile := os.Args[1]
	var err error

	// Parse config file
	err, config = misc.Parse(configFile)
	if err != nil {
		return
	}

	log.Printf("Loaded config file successfully: ")
	config.PrintConfig()
}
