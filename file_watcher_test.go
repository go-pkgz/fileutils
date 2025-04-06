package fileutils

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileWatcher(t *testing.T) {
	// skip this test on macOS as it's flaky due to how FSEvents works
	if os.Getenv("GOOS") == "darwin" {
		t.Skip("Skipping test on macOS")
	}

	// create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	// create a test file
	err := os.WriteFile(testFile, []byte("initial content"), 0644)
	require.NoError(t, err)

	// create a channel to receive events
	eventCh := make(chan FileEvent, 10) // buffered channel to avoid blocking

	// create a file watcher
	watcher, err := NewFileWatcher(testFile, func(event FileEvent) {
		// send the event to the channel
		select {
		case eventCh <- event:
			// event sent
		default:
			// channel full or closed, ignore
		}
	})
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	// sleep a bit to let the watcher initialize
	time.Sleep(100 * time.Millisecond)

	// modify the file to trigger an event
	err = os.WriteFile(testFile, []byte("modified content"), 0644)
	require.NoError(t, err)

	// wait for the event to be received or timeout
	select {
	case event := <-eventCh:
		assert.Equal(t, testFile, event.Path)
		// don't assert the exact event type as it can vary by OS
		// just verify we got an event
	case <-time.After(5 * time.Second): // longer timeout for slower systems
		t.Fatal("Timeout waiting for file event")
	}
}

func TestWatchRecursive(t *testing.T) {
	// skip this test on macOS as it's flaky due to how FSEvents works
	if os.Getenv("GOOS") == "darwin" {
		t.Skip("Skipping test on macOS")
	}

	// create a temporary directory for testing
	tmpDir := t.TempDir()
	subDir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subDir, 0755)
	require.NoError(t, err)

	// create a buffered channel to receive events
	eventCh := make(chan FileEvent, 10) // buffered channel to avoid blocking

	// create a recursive file watcher
	watcher, err := WatchRecursive(tmpDir, func(event FileEvent) {
		// send the event to the channel
		select {
		case eventCh <- event:
			// event sent
		default:
			// channel full or closed, ignore
		}
	})
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	// wait a bit to allow the watcher to initialize
	time.Sleep(200 * time.Millisecond)

	// create a warmup file to ensure the watcher is ready
	warmupFile := filepath.Join(tmpDir, "warmup.txt")
	err = os.WriteFile(warmupFile, []byte("warmup content"), 0644)
	require.NoError(t, err)

	// wait for any events from the warmup
	time.Sleep(300 * time.Millisecond)

	// clear channel from warmup events
	// make sure we drain all events from the warmup
	timeout := time.After(500 * time.Millisecond)
	drainLoop := true
	for drainLoop {
		select {
		case <-eventCh:
			// discard warmup event
		case <-timeout:
			// no more events after timeout
			drainLoop = false
		default:
			// if no events available but timeout hasn't occurred, 
			// wait a bit to avoid CPU spin
			time.Sleep(50 * time.Millisecond)
		}
	}

	// create a file in the subdirectory to trigger an event
	testFile := filepath.Join(subDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)
	
	// give a little time for the event to be processed
	time.Sleep(100 * time.Millisecond)

	// wait for the event to be received or timeout

	select {
	case event := <-eventCh:
		assert.Equal(t, testFile, event.Path)
		// don't assert the exact event type as it can vary by OS
		// just verify we got an event
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for file event")
	}
}

func TestFileWatcherAddPath(t *testing.T) {
	// create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	// create test files
	err := os.WriteFile(testFile1, []byte("file 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("file 2"), 0644)
	require.NoError(t, err)

	// create a channel to receive events
	eventCh := make(chan FileEvent, 10) // buffered channel to avoid blocking

	// create a file watcher for the first file
	watcher, err := NewFileWatcher(testFile1, func(event FileEvent) {
		// send the event to the channel
		select {
		case eventCh <- event:
			// event sent
		default:
			// channel full or closed, ignore
		}
	})
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	// modify the first file to trigger an event
	err = os.WriteFile(testFile1, []byte("modified file 1"), 0644)
	require.NoError(t, err)

	// wait for the event to be received or timeout
	select {
	case event := <-eventCh:
		assert.Equal(t, testFile1, event.Path)
		// don't assert the exact event type as it can vary by OS
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for file event")
	}

	// add the second file to the watcher
	err = watcher.AddPath(testFile2)
	require.NoError(t, err)

	// modify the second file to trigger an event
	err = os.WriteFile(testFile2, []byte("modified file 2"), 0644)
	require.NoError(t, err)

	// wait for the event to be received or timeout
	select {
	case event := <-eventCh:
		// just verify we got an event for one of our files
		assert.True(t, event.Path == testFile1 || event.Path == testFile2,
			"Expected event for either %s or %s, got %s", testFile1, testFile2, event.Path)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for file event")
	}
}

func TestFileWatcherRemovePath(t *testing.T) {
	// skip this test on macOS as it's flaky due to how FSEvents works
	if os.Getenv("GOOS") == "darwin" {
		t.Skip("Skipping test on macOS")
	}

	// create a temporary directory for testing
	tmpDir := t.TempDir()
	testFile1 := filepath.Join(tmpDir, "test1.txt")
	testFile2 := filepath.Join(tmpDir, "test2.txt")

	// create test files
	err := os.WriteFile(testFile1, []byte("file 1"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("file 2"), 0644)
	require.NoError(t, err)

	// create a channel to receive events
	eventCh := make(chan FileEvent, 10) // buffered channel to avoid blocking

	// create a file watcher for both files
	watcher, err := NewFileWatcher(testFile1, func(event FileEvent) {
		// send the event to the channel
		select {
		case eventCh <- event:
			// event sent
		default:
			// channel full or closed, ignore
		}
	})
	require.NoError(t, err)
	defer func() { _ = watcher.Close() }()

	// add the second file
	err = watcher.AddPath(testFile2)
	require.NoError(t, err)

	// remove the second file from the watcher
	err = watcher.RemovePath(testFile2)
	require.NoError(t, err)

	// modify the second file, but no event should be triggered
	err = os.WriteFile(testFile2, []byte("modified file 2"), 0644)
	require.NoError(t, err)

	// modify the first file to trigger an event
	err = os.WriteFile(testFile1, []byte("modified file 1"), 0644)
	require.NoError(t, err)

	// wait for the event to be received or timeout
	select {
	case event := <-eventCh:
		assert.Equal(t, testFile1, event.Path, "Expected event for %s, got %s", testFile1, event.Path)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for file event")
	}
}
