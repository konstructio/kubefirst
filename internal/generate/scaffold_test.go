package generate

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_generateAppScaffoldEnvironmentFiles(t *testing.T) {
	tests := []struct {
		Name        string
		Environment string
		Error       error
	}{
		{
			Name:        "app",
			Environment: "development",
		},
		{
			Name:        "metaphor",
			Environment: "production",
		},
		{
			Name:        "some-app",
			Environment: "some-environment",
		},
	}

	for _, test := range tests {
		goldenDir := filepath.Join(".", "testdata", "scaffold", test.Environment)

		fileData, err := generateAppScaffoldEnvironmentFiles(test.Name, test.Environment)
		require.Equal(t, test.Error, err)

		expectedFiles := []string{}
		for k := range fileData.data {
			expectedFiles = append(expectedFiles, k)
		}

		actualFiles := make([]string, 0)
		err = filepath.WalkDir(goldenDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			f, _ := strings.CutPrefix(path, goldenDir+string(filepath.Separator))
			actualFiles = append(actualFiles, f)

			actualFileContent, err := os.ReadFile(path)
			require.Nil(t, err)

			expectedFileContent, ok := fileData.data[f]
			require.True(t, ok)

			require.Equal(t, expectedFileContent.String(), string(actualFileContent))

			return nil
		})
		require.Nil(t, err)

		slices.Sort(expectedFiles)
		slices.Sort(actualFiles)
		require.Equal(t, expectedFiles, actualFiles)
	}
}
