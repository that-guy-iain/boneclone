package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
)

// writeTempConfig writes the provided YAML string to a temp file and returns its path.
func writeTempConfig(t *testing.T, dir string, yaml string) string {
	t.Helper()
	p := filepath.Join(dir, ".boneclone.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}
	return p
}

func TestRun_WithExplicitConfigFlag_NoProviders(t *testing.T) {
	// Reset global koanf instance to avoid cross-test state.
	k = koanf.NewWithConf(conf)

	dir := t.TempDir()
	cfg := "providers: []\n"
	cfgPath := writeTempConfig(t, dir, cfg)

	if err := runWithArgs([]string{"boneclone", "-c", cfgPath}); err != nil {
		t.Fatalf("runWithArgs returned error: %v", err)
	}
}

func TestRun_WithDefaultConfigPath_NoProviders(t *testing.T) {
	// Reset global koanf instance to avoid cross-test state.
	k = koanf.NewWithConf(conf)

	dir := t.TempDir()
	_ = writeTempConfig(t, dir, "providers: []\n")

	// Change CWD so default .boneclone.yaml is found
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd failed: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir failed: %v", err)
	}

	if err := runWithArgs([]string{"boneclone"}); err != nil {
		t.Fatalf("runWithArgs returned error: %v", err)
	}
}
