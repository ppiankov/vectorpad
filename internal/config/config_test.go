package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	cfg := &Config{
		VectorCourt: VectorCourtConfig{
			APIKey:   "vc_test_key",
			Endpoint: "https://custom.example.com",
		},
	}

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.VectorCourt.APIKey != cfg.VectorCourt.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.VectorCourt.APIKey, cfg.VectorCourt.APIKey)
	}
	if loaded.VectorCourt.Endpoint != cfg.VectorCourt.Endpoint {
		t.Errorf("Endpoint = %q, want %q", loaded.VectorCourt.Endpoint, cfg.VectorCourt.Endpoint)
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.VectorCourt.APIKey != "" {
		t.Errorf("expected empty APIKey, got %q", cfg.VectorCourt.APIKey)
	}
}

func TestSetGet(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	if err := Set("vectorcourt.api_key", "my_key"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	if err := Set("vectorcourt.endpoint", "https://local.test"); err != nil {
		t.Fatalf("Set endpoint: %v", err)
	}

	key, err := Get("vectorcourt.api_key")
	if err != nil {
		t.Fatalf("Get api_key: %v", err)
	}
	if key != "my_key" {
		t.Errorf("api_key = %q, want %q", key, "my_key")
	}

	ep, err := Get("vectorcourt.endpoint")
	if err != nil {
		t.Fatalf("Get endpoint: %v", err)
	}
	if ep != "https://local.test" {
		t.Errorf("endpoint = %q, want %q", ep, "https://local.test")
	}
}

func TestGetDefaultEndpoint(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	ep, err := Get("vectorcourt.endpoint")
	if err != nil {
		t.Fatalf("Get endpoint: %v", err)
	}
	if ep != DefaultEndpoint() {
		t.Errorf("endpoint = %q, want default %q", ep, DefaultEndpoint())
	}
}

func TestSetUnknownKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	err := Set("unknown.key", "value")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestGetUnknownKey(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	_, err := Get("unknown.key")
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
}

func TestFilePermissions(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	if err := Save(&Config{}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(filepath.Join(tmp, configFileName))
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("file perms = %o, want 0600", perm)
	}
}

func TestEndpointMethod(t *testing.T) {
	cfg := &Config{}
	if cfg.Endpoint() != DefaultEndpoint() {
		t.Errorf("empty config endpoint = %q, want %q", cfg.Endpoint(), DefaultEndpoint())
	}

	cfg.VectorCourt.Endpoint = "https://custom.test"
	if cfg.Endpoint() != "https://custom.test" {
		t.Errorf("custom endpoint = %q, want %q", cfg.Endpoint(), "https://custom.test")
	}
}

func TestLegacyOraculMigration(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	// Write a legacy config with "oracul" key.
	legacyCfg := map[string]interface{}{
		"oracul": map[string]string{
			"api_key":  "oracul_pro_legacy123",
			"endpoint": "https://oracul.app",
		},
	}
	data, _ := json.MarshalIndent(legacyCfg, "", "  ")
	if err := os.WriteFile(filepath.Join(tmp, configFileName), data, 0600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.VectorCourt.APIKey != "oracul_pro_legacy123" {
		t.Errorf("migrated APIKey = %q, want legacy key", cfg.VectorCourt.APIKey)
	}
	if cfg.VectorCourt.Endpoint != "https://oracul.app" {
		t.Errorf("migrated Endpoint = %q, want legacy endpoint", cfg.VectorCourt.Endpoint)
	}
}
