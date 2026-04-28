package session

import (
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrInvalidSessionID = errors.New("invalid session identifier")
	ErrRepoMismatch     = errors.New("session belongs to a different repository")
	ErrSessionNotFound  = errors.New("session not found")
)

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func ValidateSessionID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" || id == "." || id == ".." || len(id) > 128 {
		return ErrInvalidSessionID
	}
	if strings.ContainsAny(id, `/\`) || !sessionIDPattern.MatchString(id) {
		return ErrInvalidSessionID
	}
	return nil
}

func HashSessionID(id string) string {
	if strings.TrimSpace(id) == "" {
		return ""
	}
	sum := md5.Sum([]byte(id))
	return hex.EncodeToString(sum[:])
}

func GenerateSessionID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "session-default"
	}
	return "session-" + hex.EncodeToString(b[:])
}

func RepoKey(repoRoot string) (string, error) {
	if strings.TrimSpace(repoRoot) == "" {
		return "", fmt.Errorf("repo root is required")
	}
	abs, err := filepath.Abs(repoRoot)
	if err != nil {
		return "", err
	}
	clean := filepath.Clean(abs)
	if vol := filepath.VolumeName(clean); vol != "" {
		clean = vol + strings.ToLower(strings.TrimPrefix(clean, vol))
	}
	sum := sha256.Sum256([]byte(clean))
	return hex.EncodeToString(sum[:16]), nil
}
