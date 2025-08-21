package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/knadh/koanf/v2"
)

// writeTempConfig writes the provided YAML string to a temp file and returns its path.
func writeTempConfig(t *testing.T, dir, yaml string) string {
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

func TestExpandEnvValues_SuccessAndNoPartialExpansion(t *testing.T) {
	k := koanf.NewWithConf(conf)
	// Set environment variables
	os.Setenv("GITHUB_TOKEN", "pat-token")
	os.Setenv("APP_NAME", "Skeleton Template")
	t.Cleanup(func() {
		os.Unsetenv("GITHUB_TOKEN")
		os.Unsetenv("APP_NAME")
	})

	// Set some config values
	_ = k.Set("providers.0.token", "${GITHUB_TOKEN}")
	_ = k.Set("identifier.name", "${APP_NAME}")
	_ = k.Set("git.targetBranch", "main")
	_ = k.Set("files.include.0", "ci")
	_ = k.Set("files.exclude.0", "prefix-${GITHUB_TOKEN}") // should not be expanded (partial)

	if err := expandEnvValues(k); err != nil {
		t.Fatalf("expandEnvValues returned error: %v", err)
	}

	if got := k.String("providers.0.token"); got != "pat-token" {
		t.Fatalf("providers.0.token = %q, want %q", got, "pat-token")
	}
	if got := k.String("identifier.name"); got != "Skeleton Template" {
		t.Fatalf("identifier.name = %q, want %q", got, "Skeleton Template")
	}
	if got := k.String("files.exclude.0"); got != "prefix-${GITHUB_TOKEN}" {
		t.Fatalf("files.exclude.0 = %q, want %q (unchanged)", got, "prefix-${GITHUB_TOKEN}")
	}
}

func TestExpandEnvValues_MissingEnvCausesError(t *testing.T) {
	k := koanf.NewWithConf(conf)
	_ = k.Set("providers.0.token", "${MISSING}")
	os.Unsetenv("MISSING")
	if err := expandEnvValues(k); err == nil {
		t.Fatalf("expected error for missing env, got nil")
	}
}

func TestExpandEnvValues_InvalidNAValueCausesError(t *testing.T) {
	k := koanf.NewWithConf(conf)
	_ = k.Set("identifier.name", "${APP_NAME}")
	os.Setenv("APP_NAME", "n/a")
	t.Cleanup(func() { os.Unsetenv("APP_NAME") })
	if err := expandEnvValues(k); err == nil {
		t.Fatalf("expected error for env value 'n/a', got nil")
	}
}

func TestExpandEnvValues_InvalidPatternIsIgnored(t *testing.T) {
	k := koanf.NewWithConf(conf)
	_ = k.Set("git.name", "${$bad}") // does not match allowed pattern, should be left as-is
	if err := expandEnvValues(k); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := k.String("git.name"); got != "${$bad}" {
		t.Fatalf("git.name = %q, want %q", got, "${$bad}")
	}
}
