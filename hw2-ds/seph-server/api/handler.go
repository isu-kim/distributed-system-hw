package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"seph/api/local"
	"seph/api/remote"
	"seph/ds"
	"strings"
)

// Handler represents a single API handler
type Handler struct {
	engine   *gin.Engine
	addr     string
	port     int
	syncMode int
	dsh      *ds.Handler
}

// New creates a new API handler
func New(addr string, port int, sync string, dsh *ds.Handler) *Handler {
	// Create a new gin engine and init API routes
	engine := gin.Default()

	// Parse sync type
	var syncMode int
	if strings.Contains(sync, "local-write") {
		syncMode = 1
	} else if strings.Contains(sync, "remote-write") {
		syncMode = 2
	}

	initRoutes(engine, syncMode)

	return &Handler{
		engine:   engine,
		addr:     addr,
		port:     port,
		syncMode: syncMode,
		dsh:      dsh,
	}
}

// initRoutes initializes all routes
func initRoutes(engine *gin.Engine, syncMode int) {
	// All APIs for /note
	engine.GET("/note", getNoteAll)
	engine.GET("/note/:id", getNoteSpecific)
	engine.POST("/note", postNote)
	engine.PUT("/note/:id", putNoteSpecific)
	engine.PATCH("/note/:id", patchNoteSpecific)
	engine.DELETE("/note/:id", deleteNoteSpecific)

	// Init routes accordingly
	if syncMode == 1 { // local-write
		engine.GET("/primary/:id", local.GetPrimarySpecific)
		engine.POST("/backup", local.UpdateBackup)
		engine.PUT("/backup", local.UpdateBackup)
		engine.PATCH("/backup", local.UpdateBackup)
		engine.DELETE("/backup/:id", local.DeleteBackup)
	} else if syncMode == 2 {
		engine.POST("/primary", remote.ForwardPrimary)
		engine.PUT("/primary", remote.ForwardPrimary)
		engine.PATCH("/primary", remote.ForwardPrimary)
		engine.DELETE("/primary/:id", remote.DeletePrimary)
		engine.POST("/backup", remote.UpdateBackup)
		engine.PUT("/backup", remote.UpdateBackup)
		engine.PATCH("/backup", remote.UpdateBackup)
		engine.DELETE("/backup/:id", remote.DeleteBackup)
	} // I am just too lazy to consider edge cases :b
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
