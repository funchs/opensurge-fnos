package controlapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

const (
	guiPasswordFile     = "gui-password"
	guiPasswordMinRunes = 8
	guiPasswordMaxRunes = 128
	guiPasswordAlgo     = "bcrypt"
)

var (
	errGUIPasswordConfigured = errors.New("gui password is already configured")
	errGUIPasswordInvalid    = errors.New("gui password must be 8 to 128 characters")
)

type guiPasswordRecord struct {
	SchemaVersion int    `json:"schema_version"`
	Algo          string `json:"algo"`
	Hash          string `json:"hash"`
	UpdatedAt     string `json:"updated_at"`
}

func (s *Store) guiPasswordPath() string {
	return filepath.Join(s.dir, guiPasswordFile)
}

func validateGUIPassword(password string) error {
	n := utf8.RuneCountInString(password)
	if n < guiPasswordMinRunes || n > guiPasswordMaxRunes {
		return errGUIPasswordInvalid
	}
	return nil
}

func hashGUIPassword(password string) (guiPasswordRecord, error) {
	if err := validateGUIPassword(password); err != nil {
		return guiPasswordRecord{}, err
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return guiPasswordRecord{}, err
	}
	return guiPasswordRecord{
		SchemaVersion: SchemaVersion,
		Algo:          guiPasswordAlgo,
		Hash:          string(hashed),
		UpdatedAt:     time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func writeGUIPasswordRecord(path string, record guiPasswordRecord) error {
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(data, '\n'), 0o600)
}

func (s *Store) CreateGUIPassword(password string) error {
	record, err := hashGUIPassword(password)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	path := s.guiPasswordPath()
	if _, err := os.Stat(path); err == nil {
		return errGUIPasswordConfigured
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return writeGUIPasswordRecord(path, record)
}

func (s *Store) ReplaceGUIPassword(password string) error {
	record, err := hashGUIPassword(password)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return writeGUIPasswordRecord(s.guiPasswordPath(), record)
}

func (s *Store) loadGUIPasswordLocked() (guiPasswordRecord, error) {
	var record guiPasswordRecord
	err := readJSON(s.guiPasswordPath(), &record)
	if err != nil {
		return guiPasswordRecord{}, err
	}
	if record.Algo != guiPasswordAlgo || record.Hash == "" {
		return guiPasswordRecord{}, fmt.Errorf("gui password file is unusable")
	}
	return record, nil
}

func (s *Store) PasswordMatches(password string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	record, err := s.loadGUIPasswordLocked()
	if err != nil {
		return false, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(record.Hash), []byte(password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (s *Store) GUIPasswordConfigured() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.loadGUIPasswordLocked()
	return err == nil
}
