package stash

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := OpenDB(path)
	if err != nil {
		t.Fatalf("failed to open test db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestDBInsertAndGet(t *testing.T) {
	db := openTestDB(t)

	item := Item{
		ID:      "test-001",
		Text:    "test idea",
		Title:   "Test",
		Type:    ItemTypeInsight,
		Project: "vectorpad",
		Tags:    []string{"go", "tui"},
		ClaimID: "claim-abc",
		Source:  SourceCLI,
		Created: time.Now().UTC().Truncate(time.Second),
	}

	if err := db.Insert(item); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := db.Get("test-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if got.Text != item.Text {
		t.Errorf("text: got %q, want %q", got.Text, item.Text)
	}
	if got.Title != item.Title {
		t.Errorf("title: got %q, want %q", got.Title, item.Title)
	}
	if got.Type != item.Type {
		t.Errorf("type: got %q, want %q", got.Type, item.Type)
	}
	if got.Project != item.Project {
		t.Errorf("project: got %q, want %q", got.Project, item.Project)
	}
	if got.ClaimID != item.ClaimID {
		t.Errorf("claim_id: got %q, want %q", got.ClaimID, item.ClaimID)
	}
	if len(got.Tags) != 2 || got.Tags[0] != "go" {
		t.Errorf("tags: got %v, want [go tui]", got.Tags)
	}
}

func TestDBAll(t *testing.T) {
	db := openTestDB(t)

	for i := 0; i < 3; i++ {
		item := Item{
			ID:      fmt.Sprintf("item-%d", i),
			Text:    fmt.Sprintf("idea %d", i),
			Source:  SourceCLI,
			Created: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		if err := db.Insert(item); err != nil {
			t.Fatalf("insert %d: %v", i, err)
		}
	}

	items, err := db.All()
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
}

func TestDBDelete(t *testing.T) {
	db := openTestDB(t)

	item := Item{ID: "del-001", Text: "to delete", Source: SourceCLI, Created: time.Now().UTC()}
	_ = db.Insert(item)
	_ = db.Delete("del-001")

	_, err := db.Get("del-001")
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestDBEmbeddingRoundTrip(t *testing.T) {
	db := openTestDB(t)

	embed := []float32{0.1, 0.2, 0.3, -0.5}
	item := Item{
		ID:        "emb-001",
		Text:      "embedded idea",
		Source:    SourceCLI,
		Created:   time.Now().UTC(),
		Embedding: embed,
	}
	if err := db.Insert(item); err != nil {
		t.Fatalf("insert: %v", err)
	}

	got, err := db.Get("emb-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}

	if len(got.Embedding) != len(embed) {
		t.Fatalf("embedding length: got %d, want %d", len(got.Embedding), len(embed))
	}
	for i, v := range got.Embedding {
		if v != embed[i] {
			t.Errorf("embedding[%d]: got %f, want %f", i, v, embed[i])
		}
	}
}

func TestDBFindSimilar(t *testing.T) {
	db := openTestDB(t)

	// Insert items with known embeddings.
	items := []Item{
		{ID: "a", Text: "alpha", Source: SourceCLI, Created: time.Now().UTC(), Embedding: []float32{1, 0, 0}},
		{ID: "b", Text: "beta", Source: SourceCLI, Created: time.Now().UTC(), Embedding: []float32{0.9, 0.1, 0}},
		{ID: "c", Text: "gamma", Source: SourceCLI, Created: time.Now().UTC(), Embedding: []float32{0, 1, 0}},
	}
	for _, item := range items {
		if err := db.Insert(item); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	// Query close to "a".
	results, err := db.FindSimilar([]float32{1, 0, 0}, 0.5, 10)
	if err != nil {
		t.Fatalf("find similar: %v", err)
	}
	if len(results) < 2 {
		t.Fatalf("expected at least 2 results above 0.5, got %d", len(results))
	}
	if results[0].Item.ID != "a" {
		t.Errorf("expected first result to be 'a', got %q", results[0].Item.ID)
	}
}

func TestDBFilter(t *testing.T) {
	db := openTestDB(t)

	_ = db.Insert(Item{ID: "f1", Text: "go insight", Type: ItemTypeInsight, Project: "vp", Source: SourceCLI, Created: time.Now().UTC()})
	_ = db.Insert(Item{ID: "f2", Text: "py question", Type: ItemTypeQuestion, Project: "cs", Source: SourceCLI, Created: time.Now().UTC()})

	items, err := db.Filter("vp", "", "")
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(items) != 1 || items[0].ID != "f1" {
		t.Errorf("filter by project: got %v", items)
	}

	items, err = db.Filter("", "question", "")
	if err != nil {
		t.Fatalf("filter: %v", err)
	}
	if len(items) != 1 || items[0].ID != "f2" {
		t.Errorf("filter by type: got %v", items)
	}
}

func TestDBByClaimID(t *testing.T) {
	db := openTestDB(t)

	_ = db.Insert(Item{ID: "c1", Text: "v1", ClaimID: "claim-x", Source: SourceCLI, Created: time.Now().UTC()})
	_ = db.Insert(Item{ID: "c2", Text: "v2", ClaimID: "claim-x", Source: SourceCLI, Created: time.Now().UTC().Add(time.Minute)})
	_ = db.Insert(Item{ID: "c3", Text: "other", ClaimID: "claim-y", Source: SourceCLI, Created: time.Now().UTC()})

	items, err := db.ByClaimID("claim-x")
	if err != nil {
		t.Fatalf("by claim id: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items for claim-x, got %d", len(items))
	}
}

func TestDBUpdateEmbedding(t *testing.T) {
	db := openTestDB(t)

	_ = db.Insert(Item{ID: "u1", Text: "no embed", Source: SourceCLI, Created: time.Now().UTC()})

	newEmbed := []float32{0.5, 0.5, 0.5}
	if err := db.UpdateEmbedding("u1", newEmbed); err != nil {
		t.Fatalf("update embedding: %v", err)
	}

	got, _ := db.Get("u1")
	if len(got.Embedding) != 3 {
		t.Errorf("expected 3-dim embedding after update, got %d", len(got.Embedding))
	}
}

func TestDBMigrationIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrate.db")

	db1, err := OpenDB(path)
	if err != nil {
		t.Fatalf("first open: %v", err)
	}
	_ = db1.Insert(Item{ID: "m1", Text: "test", Source: SourceCLI, Created: time.Now().UTC()})
	_ = db1.Close()

	// Re-open should not fail or lose data.
	db2, err := OpenDB(path)
	if err != nil {
		t.Fatalf("second open: %v", err)
	}
	defer func() { _ = db2.Close() }()

	got, err := db2.Get("m1")
	if err != nil {
		t.Fatalf("get after reopen: %v", err)
	}
	if got.Text != "test" {
		t.Errorf("data lost after reopen")
	}
}

func TestEncodeDecodeEmbedding(t *testing.T) {
	original := []float32{1.5, -2.3, 0.0, 100.0}
	encoded := encodeEmbedding(original)
	decoded := decodeEmbedding(encoded)

	if len(decoded) != len(original) {
		t.Fatalf("length mismatch: got %d, want %d", len(decoded), len(original))
	}
	for i := range original {
		if decoded[i] != original[i] {
			t.Errorf("[%d]: got %f, want %f", i, decoded[i], original[i])
		}
	}
}

func TestDecodeEmbeddingNil(t *testing.T) {
	if got := decodeEmbedding(nil); got != nil {
		t.Errorf("expected nil for nil input, got %v", got)
	}
}
