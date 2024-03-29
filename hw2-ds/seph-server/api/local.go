package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"log"
	"net/http"
	"os"
	"seph/common"
	"seph/misc"
	"strconv"
	"strings"
)

// localGetPrimarySpecific is for [GET] /primary/{0-9} API
func (h *Handler) localGetPrimarySpecific(c *gin.Context) {
	// Read ID param from API
	noteID := c.Param("id")
	newPrimary := c.GetHeader("primary")

	// ID was unable to be converted as an integer
	note, err := strconv.Atoi(noteID)
	if err != nil {
		errResponse := common.NoteErrorResponse{
			Msg:    "wrong URI, ID was invalid",
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "",
		}

		c.JSON(http.StatusBadRequest, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// Check if the note already had a primary before
	oldPrimary, ok := h.primaryMap[note]

	// Just show user that we are moving primaries
	if ok { // This means that there already was a primary before
		log.Printf("%s [REQUEST] Move item to new primary %s->%s", misc.ColoredReplica, oldPrimary, newPrimary)
	} else { // This means that this was a new note
		log.Printf("%s [REQUEST] Move item to new primary %s", misc.ColoredReplica, newPrimary)
	}

	// Construct response
	response := struct {
		Msg string `json:"msg"`
	}{Msg: "OK"}
	c.JSON(http.StatusOK, response)
	return
}

// localUpdateBackup is for [POST/PUT/PATCH] /backup API
func (h *Handler) localUpdateBackup(c *gin.Context) {
	// Try parsing body JSON
	err, reqNote := clientRequest(c)
	if err != nil {
		errResponse := common.NoteErrorResponse{
			Msg:    err.Error(),
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "", // When json parsing failed, we regard body as empty
		}

		// Return bad request, user sent us bad thing!
		c.JSON(http.StatusBadRequest, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// Perform remote write
	err, newNote := h.performLocalWrite(c, reqNote)
	if err != nil {
		errResponse := common.NoteErrorResponse{
			Msg:    err.Error(),
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   fmt.Sprintf("%v", reqNote),
		}

		// Return bad request, user sent us bad thing!
		c.JSON(http.StatusInternalServerError, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// This replica's update was successful
	c.JSON(http.StatusOK, newNote)
	return
}

// localDeleteBackup is for [DELETE] /backup/{0-9} API
func (h *Handler) localDeleteBackup(c *gin.Context) {
	log.Printf("%s [REQUEST][%s] %s {} ",
		misc.ColoredReplica, c.Request.Method, c.Request.RequestURI)

	// Read ID param from API
	noteID := c.Param("id")

	// ID was unable to be converted as an integer
	id, err := strconv.Atoi(noteID)
	if err != nil {
		errResponse := common.NoteErrorResponse{
			Msg:    "wrong URI, ID was invalid",
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "",
		}

		c.JSON(http.StatusBadRequest, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// Perform note delete
	err = h.dsh.DeleteNote(id)
	response := struct {
		Msg string `json:"msg"`
	}{}
	if err != nil {
		response.Msg = "FAILED"
		c.JSON(http.StatusInternalServerError, response)
		return
	} else {
		response.Msg = "OK"
		c.JSON(http.StatusOK, response)
		return
	}
}

// handleLocalWrite handles local write
func (h *Handler) handleLocalWrite(c *gin.Context, note common.Note, primary string) (error, common.Note) {
	// Check if the note already had a primary before
	oldPrimary, ok := h.primaryMap[note.Id]
	newPrimary := os.Getenv("REPLICA_ID")

	// Just show user that we are moving primaries
	if ok { // This means that there already was a primary before
		log.Printf("%s [REQUEST] Move item to new primary %s->%s", misc.ColoredReplica, oldPrimary, newPrimary)
	} else { // This means that this was a new note
		log.Printf("%s [REQUEST] Move item to new primary %s", misc.ColoredReplica, newPrimary)
	}

	// If this was a POST request, create a new file and propagate
	if strings.Contains(c.Request.Method, "POST") {
		// Assign new ID for the new note
		err, newID := h.dsh.AssignNewID()
		if err != nil {
			log.Printf("Unable to assign new ID for note: %v", err)
			return err, common.Note{}
		}

		// Update note and try creating the note
		note.Id = newID
		err = h.dsh.CreateNote(note)
		if err != nil {
			log.Printf("Unable to create new note: %v", err)
			return err, common.Note{}
		}

		// Update primary map, this is the current primary
		h.primaryMap[newID] = primary
		c.JSON(http.StatusOK, note) // We just return write response

		// Till here, only the new primary knows the current note's information
		// Serialize the payload to JSON
		payloadBytes, err := json.Marshal(note)
		if err != nil {
			log.Println("Error marshaling JSON payload:", err)
			return err, common.Note{}
		}

		// For all replicas, tell them to update
		for _, replica := range h.replicas {
			// If this was the primary, skip
			if strings.Contains(replica, primary) {
				continue
			}

			log.Printf("[DEBUG] propagating to %s", replica)

			backupEndpoint := fmt.Sprintf("http://%s/backup", replica)                     // For backup
			primaryUpdateEndpoint := fmt.Sprintf("http://%s/primary/%d", replica, note.Id) // For keeping track of primary

			// Perform backup API
			response, err := http.Post(backupEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Error making POST request to replica %s: %v\n", backupEndpoint, err)
				return err, common.Note{}
			}

			// Check if the replica got response correct
			if response.StatusCode == http.StatusOK {
				// Read the response body
				body, err := io.ReadAll(response.Body)
				if err != nil {
					log.Printf("Error reading response body from replica %s: %v", backupEndpoint, err)
					return err, common.Note{}
				}

				// Try unmarshalling the body into our note format
				var newNote common.Note
				err = json.Unmarshal(body, &newNote)
				if err != nil {
					log.Printf("Error unmarshalling response from replica %s: %v", backupEndpoint, err)
					return err, common.Note{}
				}
			} else {
				log.Printf("Non-OK response from replica %s: %v", backupEndpoint, err)
				return err, common.Note{}
			}
			response.Body.Close()

			// Perform primary API
			request, err := http.NewRequest("GET", primaryUpdateEndpoint, nil)
			if err != nil {
				log.Printf("Error creating GET request to replica %s: %v", replica, err)
				return err, common.Note{}
			}

			// Add primary value to header
			request.Header.Add("primary", primary)

			// perform GET request
			client := &http.Client{}
			response, err = client.Do(request)
			if err != nil {
				log.Printf("Error making GET request to replica %s: %v", replica, err)
			}

			if response.StatusCode == http.StatusOK {
				// Read the response body
				body, err := io.ReadAll(response.Body)
				if err != nil {
					log.Printf("Error reading response body from replica %s: %v", primaryUpdateEndpoint, err)
					return err, common.Note{}
				}

				// Try unmarshalling the body into our note format
				var newNote common.Note
				err = json.Unmarshal(body, &newNote)
				if err != nil {
					log.Printf("Error unmarshalling response from replica %s: %v", primaryUpdateEndpoint, err)
					return err, common.Note{}
				}
			} else {
				log.Printf("Non-OK response from replica %s: %v", primaryUpdateEndpoint, err)
				return err, common.Note{}
			}
			response.Body.Close()
		}

		// We successfully wrote, so return id and the note!
		return nil, note
	} else if strings.Contains(c.Request.Method, "PUT") ||
		strings.Contains(c.Request.Method, "PATCH") { // Forward PUT or PATCH

		err, newNote := h.performLocalWrite(c, note)
		if err != nil {
			log.Printf("Unable to write local file: %v", err)
			return err, common.Note{}
		}

		// This replica's update was successful
		c.JSON(http.StatusOK, newNote)

		// Serialize the payload to JSON
		payloadBytes, err := json.Marshal(note)
		if err != nil {
			log.Println("Error marshaling JSON payload:", err)
			return err, common.Note{}
		}

		// For all replicas, tell them to update as well
		for _, replica := range h.replicas {
			// If this was the primary, skip
			if strings.Contains(replica, primary) {
				continue
			}

			log.Printf("[DEBUG] propagating to %s", replica)

			backupEndpoint := fmt.Sprintf("http://%s/backup", replica)                     // For backup
			primaryUpdateEndpoint := fmt.Sprintf("http://%s/primary/%d", replica, note.Id) // For keeping track of primary

			// Perform backup API
			// Create a new request accordingly
			request, err := http.NewRequest(c.Request.Method, backupEndpoint, bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Error creating %s request: %v to primary\n", c.Request.Method, err)
				return err, common.Note{}
			}
			request.Header.Set("Content-Type", "application/json")

			// Perform the request
			client := &http.Client{}
			response, err := client.Do(request)
			if err != nil {
				log.Printf("Error making %s request to primary %s: %v\n", c.Request.Method, backupEndpoint, err)
				return err, common.Note{}
			}

			// Check if the replica got response correct
			if response.StatusCode == http.StatusOK {
				// Read the response body
				body, err := io.ReadAll(response.Body)
				if err != nil {
					log.Printf("Error reading response body from replica %s: %v", backupEndpoint, err)
					return err, common.Note{}
				}

				// Try unmarshalling the body into our note format
				var newNote common.Note
				err = json.Unmarshal(body, &newNote)
				if err != nil {
					log.Printf("Error unmarshalling response from replica %s: %v", backupEndpoint, err)
					return err, common.Note{}
				}
			} else {
				log.Printf("Non-OK response from replica %s: %v", backupEndpoint, err)
				return err, common.Note{}
			}
			response.Body.Close()

			// Perform primary API
			request, err = http.NewRequest("GET", primaryUpdateEndpoint, nil)
			if err != nil {
				log.Printf("Error creating GET request to replica %s: %v", replica, err)
				return err, common.Note{}
			}

			// Add primary value to header
			request.Header.Add("primary", primary)

			// perform GET request
			client = &http.Client{}
			response, err = client.Do(request)
			if err != nil {
				log.Printf("Error making GET request to replica %s: %v", replica, err)
			}

			if response.StatusCode == http.StatusOK {
				// Read the response body
				body, err := io.ReadAll(response.Body)
				if err != nil {
					log.Printf("Error reading response body from replica %s: %v", primaryUpdateEndpoint, err)
					return err, common.Note{}
				}

				// Try unmarshalling the body into our note format
				var newNote common.Note
				err = json.Unmarshal(body, &newNote)
				if err != nil {
					log.Printf("Error unmarshalling response from replica %s: %v", primaryUpdateEndpoint, err)
					return err, common.Note{}
				}
			} else {
				log.Printf("Non-OK response from replica %s: %v", primaryUpdateEndpoint, err)
				return err, common.Note{}
			}
			response.Body.Close()
		}
		// We successfully wrote, so return id and the note!
		return nil, note
	} // I won't give a fuck on other cases

	return errors.New("invalid method"), common.Note{}
}

// performLocalWrite performs local write
func (h *Handler) performLocalWrite(c *gin.Context, note common.Note) (error, common.Note) {
	if strings.Contains(c.Request.Method, "POST") { // If this was POST, create new one
		// Update note and try creating the note
		err := h.dsh.CreateNote(note)
		if err != nil {
			log.Printf("Unable to create new note: %v", err)
			return err, common.Note{}
		}

		// We successfully wrote, so return id and the note!
		return nil, note
	} else if strings.Contains(c.Request.Method, "PATCH") {
		// First, find original note
		err, original := h.dsh.ReadSpecific(note.Id)
		if err != nil {
			log.Printf("Unable to find existing note with ID  %d: %v", note.Id)
			return err, common.Note{}
		}

		// Patch will only modify if the field was not empty
		if len(note.Body) != 0 {
			original.Body = note.Body
		}

		if len(note.Title) != 0 {
			original.Title = note.Title
		}

		// Try updating the note
		err = h.dsh.UpdateNote(original)
		if err != nil {
			log.Printf("Unable to update existing note with ID %d: %v", note.Id, err)
			return err, common.Note{}
		}

		return nil, original
	} else if strings.Contains(c.Request.Method, "PUT") {
		// First, find original note
		err, original := h.dsh.ReadSpecific(note.Id)
		if err != nil {
			log.Printf("Unable to find existing note with ID  %d: %v", note.Id)
			return err, common.Note{}
		}

		// Put will just overwrite
		original.Body = note.Body
		original.Title = note.Title

		// Try updating the note
		err = h.dsh.UpdateNote(original)
		if err != nil {
			log.Printf("Unable to update existing note with ID %d: %v", note.Id, err)
			return err, common.Note{}
		}

		return nil, original
	} else {
		return errors.New("unknown method"), common.Note{}
	}
}

// handleLocalDelete handles the delete operation
func (h *Handler) handleLocalDelete(id int, primary string) error {
	// Check if the note already had a primary before
	oldPrimary, ok := h.primaryMap[id]
	newPrimary := os.Getenv("REPLICA_ID")

	// Just show user that we are moving primaries
	if ok { // This means that there already was a primary before
		log.Printf("%s [REQUEST] Move item to new primary %s->%s", misc.ColoredReplica, oldPrimary, newPrimary)
	} else { // This means that this was a new note
		log.Printf("%s [REQUEST] Move item to new primary %s", misc.ColoredReplica, newPrimary)
	}

	// First, delete the file from current replica
	err := h.dsh.DeleteNote(id)
	if err != nil {
		log.Printf("Error deleting note %d: %v", id, err)
		return err
	}

	// For all replicas, tell them to update as well
	for _, replica := range h.replicas {
		// If this was the primary, skip
		if strings.Contains(replica, primary) {
			continue
		}

		// Perform delete request
		var response *http.Response
		endpoint := fmt.Sprintf("http://%s/backup/%d", replica, id)

		// Create a new request accordingly
		request, err := http.NewRequest("DELETE", endpoint, nil)
		if err != nil {
			log.Printf("Error creating DELETE request to replica: %v\n", err)
			return err
		}
		request.Header.Set("Content-Type", "application/json")

		// Perform the request
		client := http.Client{}
		response, err = client.Do(request)
		if err != nil {
			log.Printf("Error making DELETE request to replica %s: %v\n", endpoint, err)
			return err
		}

		// Just check if code was OK
		if response.StatusCode == http.StatusOK {
			continue
		} else {
			return err
		}
	}

	return nil
}
