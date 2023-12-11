package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
)

// Handler represents a single API handler
type Handler struct {
	engine *gin.Engine
	addr   string
	port   int
}

// New creates a new API handler
func New(addr string, port int) *Handler {
	// Create a new gin engine and init API routes
	engine := gin.Default()
	initRoutes(engine)

	return &Handler{
		engine: engine,
		addr:   addr,
		port:   port,
	}
}

// initRoutes initializes all routes
func initRoutes(engine *gin.Engine) {
	// All APIs for /note
	engine.GET("/note", getNoteAll)
	engine.GET("/note/:id", getNoteSpecific)
	engine.POST("/note", postNote)
	engine.PUT("/note/:id", putNoteSpecific)
	engine.PATCH("/note/:id", patchNoteSpecific)
	engine.DELETE("/note/:id", deleteNoteSpecific)
}

// Run starts running the API server
// This function is blocking function
func (h *Handler) Run() error {
	addr := fmt.Sprintf("%s:%d", h.addr, h.port)
	log.Printf("Now starting API server in %s", addr)

	err := h.engine.Run(addr)
	if err != nil {
		log.Fatalf("Could not start API server in %s: %v", addr, err)
		return err
	}

	return nil
}
