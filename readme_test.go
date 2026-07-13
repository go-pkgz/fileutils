package fileutils

import (
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadmeExample(t *testing.T) {
	readme, err := os.ReadFile("README.md") //nolint:gosec // repository documentation is the test input
	require.NoError(t, err)

	const startMarker = "<!-- fileutils-example-start -->\n```go\n"
	const endMarker = "\n```\n<!-- fileutils-example-end -->"
	start := strings.Index(string(readme), startMarker)
	require.NotEqual(t, -1, start, "README example start marker is missing")
	start += len(startMarker)
	end := strings.Index(string(readme)[start:], endMarker)
	require.NotEqual(t, -1, end, "README example end marker is missing")

	example, err := os.ReadFile("examples/basic/main.go") //nolint:gosec // repository example is the test input
	require.NoError(t, err)
	require.Equal(t, strings.TrimSpace(string(example)), strings.TrimSpace(string(readme)[start:start+end]))

	cmd := exec.Command("go", "run", "./examples/basic") //nolint:gosec // fixed repository example path
	output, err := cmd.CombinedOutput()
	require.NoErrorf(t, err, "documented example failed: %s", output)
	require.Contains(t, string(output), "SHA-256:")
}
