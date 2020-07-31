package fileutils

import (
	"os"
	"testing"

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
	defer os.Remove("/tmp/file1.txt")
	err := CopyFile("testfiles/file1.txt", "/tmp/file1.txt")
	require.NoError(t, err)

	fi, err := os.Stat("/tmp/file1.txt")
	assert.NoError(t, err)
	assert.Equal(t, int64(17), fi.Size())

	err = CopyFile("testfiles/file1.txt", "/tmp/file1.txt")
	assert.NoError(t, err)

	err = CopyFile("testfiles/file-not-found.txt", "/tmp/file1.txt")
	assert.Error(t, err)

	err = CopyFile("testfiles/file1.txt", "/dev/null")
	assert.Error(t, err)

	err = CopyFile("testfiles", "/tmp/file1.txt")
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
	defer os.RemoveAll("/tmp/copydir.test")
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
