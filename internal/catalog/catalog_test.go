package catalog

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCatalogApps(t *testing.T) {
	testCases := []struct {
		name string
		app  string
		err  bool
	}{
		{
			name: "valid app: tracetest",
			app:  "tracetest",
		},
		{
			name: "invalid app: doesnotexist",
			err:  true,
			app:  "doesnotexist",
		},
		{
			name: "valid multiple app: tracetest,yourls",
			app:  "tracetest,yourls",
		},
	}

	for _, tc := range testCases {

		ok, items, err := ValidateCatalogApps(tc.app)

		// should not exist
		if tc.err {
			// should error
			require.Error(t, err, tc.name)

			// app has not been found
			require.False(t, ok, tc.name)
			// exists
		} else {
			// should not error
			require.NoError(t, err, tc.name)

			// name equals app name
			for i, a := range strings.Split(tc.app, ",") {
				require.Equal(t, a, items[i].Name, tc.name)
			}

			// app has been found
			require.True(t, ok, tc.name)
		}
	}
}
