package main

import (
	"log"
	"os"
	"seph/api"
	"seph/ds"
	"seph/misc"
)

var Config misc.Config

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
	err, Config = misc.Parse(configFile)
	if err != nil {
		return
	}

	// Print logo and initialize colors!
	misc.PrintLogo()
	misc.InitColoredLogs()

	log.Printf("Loaded config file successfully: ")
	Config.PrintConfig()

	// Fire up distributed storage handler
	// I am literally too lazy to set this as an env variable, I will just hard code this :b
	dsh := ds.New("./data")

	// Start up the API server
	h := api.New("0.0.0.0", Config.ServicePort, Config.Sync, dsh)
	err = h.Run()
	if err != nil {
		return
	}
}
