package stash

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	vectorpadHomeEnv     = "VECTORPAD_HOME"
	defaultVectorpadHome = ".vectorpad"
	stashDirName         = "stash"
	stashFileName        = "stacks.json"
	filePermissions      = 0o600
	directoryPermissions = 0o700
)

var (
	ErrEmptyText          = errors.New("stash text cannot be empty")
	ErrUnsupportedVersion = errors.New("unsupported stash file version")
)

type Store struct {
	path        string
	now         func() time.Time
	idGenerator func() (string, error)
	mu          sync.Mutex
}

func NewStore(path string) *Store {
	return &Store{
		path:        path,
		now:         time.Now,
		idGenerator: generateItemID,
	}
}

func NewDefaultStore() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return NewStore(path), nil
}

func DefaultPath() (string, error) {
	base, err := DefaultHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, stashDirName, stashFileName), nil
}

func DefaultHome() (string, error) {
	home := strings.TrimSpace(os.Getenv(vectorpadHomeEnv))
	if home != "" {
		return filepath.Clean(home), nil
	}

	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(userHome, defaultVectorpadHome), nil
}

func (store *Store) Path() string {
	return store.path
}

func (store *Store) Load() (StashFile, error) {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.loadLocked()
}

func (store *Store) Save(file StashFile) error {
	store.mu.Lock()
	defer store.mu.Unlock()

	return store.saveLocked(file)
}

func (store *Store) Add(text string, source Source) (Item, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return Item{}, ErrEmptyText
	}

	if source == "" {
		source = SourceCLI
	}

	store.mu.Lock()
	defer store.mu.Unlock()

	file, err := store.loadLocked()
	if err != nil {
		return Item{}, err
	}

	now := store.now().UTC()
	id, err := store.idGenerator()
	if err != nil {
		return Item{}, fmt.Errorf("generate stash item id: %w", err)
	}

	item := Item{
		ID:         id,
		Text:       trimmed,
		Created:    now,
		Uniqueness: UniquenessHigh,
		Source:     source,
	}

	items := flattenItems(file.Stacks)
	items = append(items, item)

	file.Stacks = ClusterItems(items, now)
	file.Version = CurrentVersion
	if err := store.saveLocked(file); err != nil {
		return Item{}, err
	}

	stored, ok := findItemByID(file.Stacks, id)
	if !ok {
		return item, nil
	}
	return stored, nil
}

func findItemByID(stacks []Stack, id string) (Item, bool) {
	for _, stack := range stacks {
		for _, item := range stack.Items {
			if item.ID == id {
				return item, true
			}
		}
	}
	return Item{}, false
}

func (store *Store) loadLocked() (StashFile, error) {
	data, err := os.ReadFile(store.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return newEmptyStash(), nil
		}
		return StashFile{}, fmt.Errorf("read stash file: %w", err)
	}

	file, err := decodeStashFile(data)
	if err != nil {
		backupData, backupErr := os.ReadFile(store.backupPath())
		if backupErr == nil {
			backupFile, backupDecodeErr := decodeStashFile(backupData)
			if backupDecodeErr == nil {
				migratedBackup, _, migrateErr := migrate(backupFile)
				if migrateErr != nil {
					return StashFile{}, migrateErr
				}
				return migratedBackup, nil
			}
		}
		return StashFile{}, fmt.Errorf("decode stash file: %w", err)
	}

	migrated, didMigrate, err := migrate(file)
	if err != nil {
		return StashFile{}, err
	}
	if didMigrate {
		if err := store.saveLocked(migrated); err != nil {
			return StashFile{}, fmt.Errorf("persist migrated stash file: %w", err)
		}
	}

	return migrated, nil
}

func decodeStashFile(data []byte) (StashFile, error) {
	var file StashFile
	if err := json.Unmarshal(data, &file); err != nil {
		return StashFile{}, err
	}
	return file, nil
}

func migrate(file StashFile) (StashFile, bool, error) {
	didMigrate := false
	switch file.Version {
	case 0:
		file.Version = CurrentVersion
		didMigrate = true
	case CurrentVersion:
		// no-op
	default:
		return StashFile{}, false, fmt.Errorf("%w: got %d, support is up to %d", ErrUnsupportedVersion, file.Version, CurrentVersion)
	}

	if file.Stacks == nil {
		file.Stacks = []Stack{}
		didMigrate = true
	}

	for stackIndex := range file.Stacks {
		stack := &file.Stacks[stackIndex]
		if stack.Items == nil {
			stack.Items = []Item{}
			didMigrate = true
		}
		if stack.Label == "" && stack.ID == UnclusteredStackID {
			stack.Label = UnclusteredStackLabel
			didMigrate = true
		}

		for itemIndex := range stack.Items {
			item := &stack.Items[itemIndex]
			if item.ID == "" {
				item.ID = fmt.Sprintf("legacy-%d-%d", stackIndex, itemIndex)
				didMigrate = true
			}
			if item.Source == "" {
				item.Source = SourcePaste
				didMigrate = true
			}
			if item.Uniqueness == "" {
				item.Uniqueness = UniquenessHigh
				didMigrate = true
			}
		}

		if stack.Created.IsZero() || stack.Updated.IsZero() {
			created, updated := stackBounds(stack.Items, time.Now().UTC())
			if stack.Created.IsZero() {
				stack.Created = created
				didMigrate = true
			}
			if stack.Updated.IsZero() {
				stack.Updated = updated
				didMigrate = true
			}
		}
	}

	return file, didMigrate, nil
}

func newEmptyStash() StashFile {
	return StashFile{
		Stacks:  []Stack{},
		Version: CurrentVersion,
	}
}

func (store *Store) saveLocked(file StashFile) error {
	normalized, _, err := migrate(file)
	if err != nil {
		return err
	}
	normalized.Version = CurrentVersion

	encoded, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return fmt.Errorf("encode stash file: %w", err)
	}
	encoded = append(encoded, '\n')

	if err := os.MkdirAll(filepath.Dir(store.path), directoryPermissions); err != nil {
		return fmt.Errorf("create stash directory: %w", err)
	}

	existing, err := os.ReadFile(store.path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("read existing stash file: %w", err)
	}
	if err == nil {
		if backupErr := writeAtomic(store.backupPath(), existing); backupErr != nil {
			return fmt.Errorf("write stash backup: %w", backupErr)
		}
	}

	if err := writeAtomic(store.path, encoded); err != nil {
		return fmt.Errorf("write stash file: %w", err)
	}

	return nil
}

func writeAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, directoryPermissions); err != nil {
		return err
	}

	tempFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-")
	if err != nil {
		return err
	}
	tempPath := tempFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tempPath)
		}
	}()

	if err := tempFile.Chmod(filePermissions); err != nil {
		_ = tempFile.Close()
		return err
	}
	if _, err := tempFile.Write(content); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Sync(); err != nil {
		_ = tempFile.Close()
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	cleanup = false

	return syncDirectory(dir)
}

func syncDirectory(dir string) error {
	handle, err := os.Open(dir)
	if err != nil {
		return nil
	}
	defer func() { _ = handle.Close() }()

	if err := handle.Sync(); err != nil {
		return nil
	}
	return nil
}

func (store *Store) backupPath() string {
	return store.path + ".bak"
}

func generateItemID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
