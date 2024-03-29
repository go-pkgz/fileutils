// Package fileutils provides useful, high-level file operations
package fileutils

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var once sync.Once

// IsFile returns true if filename exists
func IsFile(filename string) bool {
	return exists(filename, false)
}

// IsDir returns true if directory exists
func IsDir(dirname string) bool {
	return exists(dirname, true)
}

func exists(name string, dir bool) bool {
	info, err := os.Stat(name)
	if os.IsNotExist(err) {
		return false
	}
	if dir {
		return info.IsDir()
	}
	return !info.IsDir()
}

// CopyFile copies a file from source to dest. Any existing file will be overwritten
// and attributes will not be copied
func CopyFile(src string, dst string) error {

	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("can't stat %s: %w", src, err)
	}

	if !srcInfo.Mode().IsRegular() {
		return fmt.Errorf("can't copy non-regular source file %s (%s)", src, srcInfo.Mode().String())
	}

	srcFh, err := os.Open(src) //nolint
	if err != nil {
		return fmt.Errorf("can't open source file %s: %w", src, err)
	}
	defer srcFh.Close() //nolint

	err = os.MkdirAll(filepath.Dir(dst), 0750)
	if err != nil {
		return fmt.Errorf("can't make destination directory %s: %w", filepath.Dir(dst), err)
	}

	dstFh, err := os.Create(dst) //nolint
	if err != nil {
		return fmt.Errorf("can't create destination file %s: %w", dst, err)
	}
	defer dstFh.Close() //nolint

	size, err := io.Copy(dstFh, srcFh)
	if err != nil {
		return fmt.Errorf("can't copy data: %w", err)
	}
	if size != srcInfo.Size() {
		return fmt.Errorf("incomplete copy, %d of %d", size, srcInfo.Size())
	}
	return dstFh.Sync()
}

// CopyDir copies all files from src to dst, recursively
func CopyDir(src string, dst string) error {
	list, err := ListFiles(src)
	if err != nil {
		return fmt.Errorf("can't list source files in %s: %w", src, err)
	}
	for _, srcFile := range list {
		stripSrcDir := strings.TrimPrefix(srcFile, src)
		dstFile := filepath.Join(dst, stripSrcDir)
		if err = CopyFile(srcFile, dstFile); err != nil {
			return fmt.Errorf("can't copy %s to %s: %w", srcFile, dstFile, err)
		}
	}
	return nil
}

// ListFiles gets recursive list of all files in a directory
func ListFiles(directory string) (list []string, err error) {
	err = filepath.Walk(directory, func(path string, info os.FileInfo, e error) error {
		if e != nil {
			return e
		}
		if info.IsDir() {
			return nil
		}
		list = append(list, path)
		return nil
	})
	sort.Slice(list, func(i, j int) bool {
		return list[i] < list[j]
	})
	return list, err
}

// TempFileName returns a new temporary file name in the directory dir.
// The filename is generated by taking pattern and adding a random
// string to the end. If pattern includes a "*", the random string
// replaces the last "*".
// If dir is the empty string, TempFileName uses the default directory
// for temporary files (see os.TempDir).
// Multiple programs calling TempFileName simultaneously
// will not choose the same file name.
// some code borrowed from stdlib https://golang.org/src/io/ioutil/tempfile.go
func TempFileName(dir, pattern string) (string, error) {
	once.Do(func() {
		rand.Seed(time.Now().UnixNano())
	})
	// prefixAndSuffix splits pattern by the last wildcard "*", if applicable,
	// returning prefix as the part before "*" and suffix as the part after "*".
	prefixAndSuffix := func(pattern string) (prefix, suffix string) {
		if pos := strings.LastIndex(pattern, "*"); pos != -1 {
			prefix, suffix = pattern[:pos], pattern[pos+1:]
		} else {
			prefix = pattern
		}
		return
	}

	if dir == "" {
		dir = os.TempDir()
	}

	prefix, suffix := prefixAndSuffix(pattern)

	for i := 0; i < 10000; i++ {
		name := filepath.Join(dir, prefix+fmt.Sprintf("%x", rand.Int())+suffix) //nolint
		_, err := os.Stat(name)
		if os.IsNotExist(err) {
			return name, nil
		}
	}
	return "", errors.New("can't generate temp file name")
}

var reInvalidPathChars = regexp.MustCompile(`[<>:"|?*]+`) // invalid path characters
const maxPathLength = 1024                                // maximum length for path

// SanitizePath returns a sanitized version of the given path.
func SanitizePath(s string) string {
	s = strings.TrimSpace(s)
	s = reInvalidPathChars.ReplaceAllString(filepath.Clean(s), "_")

	// Normalize path separators to '/'
	s = strings.ReplaceAll(s, `\`, "/")

	if len(s) > maxPathLength {
		s = s[:maxPathLength]
	}

	return s
}
