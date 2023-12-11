package main

import (
	"log"
	"os"
	"seph/api"
	"seph/misc"
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

	// Print logo and initialize colors!
	misc.PrintLogo()
	misc.InitColoredLogs()

	log.Printf("Loaded config file successfully: ")
	config.PrintConfig()

	// Start up the API server
	h := api.New("0.0.0.0", config.ServicePort)
	err = h.Run()
	if err != nil {
		return
	}
}
