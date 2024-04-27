package utilities

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvOrDefault(t *testing.T) {
	require.Equal(t, EnvOrDefault("INVALID", "default"), "default")

	t.Setenv("ENV", "123")

	require.Equal(t, EnvOrDefault("ENV", "default"), "123")
}

func TestStringInSlice(t *testing.T) {
	require.Equal(t, StringInSlice("test", []string{"test", "test2"}), true)
	require.Equal(t, StringInSlice("test", []string{"test2"}), false)
}

func TestCreateDirIfNotExists(t *testing.T) {
	CreateDirIfNotExists("test")

	require.DirExists(t, "test")

	defer os.Remove("test")
}

func TestParseJSONToMap(t *testing.T) {
	str := `{
  "key": "value",
  "key2": "hallo"
}`
	exp := map[string][]byte{
		"key":  []byte("value"),
		"key2": []byte("hallo"),
	}

	res, err := ParseJSONToMap(str)

	require.NoError(t, err)
	require.Equal(t, exp, res)
}
