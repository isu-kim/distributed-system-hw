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
)

// getNoteAll is for [GET] /note API
func getNoteAll(c *gin.Context) {
	log.Printf("%s [REQUEST][%s] %s {}",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)
}

// getNoteSpecific is for [GET] /note/{0-9} API
func getNoteSpecific(c *gin.Context) {
	log.Printf("%s [REQUEST][%s] %s {} ",
		misc.ColoredClient, c.Request.Method, c.Request.RequestURI)
}

// postNote is for [POST] /note API
func postNote(c *gin.Context) {
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

// putNoteSpecific is for [PUT] /note/{0-9} API
func putNoteSpecific(c *gin.Context) {
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
func patchNoteSpecific(c *gin.Context) {
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
func deleteNoteSpecific(c *gin.Context) {
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
