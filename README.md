# fileutils [![Build Status](https://github.com/go-pkgz/fileutils/workflows/build/badge.svg)](https://github.com/go-pkgz/fileutils/actions) [![Go Report Card](https://goreportcard.com/badge/github.com/go-pkgz/fileutils)](https://goreportcard.com/report/github.com/go-pkgz/fileutils) [![Coverage Status](https://coveralls.io/repos/github/go-pkgz/fileutils/badge.svg?branch=master)](https://coveralls.io/github/go-pkgz/fileutils?branch=master)

Package `fileutils` provides useful, high-level file operations.

## Details

- `IsFile` and `IsDir` check whether a file or directory exists
- `CopyFile` copies a file from source to destination, preserving its mode, and refuses to copy a file onto itself
- `CopyDir` copies all files recursively from the source to the destination directory
- `MoveFile` moves a file, using atomic rename when possible with a copy-and-delete fallback
- `ListFiles` returns a sorted slice of file paths in a directory
- `TempFileName` returns a new temporary file name using secure random generation
- `SanitizePath` cleans a file path
- `TouchFile` creates an empty file or updates the timestamps of an existing one
- `Checksum` calculates a file checksum using MD5, SHA-1, SHA-2 and related algorithms
- `FileWatcher` watches files or directories for changes
- `WatchRecursive` watches a directory recursively for changes

## Complete example

The following program copies and moves a file, checks what exists, calculates MD5 and SHA-256 checksums, reserves a temporary name, and then runs both watcher variants until a change is reported. It is also tracked as [`examples/basic/main.go`](examples/basic/main.go).

<!-- fileutils-example-start -->
```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/go-pkgz/fileutils"
	"github.com/go-pkgz/fileutils/enum"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	workDir, err := os.MkdirTemp("", "fileutils-example-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(workDir) }()

	source := filepath.Join(workDir, "source.txt")
	if writeErr := os.WriteFile(source, []byte("fileutils example\n"), 0o600); writeErr != nil {
		return writeErr
	}

	// copy the file, then move the copy into a directory that does not exist yet
	copied := filepath.Join(workDir, "copied.txt")
	if copyErr := fileutils.CopyFile(source, copied); copyErr != nil {
		return copyErr
	}
	moved := filepath.Join(workDir, "archive", "moved.txt")
	if moveErr := fileutils.MoveFile(copied, moved); moveErr != nil {
		return moveErr
	}
	fmt.Printf("IsFile(moved.txt): %v\n", fileutils.IsFile(moved))
	fmt.Printf("IsDir(archive): %v\n", fileutils.IsDir(filepath.Dir(moved)))

	// any algorithm from the enum package works the same way
	md5sum, err := fileutils.Checksum(moved, enum.HashAlgMD5)
	if err != nil {
		return err
	}
	sha256sum, err := fileutils.Checksum(moved, enum.HashAlgSHA256)
	if err != nil {
		return err
	}
	fmt.Printf("MD5: %s\n", md5sum)
	fmt.Printf("SHA-256: %s\n", sha256sum)

	// a free name to write to, the file itself is not created
	tempName, err := fileutils.TempFileName(workDir, "upload-*.tmp")
	if err != nil {
		return err
	}
	fmt.Printf("temp name: %s\n", filepath.Base(tempName))

	return watch(workDir, source)
}

// watch starts both watcher variants, makes a change and waits for it to be reported.
func watch(workDir, source string) error {
	// the callback runs on the watcher goroutine, so hand the event over instead of working here
	events := make(chan fileutils.FileEvent, 16)
	handleEvent := func(event fileutils.FileEvent) {
		select {
		case events <- event:
		default: // the buffer is enough for this example, drop the rest
		}
	}

	fileWatcher, err := fileutils.NewFileWatcher(source, handleEvent)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := fileWatcher.Close(); closeErr != nil {
			log.Printf("close file watcher: %v", closeErr)
		}
	}()

	// paths can be added and removed while the watcher runs
	extra := filepath.Join(workDir, "extra.txt")
	if touchErr := fileutils.TouchFile(extra); touchErr != nil {
		return touchErr
	}
	if addErr := fileWatcher.AddPath(extra); addErr != nil {
		return addErr
	}
	if removeErr := fileWatcher.RemovePath(extra); removeErr != nil {
		return removeErr
	}

	recursiveWatcher, err := fileutils.WatchRecursive(workDir, handleEvent)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := recursiveWatcher.Close(); closeErr != nil {
			log.Printf("close recursive watcher: %v", closeErr)
		}
	}()

	// the paths are registered before the constructors return, so this change is already watched
	if changeErr := os.WriteFile(source, []byte("changed\n"), 0o600); changeErr != nil {
		return changeErr
	}

	select {
	case event := <-events:
		fmt.Printf("Event: %s, path: %s\n", event.Type, event.Path)
	case <-time.After(5 * time.Second):
		fmt.Println("no event within the timeout")
	}
	return nil
}
```
<!-- fileutils-example-end -->

To run this exact program from a repository checkout:

```sh
git clone https://github.com/go-pkgz/fileutils.git
cd fileutils
go mod download
go run ./examples/basic
```

To copy it into a new module instead, save the program as `main.go` and run:

```sh
mkdir fileutils-example
cd fileutils-example
go mod init example.com/fileutils-example
go get github.com/go-pkgz/fileutils@latest
go run .
```

## Install and update

```sh
go get github.com/go-pkgz/fileutils@latest
```
