package git

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsExcluded(t *testing.T) {
	excluded := []string{"ci/mocks.sh", "README.md"}
	if !isExcluded("ci/mocks.sh", excluded) {
		t.Fatalf("expected ci/mocks.sh to be excluded")
	}
	if isExcluded("ci/build.sh", excluded) {
		t.Fatalf("did not expect ci/build.sh to be excluded")
	}
}

func TestGetAllFilenames_DirAndFile(t *testing.T) {
	dir := t.TempDir()

	// Create nested structure: dir/a.txt, dir/sub/b.txt
	a := filepath.Join(dir, "a.txt")
	sub := filepath.Join(dir, "sub")
	b := filepath.Join(sub, "b.txt")

	if err := os.WriteFile(a, []byte("a"), 0o644); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("mkdir sub: %v", err)
	}
	if err := os.WriteFile(b, []byte("b"), 0o644); err != nil {
		t.Fatalf("write b: %v", err)
	}

	// Call with directory
	files, err := getAllFilenames(dir)
	if err != nil {
		t.Fatalf("getAllFilenames dir: %v", err)
	}

	// Expect exactly the two files we created
	// Order is not guaranteed, so compare as a set
	want := map[string]bool{a: true, b: true}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
	for _, f := range files {
		if !want[f] {
			t.Fatalf("unexpected file listed: %s", f)
		}
	}

	// Call with single file
	files2, err := getAllFilenames(a)
	if err != nil {
		t.Fatalf("getAllFilenames file: %v", err)
	}
	if len(files2) != 1 || files2[0] != a {
		t.Fatalf("expected single file %s, got %v", a, files2)
	}
}
