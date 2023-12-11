package ds

import (
	"encoding/json"
	"errors"
	"fmt"
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
