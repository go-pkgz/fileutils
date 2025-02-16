package fileutils

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExistsFile(t *testing.T) {
	assert.True(t, IsFile("testfiles/file1.txt"))
	assert.False(t, IsFile("testfiles/file-not-found.txt"))
	assert.False(t, IsFile(""))
	assert.False(t, IsFile(".."))
	assert.False(t, IsFile("testfiles"))
}

func TestExistsDir(t *testing.T) {
	assert.True(t, IsDir("testfiles"))
	assert.False(t, IsDir("testfiles/file1.txt"))
	assert.False(t, IsDir(""))
	assert.True(t, IsDir(".."))
	assert.True(t, IsDir("."))
	assert.False(t, IsDir("testfiles-nop"))
}

func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()

	// create source file with specific mode
	srcFile := filepath.Join(tmpDir, "src.txt")
	err := os.WriteFile(srcFile, []byte("test content"), 0o600)
	require.NoError(t, err)

	// get source info for comparison
	srcInfo, err := os.Stat(srcFile)
	require.NoError(t, err)

	// copy file
	dstFile := filepath.Join(tmpDir, "dst.txt")
	err = CopyFile(srcFile, dstFile)
	require.NoError(t, err)

	// verify content
	content, err := os.ReadFile(dstFile) //nolint:gosec
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))

	// verify mode
	dstInfo, err := os.Stat(dstFile)
	require.NoError(t, err)
	assert.Equal(t, srcInfo.Mode(), dstInfo.Mode())

	// verify error cases
	err = CopyFile("notfound.txt", dstFile)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "can't stat")

	err = CopyFile(srcFile, "/dev/null")
	assert.Error(t, err)
}

func TestListFiles(t *testing.T) {
	list, err := ListFiles("testfiles")
	require.NoError(t, err)
	assert.Equal(t, []string{"testfiles/d1/d21/file21_d21.txt", "testfiles/d1/d21/file22_d21.txt",
		"testfiles/d1/file1_d1.txt", "testfiles/file1.txt"}, list)

	_, err = ListFiles("testfiles.bad")
	assert.Error(t, err)
}

func TestCopyDir(t *testing.T) {
	defer func() { _ = os.RemoveAll("/tmp/copydir.test") }()

	err := CopyDir("testfiles", "/tmp/copydir.test")
	require.NoError(t, err)

	list, err := ListFiles("/tmp/copydir.test")
	assert.NoError(t, err)
	assert.Equal(t, []string{"/tmp/copydir.test/d1/d21/file21_d21.txt", "/tmp/copydir.test/d1/d21/file22_d21.txt",
		"/tmp/copydir.test/d1/file1_d1.txt", "/tmp/copydir.test/file1.txt"}, list)

	err = CopyDir("testfiles-no", "/tmp/copydir.test")
	assert.Error(t, err)

	err = CopyDir("testfiles", "/dev/null")
	assert.Error(t, err)
	t.Log(err)
}

func TestTempFileName(t *testing.T) {
	r1, err := TempFileName("", "something-*.txt")
	require.NoError(t, err)
	t.Log(r1)
	assert.Contains(t, r1, "something-")
	assert.Contains(t, r1, ".txt")

	r2, err := TempFileName("", "something-*.txt")
	require.NoError(t, err)
	t.Log(r2)
	assert.NotEqual(t, r1, r2)

	r3, err := TempFileName("somedir", "something-*.txt")
	require.NoError(t, err)
	t.Log(r3)
	assert.True(t, strings.HasPrefix(r3, "somedir/"))

	r4, err := TempFileName("somedir", "something-")
	require.NoError(t, err)
	t.Log(r4)
	assert.True(t, strings.HasPrefix(r4, "somedir/something-"))
}

func TestSanitizePath(t *testing.T) {
	tbl := []struct {
		inp, out string
	}{
		{"aaaa", "aaaa"},
		{"aaaa?bb", "aaaa_bb"},
		{"aaaa/bb", "aaaa/bb"},
		{"aaaa?*bb", "aaaa_bb"},
		{"aa*aa?*bb", "aa_aa_bb"},
		{"aa>aa<bb", "aa_aa_bb"},
		{"path/to/file.txt", "path/to/file.txt"},
		{"  path/to/file.txt   ", "path/to/file.txt"},
		{"path<>to|file?.txt", "path_to_file_.txt"},
		{strings.Repeat("a", maxPathLength+10), strings.Repeat("a", maxPathLength)},
		{"path\\to/file.txt", "path/to/file.txt"},
		{"con/nul", "con/nul"},
	}

	for i, tt := range tbl {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			assert.Equal(t, tt.out, SanitizePath(tt.inp))
		})
	}
}

func TestMoveFile(t *testing.T) {
	t.Run("same device move", func(t *testing.T) {
		// create temp source file
		srcFile := filepath.Join(os.TempDir(), "move_test_src.txt")
		err := os.WriteFile(srcFile, []byte("test content"), 0600)
		require.NoError(t, err)
		defer os.Remove(srcFile)

		// create temp destination
		dstFile := filepath.Join(os.TempDir(), "move_test_dst.txt")
		defer os.Remove(dstFile)

		// perform move
		err = MoveFile(srcFile, dstFile)
		require.NoError(t, err)

		// verify source is gone and destination exists
		_, err = os.Stat(srcFile)
		assert.True(t, os.IsNotExist(err), "source file should not exist")

		content, err := os.ReadFile(dstFile) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})

	t.Run("move with copy fallback", func(t *testing.T) {
		// create source dir and file
		srcDir := t.TempDir()
		srcFile := filepath.Join(srcDir, "move_test_src2.txt")
		err := os.WriteFile(srcFile, []byte("test content"), 0600)
		require.NoError(t, err)

		// create destination dir
		dstDir := t.TempDir()
		dstFile := filepath.Join(dstDir, "subdir", "move_test_dst.txt")

		// perform move
		err = MoveFile(srcFile, dstFile)
		require.NoError(t, err)

		// verify move succeeded
		_, err = os.Stat(srcFile)
		assert.True(t, os.IsNotExist(err), "source file should not exist")

		content, err := os.ReadFile(dstFile) //nolint:gosec
		require.NoError(t, err)
		assert.Equal(t, "test content", string(content))
	})

	t.Run("errors", func(t *testing.T) {
		tests := []struct {
			name    string
			src     string
			dst     string
			wantErr string
		}{
			{
				name:    "source not found",
				src:     "notfound.txt",
				dst:     "dst.txt",
				wantErr: "source file not found",
			},
			{
				name:    "empty source",
				src:     "",
				dst:     "dst.txt",
				wantErr: "empty source path",
			},
			{
				name:    "empty destination",
				src:     "src.txt",
				dst:     "",
				wantErr: "empty destination path",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				err := MoveFile(tt.src, tt.dst)
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			})
		}
	})
}

func TestTouchFile(t *testing.T) {
	t.Run("create new", func(t *testing.T) {
		tmpDir := t.TempDir()
		newFile := filepath.Join(tmpDir, "new.txt")

		err := TouchFile(newFile)
		require.NoError(t, err)

		info, err := os.Stat(newFile)
		require.NoError(t, err)
		assert.Equal(t, int64(0), info.Size())
		assert.True(t, time.Since(info.ModTime()) < time.Second)
	})

	t.Run("update existing", func(t *testing.T) {
		tmpDir := t.TempDir()
		existingFile := filepath.Join(tmpDir, "existing.txt")

		err := os.WriteFile(existingFile, []byte("test"), 0600)
		require.NoError(t, err)

		// get original time and wait a bit
		origInfo, err := os.Stat(existingFile)
		require.NoError(t, err)
		time.Sleep(time.Millisecond * 100)

		err = TouchFile(existingFile)
		require.NoError(t, err)

		// check content preserved and time updated
		info, err := os.Stat(existingFile)
		require.NoError(t, err)
		assert.Equal(t, origInfo.Size(), info.Size())
		assert.True(t, info.ModTime().After(origInfo.ModTime()))
	})

	t.Run("errors", func(t *testing.T) {
		err := TouchFile("")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty path")

		err = TouchFile("/dev/null/invalid")
		require.Error(t, err)
	})
}
