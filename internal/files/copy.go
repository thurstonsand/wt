// Package files provides file copy operations.
package files

import (
	"io"
	"os"
	"path/filepath"
)

// CopyFiles copies files from source to destination directory.
// Relative paths are preserved. Non-regular files (e.g. symlinks) are skipped,
// as are paths that no longer exist in the source.
func CopyFiles(srcDir, dstDir string, files []string) error {
	for _, f := range files {
		src := filepath.Join(srcDir, f)
		dst := filepath.Join(dstDir, f)

		info, err := os.Stat(src)
		if err != nil || !info.Mode().IsRegular() {
			continue
		}

		if err := copyFile(src, dst); err != nil {
			return err
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	srcF, err := os.Open(src) //nolint:gosec // path from trusted sources
	if err != nil {
		return err
	}
	defer func() { _ = srcF.Close() }()

	srcInfo, err := srcF.Stat()
	if err != nil {
		return err
	}

	if err = ensureDir(filepath.Dir(dst)); err != nil {
		return err
	}

	dstF, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode()) //nolint:gosec // path from trusted sources
	if err != nil {
		return err
	}
	defer func() { _ = dstF.Close() }()

	_, err = io.Copy(dstF, srcF)
	return err
}

func ensureDir(dir string) error {
	info, err := os.Stat(dir)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return &os.PathError{Op: "mkdir", Path: dir, Err: os.ErrExist}
	}
	if !os.IsNotExist(err) {
		return err
	}

	parent := filepath.Dir(dir)
	if err = ensureDir(parent); err != nil {
		return err
	}
	parentInfo, err := os.Stat(parent)
	if err != nil {
		return err
	}
	return os.Mkdir(dir, parentInfo.Mode().Perm())
}
