package api

import (
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
)

// getNoteAll is for [GET] /note API
func (h *Handler) getNoteAll(c *gin.Context) {
	log.Printf("%s [REQUEST][%s] %s {}",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)

	// Read all notes from local storage, send them
	notes := h.dsh.ReadAll()
	c.JSON(http.StatusOK, notes)
	log.Printf("%s [REPLY][%s] %s %v",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI, notes)
}

// getNoteSpecific is for [GET] /note/{0-9} API
func (h *Handler) getNoteSpecific(c *gin.Context) {
	log.Printf("%s [REQUEST][%s] %s {} ",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)

	// Read ID param from API
	noteID := c.Param("id")

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
			misc.ColoredClient, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// If ID was able to be converted, try reading it from local storage
	err, notes := h.dsh.ReadSpecific(note)
	if err != nil {
		errResponse := common.NoteErrorResponse{
			Msg:    "wrong URI, non existing ID",
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "",
		}

		c.JSON(http.StatusBadRequest, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredClient, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	}

	// This worked, so return note information
	c.JSON(http.StatusOK, notes)
	log.Printf("%s [REPLY][%s] %s %v",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI, notes)
}

// postNote is for [POST] /note API
func (h *Handler) postNote(c *gin.Context) {
	msg := fmt.Sprintf("%s [REQUEST][%s] %s ",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)

	// Try parsing body JSON
	err, req := clientRequest(c)
	if err != nil {
		msg = msg + "{err}"
		errResponse := common.NoteErrorResponse{
			Msg:    err.Error(),
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "", // When json parsing failed, we regard body as empty
		}

		// Return bad request, user sent us bad thing!
		log.Println(msg)
		c.JSON(http.StatusBadRequest, errResponse)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredClient, c.Request.Method, c.Request.RequestURI, errResponse)
		return
	} else {
		msg = fmt.Sprintf("%s %v", msg, req)
	}

	// Print out the request information
	log.Println(msg)

	// Now the distributed storage part!
	switch h.syncMode {
	case misc.SyncLocalWrite:
		// @todo
		break
	case misc.SyncRemoteWrite:
		err, result := h.handleRemoteWrite(c, req)
		if err != nil {
			errResponse := common.NoteErrorResponse{
				Msg:    err.Error(),
				Method: c.Request.Method,
				Uri:    c.Request.RequestURI,
				Body:   fmt.Sprintf("%v", req),
			}
			c.JSON(http.StatusInternalServerError, errResponse)
			log.Printf("%s [REPLY][%s] %s %v",
				misc.ColoredClient, c.Request.Method, c.Request.RequestURI, errResponse)
			return
		}

		// Yes this worked
		c.JSON(http.StatusOK, result)
		log.Printf("%s [REPLY][%s] %s %v",
			misc.ColoredClient, c.Request.Method, c.Request.RequestURI, result)
		return
	}
}

// putNoteSpecific is for [PUT] /note/{0-9} API
func (h *Handler) putNoteSpecific(c *gin.Context) {
	msg := fmt.Sprintf("%s [REQUEST][%s] %s ",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)

	// Try parsing body JSON
	err, req := clientRequest(c)
	if err != nil {
		msg = msg + "{err}"
		errResponse := common.NoteErrorResponse{
			Msg:    err.Error(),
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "", // When json parsing failed, we regard body as empty
		}

		// Return bad request, user sent us bad thing!
		c.JSON(http.StatusBadRequest, errResponse)
	} else {
		msg = fmt.Sprintf("%s %v", msg, req)
	}

	// Print out the request information
	log.Println(msg)
}

// patchNoteSpecific is for [PATCH] /note/{0-9} API
func (h *Handler) patchNoteSpecific(c *gin.Context) {
	msg := fmt.Sprintf("%s [REQUEST][%s] %s ",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)

	// Try parsing body JSON
	err, req := clientRequest(c)
	if err != nil {
		msg = msg + "{err}"
		errResponse := common.NoteErrorResponse{
			Msg:    err.Error(),
			Method: c.Request.Method,
			Uri:    c.Request.RequestURI,
			Body:   "", // When json parsing failed, we regard body as empty
		}

		// Return bad request, user sent us bad thing!
		c.JSON(http.StatusBadRequest, errResponse)
	} else {
		msg = fmt.Sprintf("%s %v", msg, req)
	}

	// Print out the request information
	log.Println(msg)
}

// deleteNoteSpecific is for [DELETE] /note/{0-9} API
func (h *Handler) deleteNoteSpecific(c *gin.Context) {
	log.Printf("%s [REQUEST][%s] %s {} ",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)
}

// clientRequest prints out the client's request as format mentioned
// also this will return the JSON which was in the body
func clientRequest(c *gin.Context) (error, common.Note) {
	// Read the request body
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		msg := fmt.Sprintf("invalid request body: %v", err)
		return errors.New(msg), common.Note{}
	}

	// Unmarshal the JSON body
	var noteRequest common.Note
	err = json.Unmarshal(body, &noteRequest)
	if err != nil {
		msg := fmt.Sprintf("error unmarshalling JSON body: %v", err)
		return errors.New(msg), common.Note{}
	}

	return nil, noteRequest
}
