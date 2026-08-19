package fileutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// eventTimeout bounds how long a test waits for an event it expects to arrive.
const eventTimeout = 5 * time.Second

// newTestWatcher starts a watcher on path and returns the channel its callback feeds.
// The channel is buffered generously so no expected event is dropped while the test is busy.
func newTestWatcher(t *testing.T, path string) (*FileWatcher, <-chan FileEvent) {
	t.Helper()
	eventCh := make(chan FileEvent, 100)
	watcher, err := NewFileWatcher(path, func(event FileEvent) {
		select {
		case eventCh <- event:
		default: // buffer full, the test is not interested in this many events
		}
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = watcher.Close() })
	return watcher, eventCh
}

// waitForEvent consumes events until one for path arrives and returns everything seen before it,
// so a test can also assert on what did not show up. Fails the test if nothing arrives in time.
func waitForEvent(t *testing.T, eventCh <-chan FileEvent, path string) []FileEvent {
	t.Helper()
	var skipped []FileEvent
	deadline := time.After(eventTimeout)
	for {
		select {
		case event := <-eventCh:
			if event.Path == path {
				return skipped
			}
			skipped = append(skipped, event)
		case <-deadline:
			t.Fatalf("timeout waiting for an event on %s, saw %v", path, skipped)
			return skipped
		}
	}
}

func TestFileWatcher(t *testing.T) {
	testFile := filepath.Join(t.TempDir(), "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("initial content"), 0o600))

	// the path is registered with the kernel before NewFileWatcher returns, so events
	// raised from here on are queued even if the consuming goroutine is not scheduled yet
	_, eventCh := newTestWatcher(t, testFile)

	require.NoError(t, os.WriteFile(testFile, []byte("modified content"), 0o600))

	// the event type varies by platform, seeing the event at all is what matters
	waitForEvent(t, eventCh, testFile)
}

func TestWatchRecursive(t *testing.T) {
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0o750))

	eventCh := make(chan FileEvent, 100)
	watcher, err := WatchRecursive(tmpDir, func(event FileEvent) {
		select {
		case eventCh <- event:
		default:
		}
	})
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	testFile := filepath.Join(subDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0o600))

	// events for tmpDir itself may arrive first, waitForEvent skips past them
	waitForEvent(t, eventCh, testFile)
}

func TestFileWatcherAddPath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	require.NoError(t, os.WriteFile(testFile1, []byte("file 1"), 0o600))
	require.NoError(t, os.WriteFile(testFile2, []byte("file 2"), 0o600))

	watcher, eventCh := newTestWatcher(t, testFile1)

	require.NoError(t, os.WriteFile(testFile1, []byte("modified file 1"), 0o600))
	waitForEvent(t, eventCh, testFile1)

	require.NoError(t, watcher.AddPath(testFile2))

	require.NoError(t, os.WriteFile(testFile2, []byte("modified file 2"), 0o600))
	waitForEvent(t, eventCh, testFile2)
}

func TestFileWatcherRemovePath(t *testing.T) {
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	require.NoError(t, os.WriteFile(testFile1, []byte("file 1"), 0o600))
	require.NoError(t, os.WriteFile(testFile2, []byte("file 2"), 0o600))

	watcher, eventCh := newTestWatcher(t, testFile1)

	require.NoError(t, watcher.AddPath(testFile2))
	require.NoError(t, watcher.RemovePath(testFile2))

	// the write to the unwatched file goes first, so any event it wrongly produced
	// is already queued ahead of the one the test waits for
	require.NoError(t, os.WriteFile(testFile2, []byte("modified file 2"), 0o600))
	require.NoError(t, os.WriteFile(testFile1, []byte("modified file 1"), 0o600))

	skipped := waitForEvent(t, eventCh, testFile1)
	for _, event := range skipped {
		assert.NotEqual(t, testFile2, event.Path, "removed path still reports events")
	}
}
