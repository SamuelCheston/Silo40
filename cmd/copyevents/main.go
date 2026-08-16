package main

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func main() {
	if len(os.Args) < 2 {
		fatalf("usage: copyevents <built-binary-path>")
	}

	binPath, err := filepath.Abs(os.Args[1])
	if err != nil {
		fatalf("resolve binary path: %v", err)
	}

	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		fatalf("resolve source path failed")
	}
	projectRoot := filepath.Clean(filepath.Join(filepath.Dir(sourceFile), "..", ".."))
	sourceRoot := filepath.Join(projectRoot, "events")
	if info, err := os.Stat(sourceRoot); err != nil || !info.IsDir() {
		fatalf("events directory not found at %s", sourceRoot)
	}

	targetRoot := filepath.Join(filepath.Dir(binPath), "events")
	if err := os.RemoveAll(targetRoot); err != nil {
		fatalf("clear target events directory: %v", err)
	}
	if err := copyDir(sourceRoot, targetRoot); err != nil {
		fatalf("copy events: %v", err)
	}

	fmt.Printf("copied events to %s\n", targetRoot)
}

func copyDir(sourceRoot, targetRoot string) error {
	return filepath.WalkDir(sourceRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetRoot, relPath)

		if d.IsDir() {
			return os.MkdirAll(targetPath, 0o755)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, targetPath, info.Mode())
	})
}

func copyFile(sourcePath, targetPath string, mode fs.FileMode) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return err
	}

	targetFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode.Perm())
	if err != nil {
		return err
	}
	defer targetFile.Close()

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return err
	}
	return nil
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
