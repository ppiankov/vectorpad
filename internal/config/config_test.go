package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	cfg := &Config{
		Oracul: OraculConfig{
			APIKey:   "oracul_test_key",
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

	if loaded.Oracul.APIKey != cfg.Oracul.APIKey {
		t.Errorf("APIKey = %q, want %q", loaded.Oracul.APIKey, cfg.Oracul.APIKey)
	}
	if loaded.Oracul.Endpoint != cfg.Oracul.Endpoint {
		t.Errorf("Endpoint = %q, want %q", loaded.Oracul.Endpoint, cfg.Oracul.Endpoint)
	}
}

func TestLoadMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.Oracul.APIKey != "" {
		t.Errorf("expected empty APIKey, got %q", cfg.Oracul.APIKey)
	}
}

func TestSetGet(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("VECTORPAD_HOME", tmp)

	if err := Set("oracul.api_key", "my_key"); err != nil {
		t.Fatalf("Set api_key: %v", err)
	}
	if err := Set("oracul.endpoint", "https://local.test"); err != nil {
		t.Fatalf("Set endpoint: %v", err)
	}

	key, err := Get("oracul.api_key")
	if err != nil {
		t.Fatalf("Get api_key: %v", err)
	}
	if key != "my_key" {
		t.Errorf("api_key = %q, want %q", key, "my_key")
	}

	ep, err := Get("oracul.endpoint")
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

	ep, err := Get("oracul.endpoint")
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

	cfg.Oracul.Endpoint = "https://custom.test"
	if cfg.Endpoint() != "https://custom.test" {
		t.Errorf("custom endpoint = %q, want %q", cfg.Endpoint(), "https://custom.test")
	}
}
