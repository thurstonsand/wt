package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopyFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	writeFile(t, srcDir, "a.txt", "content a")
	writeFile(t, srcDir, "sub/b.txt", "content b")

	files := []string{"a.txt", "sub/b.txt"}
	if err := CopyFiles(srcDir, dstDir, files); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFileContent(t, dstDir, "a.txt", "content a")
	assertFileContent(t, dstDir, "sub/b.txt", "content b")
}

func TestCopyFiles_SkipsMissingSource(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	writeFile(t, srcDir, "present.txt", "here")

	if err := CopyFiles(srcDir, dstDir, []string{"present.txt", "gone.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFileContent(t, dstDir, "present.txt", "here")
	if _, err := os.Stat(filepath.Join(dstDir, "gone.txt")); !os.IsNotExist(err) {
		t.Errorf("expected missing source to be skipped, got %v", err)
	}
}

func TestCopyFiles_Overwrites(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	writeFile(t, srcDir, "conf", "new")
	writeFile(t, dstDir, "conf", "old")

	if err := CopyFiles(srcDir, dstDir, []string{"conf"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	assertFileContent(t, dstDir, "conf", "new")
}

func TestCopyFiles_PreservesMode(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	srcPath := filepath.Join(srcDir, "script.sh")
	if err := os.WriteFile(srcPath, []byte("#!/bin/bash"), 0o755); err != nil { //nolint:gosec // test file
		t.Fatal(err)
	}

	if err := CopyFiles(srcDir, dstDir, []string{"script.sh"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	dstPath := filepath.Join(dstDir, "script.sh")
	info, err := os.Stat(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode() != 0o755 {
		t.Errorf("expected mode 0755, got %o", info.Mode())
	}
}

func TestCopyFiles_InheritsDirMode(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	writeFile(t, srcDir, "sub/file.txt", "data")

	if err := CopyFiles(srcDir, dstDir, []string{"sub/file.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	subDir := filepath.Join(dstDir, "sub")
	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatal(err)
	}
	dstInfo, _ := os.Stat(dstDir)
	if info.Mode().Perm() != dstInfo.Mode().Perm() {
		t.Errorf("expected subdir to inherit parent mode %o, got %o", dstInfo.Mode().Perm(), info.Mode().Perm())
	}
}

func TestCopyFiles_ExistingDirUnchanged(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	writeFile(t, srcDir, "sub/file.txt", "data")

	subDir := filepath.Join(dstDir, "sub")
	if err := os.Mkdir(subDir, 0o700); err != nil {
		t.Fatal(err)
	}

	if err := CopyFiles(srcDir, dstDir, []string{"sub/file.txt"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	info, err := os.Stat(subDir)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		t.Errorf("expected existing dir mode 0700 to be unchanged, got %o", info.Mode().Perm())
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, dir, name, want string) {
	t.Helper()
	path := filepath.Join(dir, name)
	data, err := os.ReadFile(path) //nolint:gosec // test file
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("expected content %q, got %q", want, string(data))
	}
}
