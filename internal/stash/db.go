package stash

import (
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

const (
	dbSchemaVersion = 1
	dbDriver        = "sqlite"
)

// DB wraps a SQLite connection for stash storage.
type DB struct {
	conn *sql.DB
	path string
}

// OpenDB opens or creates the SQLite stash database.
func OpenDB(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), directoryPermissions); err != nil {
		return nil, fmt.Errorf("create db directory: %w", err)
	}

	conn, err := sql.Open(dbDriver, path+"?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open stash db: %w", err)
	}

	db := &DB{conn: conn, path: path}
	if err := db.migrate(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("migrate stash db: %w", err)
	}
	return db, nil
}

// Close closes the database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) migrate() error {
	// Create schema version table.
	_, err := db.conn.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`)
	if err != nil {
		return err
	}

	var version int
	row := db.conn.QueryRow(`SELECT version FROM schema_version LIMIT 1`)
	if err := row.Scan(&version); err != nil {
		// No version row — fresh database.
		version = 0
	}

	if version < 1 {
		if err := db.migrateV1(); err != nil {
			return err
		}
	}
	return nil
}

func (db *DB) migrateV1() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS stash (
			id          TEXT PRIMARY KEY,
			title       TEXT NOT NULL DEFAULT '',
			type        TEXT NOT NULL DEFAULT '',
			project     TEXT NOT NULL DEFAULT '',
			raw_text    TEXT NOT NULL,
			tags        TEXT NOT NULL DEFAULT '[]',
			refs        TEXT NOT NULL DEFAULT '[]',
			claim_id    TEXT NOT NULL DEFAULT '',
			embedding   BLOB,
			source      TEXT NOT NULL DEFAULT 'cli',
			uniqueness  TEXT NOT NULL DEFAULT 'high',
			created_at  TEXT NOT NULL,
			updated_at  TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS similarity_cache (
			id_a        TEXT NOT NULL,
			id_b        TEXT NOT NULL,
			score       REAL NOT NULL,
			computed_at TEXT NOT NULL,
			PRIMARY KEY (id_a, id_b)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_stash_claim_id ON stash(claim_id)`,
		`CREATE INDEX IF NOT EXISTS idx_stash_project ON stash(project)`,
		`CREATE INDEX IF NOT EXISTS idx_stash_type ON stash(type)`,
		`DELETE FROM schema_version`,
		`INSERT INTO schema_version (version) VALUES (1)`,
	}
	for _, stmt := range statements {
		if _, err := db.conn.Exec(stmt); err != nil {
			return fmt.Errorf("migrate v1: %s: %w", stmt[:40], err)
		}
	}
	return nil
}

// Insert adds an item to the database.
func (db *DB) Insert(item Item) error {
	now := time.Now().UTC().Format(time.RFC3339)
	tags, _ := json.Marshal(item.Tags)
	refs, _ := json.Marshal(item.Refs)
	if tags == nil {
		tags = []byte("[]")
	}
	if refs == nil {
		refs = []byte("[]")
	}

	_, err := db.conn.Exec(
		`INSERT INTO stash (id, title, type, project, raw_text, tags, refs, claim_id, embedding, source, uniqueness, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.Title, string(item.Type), item.Project,
		item.Text, string(tags), string(refs), item.ClaimID,
		encodeEmbedding(item.Embedding), string(item.Source), string(item.Uniqueness),
		item.Created.Format(time.RFC3339), now,
	)
	return err
}

// Get retrieves a single item by ID.
func (db *DB) Get(id string) (Item, error) {
	row := db.conn.QueryRow(
		`SELECT id, title, type, project, raw_text, tags, refs, claim_id, embedding, source, uniqueness, created_at
		 FROM stash WHERE id = ?`, id)
	return scanItem(row)
}

// All returns all stash items ordered by creation time.
func (db *DB) All() ([]Item, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, type, project, raw_text, tags, refs, claim_id, embedding, source, uniqueness, created_at
		 FROM stash ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []Item
	for rows.Next() {
		item, err := scanItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Filter returns items matching optional criteria.
func (db *DB) Filter(project, itemType, tag string) ([]Item, error) {
	query := `SELECT id, title, type, project, raw_text, tags, refs, claim_id, embedding, source, uniqueness, created_at FROM stash WHERE 1=1`
	var args []interface{}

	if project != "" {
		query += ` AND project = ?`
		args = append(args, project)
	}
	if itemType != "" {
		query += ` AND type = ?`
		args = append(args, itemType)
	}
	if tag != "" {
		query += ` AND tags LIKE ?`
		args = append(args, "%"+tag+"%")
	}
	query += ` ORDER BY created_at ASC`

	rows, err := db.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []Item
	for rows.Next() {
		item, err := scanItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// ByClaimID returns all items sharing a claim ID, chronologically.
func (db *DB) ByClaimID(claimID string) ([]Item, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, type, project, raw_text, tags, refs, claim_id, embedding, source, uniqueness, created_at
		 FROM stash WHERE claim_id = ? ORDER BY created_at ASC`, claimID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []Item
	for rows.Next() {
		item, err := scanItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// Delete removes an item by ID.
func (db *DB) Delete(id string) error {
	_, err := db.conn.Exec(`DELETE FROM stash WHERE id = ?`, id)
	return err
}

// UpdateEmbedding sets the embedding for an existing item.
func (db *DB) UpdateEmbedding(id string, embedding []float32) error {
	_, err := db.conn.Exec(`UPDATE stash SET embedding = ? WHERE id = ?`, encodeEmbedding(embedding), id)
	return err
}

// FindSimilar returns items whose embeddings are above the threshold.
// Computes cosine similarity in Go (SQLite has no vector ops).
func (db *DB) FindSimilar(queryEmbed []float32, threshold float64, limit int) ([]SimilarResult, error) {
	if len(queryEmbed) == 0 {
		return nil, nil
	}

	items, err := db.All()
	if err != nil {
		return nil, err
	}

	var results []SimilarResult
	for _, item := range items {
		if len(item.Embedding) == 0 {
			continue
		}
		score := CosineSimilarity(queryEmbed, item.Embedding)
		if score >= threshold {
			results = append(results, SimilarResult{
				Item:  item,
				Score: score,
				Level: ClassifySimilarity(score),
			})
		}
	}

	// Sort by score descending.
	for i := 0; i < len(results); i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Score > results[i].Score {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	return results, nil
}

// Count returns the total number of items.
func (db *DB) Count() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM stash`).Scan(&count)
	return count, err
}

// CountWithEmbeddings returns items that have embeddings.
func (db *DB) CountWithEmbeddings() (int, error) {
	var count int
	err := db.conn.QueryRow(`SELECT COUNT(*) FROM stash WHERE embedding IS NOT NULL AND length(embedding) > 0`).Scan(&count)
	return count, err
}

// ItemsWithoutEmbeddings returns items missing embeddings.
func (db *DB) ItemsWithoutEmbeddings() ([]Item, error) {
	rows, err := db.conn.Query(
		`SELECT id, title, type, project, raw_text, tags, refs, claim_id, embedding, source, uniqueness, created_at
		 FROM stash WHERE embedding IS NULL OR length(embedding) = 0 ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []Item
	for rows.Next() {
		item, err := scanItemRows(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// CacheSimilarity stores a pre-computed similarity score.
func (db *DB) CacheSimilarity(idA, idB string, score float64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.Exec(
		`INSERT OR REPLACE INTO similarity_cache (id_a, id_b, score, computed_at) VALUES (?, ?, ?, ?)`,
		idA, idB, score, now)
	return err
}

func scanItem(row *sql.Row) (Item, error) {
	var item Item
	var itemType, source, uniqueness, createdAt string
	var tagsJSON, refsJSON string
	var embeddingBlob []byte

	err := row.Scan(&item.ID, &item.Title, &itemType, &item.Project,
		&item.Text, &tagsJSON, &refsJSON, &item.ClaimID,
		&embeddingBlob, &source, &uniqueness, &createdAt)
	if err != nil {
		return Item{}, err
	}

	item.Type = ItemType(itemType)
	item.Source = Source(source)
	item.Uniqueness = Uniqueness(uniqueness)
	item.Created, _ = time.Parse(time.RFC3339, createdAt)
	item.Embedding = decodeEmbedding(embeddingBlob)
	_ = json.Unmarshal([]byte(tagsJSON), &item.Tags)
	_ = json.Unmarshal([]byte(refsJSON), &item.Refs)

	return item, nil
}

func scanItemRows(rows *sql.Rows) (Item, error) {
	var item Item
	var itemType, source, uniqueness, createdAt string
	var tagsJSON, refsJSON string
	var embeddingBlob []byte

	err := rows.Scan(&item.ID, &item.Title, &itemType, &item.Project,
		&item.Text, &tagsJSON, &refsJSON, &item.ClaimID,
		&embeddingBlob, &source, &uniqueness, &createdAt)
	if err != nil {
		return Item{}, err
	}

	item.Type = ItemType(itemType)
	item.Source = Source(source)
	item.Uniqueness = Uniqueness(uniqueness)
	item.Created, _ = time.Parse(time.RFC3339, createdAt)
	item.Embedding = decodeEmbedding(embeddingBlob)
	_ = json.Unmarshal([]byte(tagsJSON), &item.Tags)
	_ = json.Unmarshal([]byte(refsJSON), &item.Refs)

	return item, nil
}

func encodeEmbedding(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}
	return buf
}

func decodeEmbedding(b []byte) []float32 {
	if len(b) == 0 || len(b)%4 != 0 {
		return nil
	}
	v := make([]float32, len(b)/4)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return v
}
