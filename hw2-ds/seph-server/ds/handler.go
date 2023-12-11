package ds

import "sync"

// Handler represents a single distributed storage handler
type Handler struct {
	lock      sync.Mutex
	targetDir string
}

// New creates a new Handler
func New(targetDir string) *Handler {
	return &Handler{
		lock:      sync.Mutex{},
		targetDir: targetDir,
	}
}
