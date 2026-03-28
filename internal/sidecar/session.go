package sidecar

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const claudeProjectsDir = ".claude/projects"

// Session represents a discovered Claude Code session.
type Session struct {
	ID         string    // UUID filename (without .jsonl)
	Path       string    // full path to JSONL file
	ModTime    time.Time // last modification time
	EntryCount int       // number of JSONL lines
}

// Entry is a minimal representation of a Claude Code JSONL entry.
// Only the fields needed for sidecar injection are modeled.
type Entry struct {
	Type       string    `json:"type"`
	UUID       string    `json:"uuid,omitempty"`
	ParentUUID string    `json:"parentUuid,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
	SessionID  string    `json:"sessionId,omitempty"`
	Message    *Message  `json:"message,omitempty"`
}

// Message is the role + content portion of an entry.
type Message struct {
	Role    string          `json:"role,omitempty"`
	Content json.RawMessage `json:"content"`
}

// SessionStats holds summary info about a session for display.
type SessionStats struct {
	TurnCount   int       // total entries
	UserTurns   int       // user message count
	AssistTurns int       // assistant message count
	LastActive  time.Time // timestamp of last entry
	LastUUID    string    // UUID of last entry (for parentUuid chaining)
	SessionID   string    // sessionId from entries
}

// DiscoverSessions finds Claude Code session JSONL files for the given project directory.
func DiscoverSessions(projectDir string) ([]Session, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("home dir: %w", err)
	}

	// Claude Code encodes project paths by replacing / with -
	encoded := encodeProjectPath(projectDir)
	sessDir := filepath.Join(home, claudeProjectsDir, encoded)

	entries, err := os.ReadDir(sessDir)
	if err != nil {
		return nil, fmt.Errorf("read sessions dir: %w", err)
	}

	var sessions []Session
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".jsonl")
		sessions = append(sessions, Session{
			ID:      id,
			Path:    filepath.Join(sessDir, e.Name()),
			ModTime: info.ModTime(),
		})
	}

	// Sort by most recently modified first.
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].ModTime.After(sessions[j].ModTime)
	})

	return sessions, nil
}

// ReadStats reads summary statistics from a session JSONL file.
func ReadStats(path string) (*SessionStats, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open session: %w", err)
	}
	defer func() { _ = f.Close() }()

	stats := &SessionStats{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB line buffer

	var lastEntry Entry
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var e Entry
		if json.Unmarshal(line, &e) != nil {
			continue
		}

		stats.TurnCount++
		if e.Message != nil {
			switch e.Message.Role {
			case "user":
				stats.UserTurns++
			case "assistant":
				stats.AssistTurns++
			}
		}
		if e.SessionID != "" {
			stats.SessionID = e.SessionID
		}
		lastEntry = e
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan session: %w", err)
	}

	stats.LastActive = lastEntry.Timestamp
	if lastEntry.UUID != "" {
		stats.LastUUID = lastEntry.UUID
	} else if lastEntry.ParentUUID != "" {
		// Some entries don't have UUID but have parentUUID
		stats.LastUUID = lastEntry.ParentUUID
	}

	return stats, nil
}

// LastEntryUUID reads the JSONL file and returns the UUID of the last entry
// that has one. This becomes the parentUuid for injected messages.
func LastEntryUUID(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("open session: %w", err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var lastUUID string
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		// Fast extraction: only parse uuid field.
		var partial struct {
			UUID string `json:"uuid,omitempty"`
		}
		if json.Unmarshal(line, &partial) == nil && partial.UUID != "" {
			lastUUID = partial.UUID
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("scan session: %w", err)
	}

	if lastUUID == "" {
		return "", fmt.Errorf("no UUID found in session")
	}

	return lastUUID, nil
}

// encodeProjectPath converts an absolute path to Claude Code's project directory encoding.
// /Users/pashah/dev/myproject -> -Users-pashah-dev-myproject
func encodeProjectPath(p string) string {
	return strings.ReplaceAll(p, "/", "-")
}
