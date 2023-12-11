package ds

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"seph/common"
)

// ReadAll reads all notes in the target directory
func (h *Handler) ReadAll() []common.Note {
	// Open the target directory which stores all notes
	files, err := os.ReadDir(h.targetDir)
	if err != nil {
		log.Printf("[Seph] Error reading directory %s: %v\n", h.targetDir, err)
		return nil
	}

	// Create an empty note for storing every note in the directory
	allNotes := make([]common.Note, 0)

	// Read all .json files in the target directory
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			filePath := filepath.Join(h.targetDir, file.Name())
			err, content := h.readNoteFromFile(filePath)
			if err != nil {
				log.Printf("[Seph] Error reading note from file %s: %v\n", filePath, err)
				continue
			}

			// Append the note
			allNotes = append(allNotes, content)
		}
	}

	return allNotes
}

// ReadSpecific reads specific designated note file
// The convention for a note is NOTE_ID.json, ex 1.json
func (h *Handler) ReadSpecific(id int) (error, common.Note) {
	// Construct target file name
	fileName := fmt.Sprintf("%d.json", id)
	fileName = path.Join(h.targetDir, fileName)

	return h.readNoteFromFile(fileName)
}

// readNoteFromFile reads a specific file as note format
func (h *Handler) readNoteFromFile(filePath string) (error, common.Note) {
	// Read the designated file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err, common.Note{}
	}

	// Try unmarshalling into target file
	var content common.Note
	err = json.Unmarshal(data, &content)
	if err != nil {
		return err, content
	}

	return nil, content
}
