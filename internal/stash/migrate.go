package stash

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

// MigrateJSON imports items from the old JSON stash format into the SQLite DB.
// Returns the number of items imported.
func MigrateJSON(jsonPath string, db *DB, embedder *Embedder) (int, error) {
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, fmt.Errorf("read json stash: %w", err)
	}

	var file StashFile
	if err := json.Unmarshal(data, &file); err != nil {
		return 0, fmt.Errorf("decode json stash: %w", err)
	}

	items := flattenItems(file.Stacks)
	if len(items) == 0 {
		return 0, nil
	}

	count := 0
	for _, item := range items {
		// Skip items already in DB (idempotent migration).
		if _, err := db.Get(item.ID); err == nil {
			continue
		}

		// Compute embedding if embedder is available.
		if embedder != nil && embedder.Available() {
			if vec, err := embedder.Embed(item.Text); err == nil {
				item.Embedding = vec
			}
		}

		if err := db.Insert(item); err != nil {
			return count, fmt.Errorf("insert migrated item %s: %w", item.ID, err)
		}
		count++
	}

	return count, nil
}
