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

	"github.com/go-pkgz/fileutils/enum"
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
		defer func() { _ = os.Remove(srcFile) }()

		// create temp destination
		dstFile := filepath.Join(os.TempDir(), "move_test_dst.txt")
		defer func() { _ = os.Remove(dstFile) }()

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
func TestMkdDir(t *testing.T) {
	t.Run("make dir[success]", func(t *testing.T) {
		tmpDir := t.TempDir()
		newDir := filepath.Join(tmpDir, "dir")

		err := MkDir(newDir)
		require.NoError(t, err)

		assert.True(t, IsDir(newDir))
	})

	t.Run("make existing dir", func(t *testing.T) {
		tmpDir := t.TempDir()
		newDir := filepath.Join(tmpDir, "dir")
		errMkDir := os.Mkdir(newDir, 0750)

		require.NoError(t, errMkDir)

		assert.True(t, IsDir(newDir))

		err := MkDir(newDir)
		require.NoError(t, err)
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
func TestChecksum(t *testing.T) {
	// create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "checksum_test.txt")
	content := []byte("this is a test file for checksum calculation")
	err := os.WriteFile(testFile, content, 0600)
	require.NoError(t, err)

	// expected checksums (re-calculated)
	expectedMD5 := "656b12fec36f7df11771b03c53e177ba"
	expectedSHA1 := "be4c9cf3936f6d20ee0f38637605a405ea831168"
	expectedSHA224 := "098477fd72b5128aa051cbd0a09010d14b3dd18114d66b061f0ff382"
	expectedSHA384 := "a4de83d650a8a4d07483ad61296685d9d261e6edb940a025b8b981f90c17bdca794d45f202b2b8c3de5cd9c9bcf5e1e0"
	expectedSHA512 := "a182278014c2a2d6d00de8442ab0e358e689269965eea6dfb7761abe019d0c34d47c181e19f1021901a5c0cf65b82871a0fa36b8bb187f9f5bf97ed182798e9e"
	expectedSHA512_224 := "aab1745f6f1464c67c4a46d29c3a79132afe6d104f96a1266a69a278"
	expectedSHA512_256 := "b5bc9721b180d5c79264f5fbb61404b516b6bfcb486c95b65329a1fe71ff6728"
	expectedSHA256 := "7644ba794d6c4df31bd440ea9f7ecbcb8f2f3846cc58fcbf55d13560e168c863"

	t.Run("md5", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgMD5)
		require.NoError(t, err)
		assert.Equal(t, expectedMD5, checksum)
	})

	t.Run("sha1", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA1)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA1, checksum)
	})

	t.Run("sha256", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA256)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA256, checksum)
	})

	t.Run("sha224", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA224)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA224, checksum)
	})

	t.Run("sha384", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA384)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA384, checksum)
	})

	t.Run("sha512", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA512)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA512, checksum)
	})

	t.Run("sha512_224", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA512_224)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA512_224, checksum)
	})

	t.Run("sha512_256", func(t *testing.T) {
		checksum, err := Checksum(testFile, enum.HashAlgSHA512_256)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA512_256, checksum)
	})

	t.Run("parse from string", func(t *testing.T) {
		// test parsing from string
		alg, err := enum.ParseHashAlg("sha256")
		require.NoError(t, err)
		checksum, err := Checksum(testFile, alg)
		require.NoError(t, err)
		assert.Equal(t, expectedSHA256, checksum)
	})

	t.Run("errors", func(t *testing.T) {
		_, err := Checksum("nonexistent.txt", enum.HashAlgMD5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "file not found")

		// test invalid algorithm parsing
		_, err = enum.ParseHashAlg("unsupported_algo")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid hashAlg")

		_, err = Checksum("", enum.HashAlgMD5)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "empty path")
	})
}
