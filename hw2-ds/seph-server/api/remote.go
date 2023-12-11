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
	"seph/common"
	"seph/misc"
	"strconv"
	"strings"
)

// remoteForwardPrimary is for [POST/PUT/PATCH] /primary API
func (h *Handler) remoteForwardPrimary(c *gin.Context) {
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
	err, newNote := h.performRemoteWrite(c, reqNote)
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

	// Till here, only the primary knows that a note was written
	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(newNote)
	if err != nil {
		errResponse := common.NoteErrorResponse{
			Msg:    err.Error(),
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   fmt.Sprintf("%v", newNote),
		}

		// Return bad request, user sent us bad thing!
		c.JSON(http.StatusInternalServerError, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// Now primary shall tell all replicas to update
	// For all replicas, update
	for i, replica := range h.replicas {
		if i == 0 { // Skip current replica
			continue
		}

		// Perform request accordingly
		var response *http.Response
		endpoint := fmt.Sprintf("http://%s/backup", replica)
		log.Printf("[DEBUG] propagating to %s", replica)
		if strings.Contains(c.Request.Method, "POST") { // Forward POST
			// Create POST Request
			response, err = http.Post(endpoint, "application/json", bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Error making POST request to replica %s: %v\n", endpoint, err)
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
		} else if strings.Contains(c.Request.Method, "PUT") ||
			strings.Contains(c.Request.Method, "PATCH") { // Forward PUT or PATCH

			// Create a new request accordingly
			request, err := http.NewRequest(c.Request.Method, endpoint, bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Error creating %s request: %v to replica %s", c.Request.Method, endpoint, err)
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

			request.Header.Set("Content-Type", "application/json")
			// Perform the request
			client := http.Client{}
			response, err = client.Do(request)
			if err != nil {
				log.Printf("Error making %s request to replica %s: %v\n", c.Request.Method, endpoint, err)
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
		} // I sincerely will ignore the exceptional cases

		// Check status code from the /backup API
		if response.StatusCode == http.StatusOK {
			continue
		} else {
			log.Printf("Error updating POST request to replica %s: %v\n", endpoint, err)
			errResponse := common.NoteErrorResponse{
				Msg:    fmt.Sprintf("replica %s failed to update", replica),
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

		response.Body.Close()
	}

	// Everything went on correct
	log.Printf("%s [REPLY] Forward request to primary", misc.ColoredReplica)
	c.JSON(http.StatusOK, newNote)
}

// remoteDeletePrimary is for [DELETE] /primary/{0-9} API
func (h *Handler) remoteDeletePrimary(c *gin.Context) {
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
	err = h.performRemoteDelete(id)
	response := struct {
		Msg string `json:"msg"`
	}{}
	if err != nil {
		response.Msg = "FAILED"
		c.JSON(http.StatusInternalServerError, response)
		return
	}

	// Till here, only primary knows that a note was deleted

	// Now primary shall tell all replicas to update
	// For all replicas, delete
	for i, replica := range h.replicas {
		if i == 0 { // Skip current replica
			continue
		}

		// Perform request accordingly
		endpoint := fmt.Sprintf("http://%s/backup/%d", replica, id)

		// Create a new request accordingly
		request, err := http.NewRequest("DELETE", endpoint, nil)
		if err != nil {
			log.Printf("Error creating %s request: %v to replica %s", c.Request.Method, endpoint, err)

			response.Msg = "FAILED"
			c.JSON(http.StatusInternalServerError, response)
			log.Printf("%s [REPLY][%s] %s %v",
				misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, response)
			return
		}
		request.Header.Set("Content-Type", "application/json")

		// Perform the request
		client := http.Client{}
		res, err := client.Do(request)
		if err != nil {
			log.Printf("Error making %s request to replica %s: %v\n", c.Request.Method, endpoint, err)

			response.Msg = "FAILED"
			c.JSON(http.StatusInternalServerError, response)
			log.Printf("%s [REPLY][%s] %s %v",
				misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, response)
			return
		}

		// Check status code from the /backup API
		if res.StatusCode == http.StatusOK {
			continue
		} else {
			log.Printf("Error upating DELETE request to replica %s: %v\n", endpoint, err)

			response.Msg = "FAILED"
			c.JSON(http.StatusInternalServerError, response)
			log.Printf("%s [REPLY][%s] %s %v",
				misc.ColoredReplica, c.Request.Method, c.Request.RequestURI, response)
			return
		}

		res.Body.Close()
	}

	// Yes, the update went on correctly!
	response.Msg = "OK"
	c.JSON(http.StatusOK, response)
	log.Printf("%s [REPLY] Forward request to primary", misc.ColoredReplica)
	return
}

// remoteUpdateBackup is for [POST/PUT/PATCH] /backup API
func (h *Handler) remoteUpdateBackup(c *gin.Context) {
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
	err, newNote := h.performRemoteWrite(c, reqNote)
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

// remoteDeleteBackup is for [DELETE] /backup/{0-9} API
func (h *Handler) remoteDeleteBackup(c *gin.Context) {
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
	err = h.performRemoteDelete(id)
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

// handleRemoteWrite handles remote write
func (h *Handler) handleRemoteWrite(c *gin.Context, note common.Note) (error, common.Note) {
	// If this was replica 0, skip forward
	if misc.IsReplica0() {
		return h.performRemoteWrite(c, note)
	} else { // If not, forward this request to the primary
		log.Printf("%s [REQUEST] Forward request to primary", misc.ColoredReplica)

		// Serialize the payload to JSON
		payloadBytes, err := json.Marshal(note)
		if err != nil {
			log.Println("Error marshaling JSON payload:", err)
			return err, common.Note{}
		}

		// Perform request accordingly
		var response *http.Response
		endpoint := fmt.Sprintf("http://%s/primary", h.replicas[0])
		if strings.Contains(c.Request.Method, "POST") { // Forward POST
			// Create POST Request
			response, err = http.Post(endpoint, "application/json", bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Error making POST request to %s: %v\n", endpoint, err)
				return err, common.Note{}
			}
		} else if strings.Contains(c.Request.Method, "PUT") ||
			strings.Contains(c.Request.Method, "PATCH") { // Forward PUT or PATCH

			// Create a new request accordingly
			request, err := http.NewRequest(c.Request.Method, endpoint, bytes.NewBuffer(payloadBytes))
			if err != nil {
				log.Printf("Error creating %s request: %v to primary\n", c.Request.Method, err)
				return err, common.Note{}
			}
			request.Header.Set("Content-Type", "application/json")

			// Perform the request
			client := http.Client{}
			response, err = client.Do(request)
			if err != nil {
				log.Printf("Error making %s request to primary %s: %v\n", c.Request.Method, endpoint, err)
				return err, common.Note{}
			}
		} // I sincerely will ignore the exceptional cases

		defer response.Body.Close()

		// Check if the response status code is OK (200)
		if response.StatusCode == http.StatusOK {
			// Read the response body
			body, err := io.ReadAll(response.Body)
			if err != nil {
				log.Println("Error reading response body from primary:", err)
				return err, common.Note{}
			}

			// Try unmarshalling the body into our note format
			var newNote common.Note
			err = json.Unmarshal(body, &newNote)
			if err != nil {
				log.Println("Error unmarshalling response from primary:", err)
				return err, common.Note{}
			}

			// Yes this worked
			return nil, newNote
		} else {
			log.Println("Error: Non-OK response status code from primary:", response.Status)
			return errors.New("non-ok response code"), common.Note{}
		}
	}
}

// performRemoteWrite actually performs the remote write, this will create the file as well
func (h *Handler) performRemoteWrite(c *gin.Context, note common.Note) (error, common.Note) {
	if strings.Contains(c.Request.Method, "POST") { // If this was POST, create new one
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

// handleRemoteDelete handles the delete operation
func (h *Handler) handleRemoteDelete(id int) error {
	// If this was replica 0, skip forward
	if misc.IsReplica0() {
		return h.performRemoteDelete(id)
	} else { // If not, forward this request to the primary
		log.Printf("%s [REQUEST] Forward request to primary", misc.ColoredReplica)

		// Perform delete request
		var response *http.Response
		endpoint := fmt.Sprintf("http://%s/primary/%d", h.replicas[0], id)

		// Create a new request accordingly
		request, err := http.NewRequest("DELETE", endpoint, nil)
		if err != nil {
			log.Printf("Error creating DELETE request to primary: %v\n", err)
			return err
		}
		request.Header.Set("Content-Type", "application/json")

		// Perform the request
		client := http.Client{}
		response, err = client.Do(request)
		if err != nil {
			log.Printf("Error making DELETE request to primary %s: %v\n", endpoint, err)
			return err
		}

		// Just check if code was OK
		if response.StatusCode == http.StatusOK {
			return nil
		} else {
			return err
		}
	}
}

// performRemoteDelete removes the file from current local storage
func (h *Handler) performRemoteDelete(id int) error {
	return h.dsh.DeleteNote(id)
}
