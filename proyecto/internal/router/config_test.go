package router

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigValidatesNoiseProbability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "name": "A",
  "ip": "127.0.0.1",
  "port": 5000,
  "noise_probability": 1.1
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatal("se esperaba error para noise_probability mayor que 1")
	}
}

func TestLoadConfigAcceptsOptionalNoiseProbability(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{
  "name": "A",
  "ip": "127.0.0.1",
  "port": 5000,
  "noise_probability": 0.01
}`), 0o600); err != nil {
		t.Fatal(err)
	}
	config, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.NoiseProbability != 0.01 {
		t.Fatalf("noise_probability=%g", config.NoiseProbability)
	}
}
