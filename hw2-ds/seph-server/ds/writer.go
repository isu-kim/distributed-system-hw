package ds

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"seph/common"
)

// CreateNote creates a new note file
// The convention for a note is NOTE_ID.json, ex 1.json
func (h *Handler) CreateNote(note common.Note) error {
	// Construct target file name
	fileName := fmt.Sprintf("%d.json", note.Id)
	fileName = path.Join(h.targetDir, fileName)

	// Marshal note into JSON
	noteJSON, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		msg := fmt.Sprintf("error marshalling note %s to JSON: %v", fileName, err)
		return errors.New(msg)
	}

	// Write JSON to file
	err = os.WriteFile(fileName, noteJSON, 0644)
	if err != nil {
		msg := fmt.Sprintf("error writing note %s: %v", fileName, err)
		return errors.New(msg)
	}

	return nil
}

// UpdateNote updates an existing note file
// The note file did not exist, this will return error
func (h *Handler) UpdateNote(note common.Note) error {
	// Construct target file name
	fileName := fmt.Sprintf("%d.json", note.Id)
	fileName = path.Join(h.targetDir, fileName)

	// Check if file exists
	_, err := os.Stat(fileName)
	if err != nil {
		msg := fmt.Sprintf("could not stat note file %s: %v", fileName, err)
		return errors.New(msg)
	}

	// This Means that the file exists, so just overwrite
	// Marshal note into JSON
	noteJSON, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		msg := fmt.Sprintf("error marshalling note %s to JSON: %v", fileName, err)
		return errors.New(msg)
	}

	// Write JSON to file
	err = os.WriteFile(fileName, noteJSON, 0644)
	if err != nil {
		msg := fmt.Sprintf("error writing note %s: %v", fileName, err)
		return errors.New(msg)
	}

	return nil
}

// DumpNotes dump all notes into the target directory
// This is meant for initialization process, not intended to be called afterwards
// This function will not consider any cases which partially failed
func (h *Handler) DumpNotes(notes []common.Note) {
	// For all notes, iterate and create
	for _, note := range notes {
		err := h.WriteNote(note)
		if err != nil {
			log.Printf("[Seph] Error dumping note %d: %v", note.Id, err)
			continue
		}
	}
}

// DeleteNote deletes a specific note
// Yes this will make a fragmentation, but whatsoever
func (h *Handler) DeleteNote(id int) error {
	// Construct target file name
	fileName := fmt.Sprintf("%d.json", id)
	fileName = path.Join(h.targetDir, fileName)

	// Try removing file
	return os.Remove(fileName)
}

// WriteNote will just force writing note to a file
func (h *Handler) WriteNote(note common.Note) error {
	// Construct target file name
	fileName := fmt.Sprintf("%d.json", note.Id)
	fileName = path.Join(h.targetDir, fileName)

	// This Means that the file exists, so just overwrite
	// Marshal note into JSON
	noteJSON, err := json.MarshalIndent(note, "", "  ")
	if err != nil {
		msg := fmt.Sprintf("error marshalling note %s to JSON: %v", fileName, err)
		return errors.New(msg)
	}

	// Write JSON to file
	err = os.WriteFile(fileName, noteJSON, 0644)
	if err != nil {
		msg := fmt.Sprintf("error writing note %s: %v", fileName, err)
		return errors.New(msg)
	}

	return nil
}
