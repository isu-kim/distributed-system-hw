package ds

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"seph/common"
	"sync"
)

// Handler represents a single distributed storage handler
type Handler struct {
	lock       sync.Mutex
	targetDir  string
	primaryMap map[int]string
	replicas   []string
}

// New creates a new Handler
func New(targetDir string, replicas []string) *Handler {
	return &Handler{
		lock:       sync.Mutex{},
		targetDir:  targetDir,
		primaryMap: make(map[int]string),
		replicas:   replicas,
	}
}

// Init will try retrieving all data from replicas[0]
// This is one of the requirements in the homework!
// One of the problem is that, if this seph-server was the replicas[0],
// this will literally just do a GET request to the self.
// But, I am too lazy to cover this :b
func (h *Handler) Init() error {
	// If this was replicas[0], skip
	if len(os.Getenv("IS_REPLICA_0")) != 0 {
		log.Printf("[Seph] Initialization skipped, this was replicas[0]")
		return nil
	} else { // If this was not replicas[0], then get data from replicas[0]
		endpoint := fmt.Sprintf("http://%s/note", h.replicas[0])
		response, err := http.Get(endpoint)
		if err != nil {
			log.Printf("[Seph] Error making GET request to %s: %v", endpoint, err)
			return err
		}
		defer response.Body.Close()

		// Slice of common.Note which comes from replica[0]/note
		var notes []common.Note

		// Check if the response status code is OK (200)
		if response.StatusCode == http.StatusOK {
			// Decode the response body into a slice of common.Note
			err := json.NewDecoder(response.Body).Decode(&notes)
			if err != nil {
				log.Println("Error decoding response body:", err)
				return err
			}
		} else {
			log.Println("Error: Non-OK response status code:", response.Status)
			return errors.New("non-ok response status code")
		}

		// Dump all notes into the local storage
		h.DumpNotes(notes)

		log.Printf("[Seph] Initialization finished")
		return nil
	}
}
