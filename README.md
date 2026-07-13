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

The following program creates and copies a file, calculates its SHA-256 checksum, and configures both watcher variants. It is also tracked as [`examples/basic/main.go`](examples/basic/main.go).

<!-- fileutils-example-start -->
```go
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

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
	defer os.RemoveAll(workDir)

	source := filepath.Join(workDir, "source.txt")
	if writeErr := os.WriteFile(source, []byte("fileutils example\n"), 0o600); writeErr != nil {
		return writeErr
	}

	destination := filepath.Join(workDir, "copied.txt")
	if copyErr := fileutils.CopyFile(source, destination); copyErr != nil {
		return copyErr
	}

	checksum, err := fileutils.Checksum(destination, enum.HashAlgSHA256)
	if err != nil {
		return err
	}
	fmt.Printf("SHA-256: %s\n", checksum)

	handleEvent := func(event fileutils.FileEvent) {
		fmt.Printf("Event: %s, path: %s\n", event.Type, event.Path)
	}

	fileWatcher, err := fileutils.NewFileWatcher(source, handleEvent)
	if err != nil {
		return err
	}
	defer fileWatcher.Close()

	if addErr := fileWatcher.AddPath(destination); addErr != nil {
		return addErr
	}
	if removeErr := fileWatcher.RemovePath(destination); removeErr != nil {
		return removeErr
	}

	recursiveWatcher, err := fileutils.WatchRecursive(workDir, handleEvent)
	if err != nil {
		return err
	}
	defer recursiveWatcher.Close()

	fmt.Printf("Watching %s\n", workDir)
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
