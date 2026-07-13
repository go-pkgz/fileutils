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
