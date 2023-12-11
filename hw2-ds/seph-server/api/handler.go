package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"log"
	"os"
	"seph/ds"
	"seph/misc"
	"strings"
	"time"
)

// Handler represents a single API handler
type Handler struct {
	engine     *gin.Engine
	addr       string
	port       int
	syncMode   int
	dsh        *ds.Handler
	replicas   []string
	primaryMap map[int]string
}

// New creates a new API handler
func New(addr string, port int, sync string, replicas []string) *Handler {
	// Create a new gin engine and init API routes
	engine := gin.Default()

	// Parse sync type
	var syncMode int
	if strings.Contains(sync, "local-write") {
		syncMode = misc.SyncLocalWrite
	} else if strings.Contains(sync, "remote-write") {
		syncMode = misc.SyncRemoteWrite
	}

	// Create handler and init routes
	h := Handler{
		engine:     engine,
		addr:       addr,
		port:       port,
		syncMode:   syncMode,
		dsh:        nil,
		replicas:   replicas,
		primaryMap: map[int]string{},
	}
	h.initRoutes()

	return &h
}

// initRoutes initializes all routes
func (h *Handler) initRoutes() {
	// All APIs for /note
	h.engine.GET("/note", h.getNoteAll)
	h.engine.GET("/note/:id", h.getNoteSpecific)
	h.engine.POST("/note", h.postNote)
	h.engine.PUT("/note/:id", h.putNoteSpecific)
	h.engine.PATCH("/note/:id", h.patchNoteSpecific)
	h.engine.DELETE("/note/:id", h.deleteNoteSpecific)

	// Init routes accordingly
	if h.syncMode == 1 { // local-write
		h.engine.GET("/primary/:id", h.localGetPrimarySpecific)
		h.engine.POST("/backup", h.localUpdateBackup)
		h.engine.PUT("/backup", h.localUpdateBackup)
		h.engine.PATCH("/backup", h.localUpdateBackup)
		h.engine.DELETE("/backup/:id", h.localDeleteBackup)
	} else if h.syncMode == 2 {
		h.engine.POST("/primary", h.remoteForwardPrimary)
		h.engine.PUT("/primary", h.remoteForwardPrimary)
		h.engine.PATCH("/primary", h.remoteForwardPrimary)
		h.engine.DELETE("/primary/:id", h.remoteDeletePrimary)
		h.engine.POST("/backup", h.remoteUpdateBackup)
		h.engine.PUT("/backup", h.remoteUpdateBackup)
		h.engine.PATCH("/backup", h.remoteUpdateBackup)
		h.engine.DELETE("/backup/:id", h.remoteDeleteBackup)
	} // I am just too lazy to consider edge cases :b
}

// Run starts running the API server
// This function is blocking function
func (h *Handler) Run() error {
	addr := fmt.Sprintf("%s:%d", h.addr, h.port)
	log.Printf("Now starting API server in %s", addr)

	// Fire up distributed storage handler
	// We will do 5 times of init processes
	go func() {
		failCount := 0
		h.dsh = ds.New(os.Getenv("SEPH_DATA"), h.replicas)
		for {
			err := h.dsh.Init()
			if err != nil {
				if failCount < 5 {
					failCount++
					log.Printf("[WARN] Sync with replicas[0] failed: %v (%d/%d)", err, failCount, 5)
					time.Sleep(1 * time.Second)
					continue
				} else {
					log.Fatalf("[ERROR] Could not sync with replicas[0], max retry reached")
				}
			} else {
				return
			}
		}
	}()

	err := h.engine.Run(addr)
	if err != nil {
		log.Fatalf("Could not start API server in %s: %v", addr, err)
		return err
	}

	return nil
}
