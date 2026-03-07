package stash

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultPathUsesVectorpadHomeWhenSet(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv(vectorpadHomeEnv, tempHome)

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("expected default path without error, got %v", err)
	}

	want := filepath.Join(tempHome, stashDirName, stashFileName)
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestStoreAddAndLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), stashDirName, stashFileName)
	store := NewStore(path)
	store.now = func() time.Time {
		return time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	}
	store.idGenerator = fixedIDGenerator("item-1")

	item, err := store.Add("cache-read amplification as a metric", SourceCLI)
	if err != nil {
		t.Fatalf("expected add to succeed, got %v", err)
	}
	if item.ID != "item-1" {
		t.Fatalf("expected item id item-1, got %q", item.ID)
	}

	file, err := store.Load()
	if err != nil {
		t.Fatalf("expected load to succeed, got %v", err)
	}
	if file.Version != CurrentVersion {
		t.Fatalf("expected version %d, got %d", CurrentVersion, file.Version)
	}

	items := flattenItems(file.Stacks)
	if len(items) != 1 {
		t.Fatalf("expected 1 item after round trip, got %d", len(items))
	}
	if items[0].Text != "cache-read amplification as a metric" {
		t.Fatalf("expected original item text, got %q", items[0].Text)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected stash file to exist, got %v", err)
	}
}

func TestStoreSaveCreatesSingleBackupVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), stashDirName, stashFileName)
	store := NewStore(path)

	base := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	var tick atomic.Int64
	store.now = func() time.Time {
		current := tick.Add(1)
		return base.Add(time.Duration(current) * time.Minute)
	}
	store.idGenerator = sequentialIDGenerator()

	if _, err := store.Add("first stash entry", SourceCLI); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	firstBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read first stash file: %v", err)
	}

	if _, err := store.Add("second stash entry", SourceCLI); err != nil {
		t.Fatalf("second add failed: %v", err)
	}

	backupBytes, err := os.ReadFile(path + ".bak")
	if err != nil {
		t.Fatalf("expected backup file after second save, got %v", err)
	}
	if string(backupBytes) != string(firstBytes) {
		t.Fatalf("expected backup to preserve previous version")
	}

	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), stashFileName+".tmp-*"))
	if err != nil {
		t.Fatalf("glob temp files: %v", err)
	}
	if len(matches) != 0 {
		t.Fatalf("expected no leftover temp files, found %d", len(matches))
	}
}

func TestStoreLoadMigratesVersionZeroFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), stashDirName, stashFileName)
	if err := os.MkdirAll(filepath.Dir(path), directoryPermissions); err != nil {
		t.Fatalf("create stash directory: %v", err)
	}

	legacyJSON := `{
  "stacks": [
    {
      "id": "legacy",
      "label": "Legacy",
      "items": [
        {
          "text": "legacy stash idea",
          "created": "2026-03-07T10:30:00Z"
        }
      ]
    }
  ]
}`
	if err := os.WriteFile(path, []byte(legacyJSON), filePermissions); err != nil {
		t.Fatalf("write legacy stash file: %v", err)
	}

	store := NewStore(path)
	file, err := store.Load()
	if err != nil {
		t.Fatalf("expected migrated load to succeed, got %v", err)
	}
	if file.Version != CurrentVersion {
		t.Fatalf("expected version %d after migration, got %d", CurrentVersion, file.Version)
	}

	items := flattenItems(file.Stacks)
	if len(items) != 1 {
		t.Fatalf("expected 1 migrated item, got %d", len(items))
	}
	if items[0].ID == "" {
		t.Fatal("expected migration to assign item id")
	}
	if items[0].Source == "" {
		t.Fatal("expected migration to assign item source")
	}

	migratedBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read migrated file: %v", err)
	}
	var migrated StashFile
	if err := json.Unmarshal(migratedBytes, &migrated); err != nil {
		t.Fatalf("decode migrated json: %v", err)
	}
	if migrated.Version != CurrentVersion {
		t.Fatalf("expected persisted migrated version %d, got %d", CurrentVersion, migrated.Version)
	}

	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatalf("expected backup to preserve pre-migration file, got %v", err)
	}
}

func TestStoreLoadRejectsUnsupportedFutureVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), stashDirName, stashFileName)
	if err := os.MkdirAll(filepath.Dir(path), directoryPermissions); err != nil {
		t.Fatalf("create stash directory: %v", err)
	}

	futureVersionJSON := `{"version": 99, "stacks": []}`
	if err := os.WriteFile(path, []byte(futureVersionJSON), filePermissions); err != nil {
		t.Fatalf("write future version stash file: %v", err)
	}

	store := NewStore(path)
	_, err := store.Load()
	if err == nil {
		t.Fatal("expected unsupported version error, got nil")
	}
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("expected ErrUnsupportedVersion, got %v", err)
	}
}

func TestStoreLoadFallsBackToBackupWhenPrimaryCorrupt(t *testing.T) {
	path := filepath.Join(t.TempDir(), stashDirName, stashFileName)
	store := NewStore(path)

	base := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	var tick atomic.Int64
	store.now = func() time.Time {
		current := tick.Add(1)
		return base.Add(time.Duration(current) * time.Minute)
	}
	store.idGenerator = sequentialIDGenerator()

	if _, err := store.Add("first stash entry", SourceCLI); err != nil {
		t.Fatalf("first add failed: %v", err)
	}
	if _, err := store.Add("second stash entry", SourceCLI); err != nil {
		t.Fatalf("second add failed: %v", err)
	}

	if err := os.WriteFile(path, []byte("{broken-json"), filePermissions); err != nil {
		t.Fatalf("corrupt primary stash file: %v", err)
	}

	file, err := store.Load()
	if err != nil {
		t.Fatalf("expected load fallback to backup, got %v", err)
	}
	if len(flattenItems(file.Stacks)) != 1 {
		t.Fatalf("expected backup version with 1 item, got %d", len(flattenItems(file.Stacks)))
	}
}

func TestStoreConcurrentAddsRemainConsistent(t *testing.T) {
	path := filepath.Join(t.TempDir(), stashDirName, stashFileName)
	store := NewStore(path)

	base := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	var tick atomic.Int64
	store.now = func() time.Time {
		current := tick.Add(1)
		return base.Add(time.Duration(current) * time.Second)
	}

	var idCounter atomic.Int64
	store.idGenerator = func() (string, error) {
		next := idCounter.Add(1)
		return fmt.Sprintf("item-%03d", next), nil
	}

	const goroutines = 24
	var wait sync.WaitGroup
	for index := 0; index < goroutines; index++ {
		index := index
		wait.Add(1)
		go func() {
			defer wait.Done()
			text := fmt.Sprintf("idea-%02d cache token routing", index)
			if index%2 == 0 {
				text = fmt.Sprintf("idea-%02d terminal focus shortcuts", index)
			}
			if _, err := store.Add(text, SourceCLI); err != nil {
				t.Errorf("add failed for index %d: %v", index, err)
			}
		}()
	}
	wait.Wait()

	file, err := store.Load()
	if err != nil {
		t.Fatalf("load after concurrent writes failed: %v", err)
	}
	if len(flattenItems(file.Stacks)) != goroutines {
		t.Fatalf("expected %d items, got %d", goroutines, len(flattenItems(file.Stacks)))
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read stash file: %v", err)
	}
	var decoded StashFile
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("stash file must remain valid json after concurrent writes: %v", err)
	}
}

func fixedIDGenerator(value string) func() (string, error) {
	return func() (string, error) {
		return value, nil
	}
}

func sequentialIDGenerator() func() (string, error) {
	var count int64
	return func() (string, error) {
		count++
		return fmt.Sprintf("item-%d", count), nil
	}
}
