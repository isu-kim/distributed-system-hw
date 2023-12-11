package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"seph-client/misc"
)

var config misc.Config

// loop is the main loop for keep asking the user about their actions
func loop() {
	// Ask user for target replica
	err, target := askReplica()
	if err != nil {
		return
	}

	// Ask user for action
	err, action := askAction()
	if err != nil {
		return
	}

	// Perform action
	targetReplica := config.Replicas[target]
	perform(targetReplica, action)
}

// askReplica asks the user to designate target replica
func askReplica() (error, int) {
	// Ask user to select target replica
	fmt.Println("Select Replica:")
	config.PrintReplica()

	var userInput int
	fmt.Printf(">> ")
	_, err := fmt.Scan(&userInput)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return err, -1
	}

	// Check if replica input was valid
	if userInput < 0 || userInput > len(config.Replicas) {
		fmt.Println("Invalid replica index")
		return errors.New("invalid index"), -1
	}
	return nil, userInput
}

// askAction asks user to designate the action to take
func askAction() (error, int) {
	fmt.Println("Available Actions:")
	fmt.Println("1) [GET] /note - List all notes")
	fmt.Println("2) [GET] /note/:id - Get specific note")
	fmt.Println("3) [POST] /note - Add a new note")
	fmt.Println("4) [PUT] /note/:id - Overwrite a specific note")
	fmt.Println("5) [PATCH] /note/:id - Patch a specific note")
	fmt.Println("6) [DELETE] /note/:id - Delete a specific note")

	var userInput int
	fmt.Printf(">> ")
	_, err := fmt.Scan(&userInput)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return err, -1
	}

	// Check if replica input was valid
	if userInput < 0 || userInput > 6 {
		fmt.Println("Invalid replica index")
		return errors.New("invalid index"), -1
	}
	return nil, userInput
}

// perform performs the action to target replica
func perform(target string, action int) {
	switch action {
	case 1:
		performGetNote(target)
		break
	case 2:
		performGetNoteSpecific(target)
		break
	case 3:
		performPostNote(target)
		break
	case 4:
		performPutNote(target)
		break
	case 5:
		performPatchNote(target)
		break
	case 6:
		performDeleteNoteSpecific(target)
		break
	}
}

// performGetNote will perform [GET] /note
func performGetNote(target string) {
	endpoint := fmt.Sprintf("%s/note", target)
	response, err := http.Get(endpoint)
	if err != nil {
		fmt.Printf("Error making GET request to %s: %v\n", endpoint, response)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode == http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// Print the response body
		fmt.Println("Response Body:")
		fmt.Println(string(body))
	} else {
		fmt.Println("Error: Non-OK response status code:", response.Status)
	}
}

// performGetNoteSpecific will perform [GET] /note/:id
func performGetNoteSpecific(target string) {
	fmt.Println("Input target note: ")
	var userInput int
	fmt.Printf(">> ")
	_, err := fmt.Scan(&userInput)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Check if replica input was valid
	if userInput < 0 {
		fmt.Println("Invalid note index")
		return
	}

	endpoint := fmt.Sprintf("%s/note/%d", target, userInput)
	response, err := http.Get(endpoint)
	if err != nil {
		fmt.Printf("Error making GET request to %s: %v\n", endpoint, response)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode == http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// Print the response body
		fmt.Println("Response Body:")
		fmt.Println(string(body))
	} else {
		fmt.Println("Error: Non-OK response status code:", response.Status)
	}
}

// performPostNote will perform [POST] /note
func performPostNote(target string) {
	// Ask title
	fmt.Println("Input title: ")
	var title string
	fmt.Printf(">> ")
	_, err := fmt.Scan(&title)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Ask body
	fmt.Println("Input body: ")
	var body string
	fmt.Printf(">> ")
	_, err = fmt.Scan(&body)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Construct simple payload
	payload := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{
		Title: title,
		Body:  body,
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling JSON payload:", err)
		return
	}

	// Perform POST request
	endpoint := fmt.Sprintf("%s/note", target)
	response, err := http.Post(endpoint, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Error making POST request to %s: %v\n", endpoint, err)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode == http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// Print the response body
		fmt.Println("Response Body:")
		fmt.Println(string(body))
	} else {
		fmt.Println("Error: Non-OK response status code:", response.Status)
	}
}

// performPutNote will perform [PUT] /note
func performPutNote(target string) {
	// Ask target note
	fmt.Println("Input target note: ")
	var userInput int
	fmt.Printf(">> ")
	_, err := fmt.Scan(&userInput)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Check if replica input was valid
	if userInput < 0 {
		fmt.Println("Invalid note index")
		return
	}

	// Ask title
	fmt.Println("Input title: ")
	var title string
	fmt.Printf(">> ")
	_, err = fmt.Scan(&title)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Ask body
	fmt.Println("Input body: ")
	var body string
	fmt.Printf(">> ")
	_, err = fmt.Scan(&body)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Construct simple payload
	payload := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{
		Title: title,
		Body:  body,
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling JSON payload:", err)
		return
	}

	// Create a PUT request
	endpoint := fmt.Sprintf("%s/note/%d", target, userInput)
	request, err := http.NewRequest("PUT", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Error creating PUT request: %v\n", err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	// Perform the PUT request
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Error making PUT request to %s: %v\n", endpoint, err)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode == http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// Print the response body
		fmt.Println("Response Body:")
		fmt.Println(string(body))
	} else {
		fmt.Println("Error: Non-OK response status code:", response.Status)
	}
}

// performPatchNote will perform [PATCH] /note
func performPatchNote(target string) {
	// Ask target note
	fmt.Println("Input target note: ")
	var userInput int
	fmt.Printf(">> ")
	_, err := fmt.Scan(&userInput)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Check if replica input was valid
	if userInput < 0 {
		fmt.Println("Invalid note index")
		return
	}

	// Ask title
	fmt.Println("Input title: ")
	var title string
	fmt.Printf(">> ")
	_, err = fmt.Scan(&title)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Ask body
	fmt.Println("Input body: ")
	var body string
	fmt.Printf(">> ")
	_, err = fmt.Scan(&body)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Construct simple payload
	payload := struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}{
		Title: title,
		Body:  body,
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling JSON payload:", err)
		return
	}

	// Create a PUT request
	endpoint := fmt.Sprintf("%s/note/%d", target, userInput)
	request, err := http.NewRequest("PATCH", endpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("Error creating PATCH request: %v\n", err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	// Perform the PUT request
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Error making PATCH request to %s: %v\n", endpoint, err)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode == http.StatusOK {
		// Read the response body
		body, err := io.ReadAll(response.Body)
		if err != nil {
			fmt.Println("Error reading response body:", err)
			return
		}

		// Print the response body
		fmt.Println("Response Body:")
		fmt.Println(string(body))
	} else {
		fmt.Println("Error: Non-OK response status code:", response.Status)
	}
}

// performDeleteNoteSpecific will perform [DELETE] /note/:id
func performDeleteNoteSpecific(target string) {
	fmt.Println("Input target note: ")
	var userInput int
	fmt.Printf(">> ")
	_, err := fmt.Scan(&userInput)

	if err != nil {
		fmt.Println("Error reading input:", err)
		return
	}

	// Check if replica input was valid
	if userInput < 0 {
		fmt.Println("Invalid note index")
		return
	}

	endpoint := fmt.Sprintf("%s/note/%d", target, userInput)
	request, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		fmt.Printf("Error creating DELETE request: %v\n", err)
		return
	}

	// Perform the DELETE request
	client := http.Client{}
	response, err := client.Do(request)
	if err != nil {
		fmt.Printf("Error making DELETE request to %s: %v\n", endpoint, err)
		return
	}
	defer response.Body.Close()

	// Check if the response status code is OK (200)
	if response.StatusCode == http.StatusOK {
		fmt.Println("Note deleted successfully")
	} else {
		fmt.Println("Error: Non-OK response status code:", response.Status)
	}
}

// printLogo prints out the logo!
func printLogo() {
	fmt.Println("                    __  ")
	fmt.Println("   ________  ____  / /_ ")
	fmt.Println("  / ___/ _ \\/ __ \\/ __ \\")
	fmt.Println(" (__  )  __/ /_/ / / / /")
	fmt.Println("/____/\\___/ .___/_/ /_/ ")
	fmt.Println("         /_/            ")
	fmt.Println("Simple Distributed Storage")
	fmt.Println("        32190984 - Isu Kim")
}

// main is the entry point of this program
func main() {
	// Check if a command-line argument is provided
	if len(os.Args) < 2 {
		fmt.Printf("Usage: ./%s <config.json>", os.Args[0])
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

	fmt.Printf("Loaded config file successfully: \n")
	printLogo()

	for {
		loop()
	}
}
