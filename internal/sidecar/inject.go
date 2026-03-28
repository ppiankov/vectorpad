package sidecar

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// InjectUserMessage appends a user message entry to the session JSONL file.
// The entry is chained to the last entry via parentUuid.
func InjectUserMessage(sessionPath, text string) error {
	parentUUID, err := LastEntryUUID(sessionPath)
	if err != nil {
		return fmt.Errorf("find parent: %w", err)
	}

	return InjectUserMessageWithParent(sessionPath, text, parentUUID)
}

// InjectUserMessageWithParent appends a user message entry with an explicit parent UUID.
func InjectUserMessageWithParent(sessionPath, text, parentUUID string) error {
	content, err := json.Marshal(text)
	if err != nil {
		return fmt.Errorf("encode content: %w", err)
	}

	entry := Entry{
		Type:       "user",
		ParentUUID: parentUUID,
		Timestamp:  time.Now().UTC(),
		Message: &Message{
			Role:    "user",
			Content: json.RawMessage(content),
		},
	}

	line, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("encode entry: %w", err)
	}

	f, err := os.OpenFile(sessionPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open session file: %w", err)
	}
	defer func() { _ = f.Close() }()

	// JSONL: one JSON object per line, newline terminated.
	line = append(line, '\n')
	if _, err := f.Write(line); err != nil {
		return fmt.Errorf("write entry: %w", err)
	}

	return nil
}
