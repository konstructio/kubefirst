package reports

import (
	"strings"
	"testing"
)

// todo: test all conditions using table test table
// TestBuildCreateHandOffReport test if the built string is consuming the required data
func TestBuildCreateHandOffReport(t *testing.T) {
	mockURL := "mock-url"
	mockUsername := "mock-username"
	mockPassword := "mock-password"

	handOffData := CreateHandOff{
		ArgoCDUrl:      mockURL,
		ArgoCDUsername: mockUsername,
		ArgoCDPassword: mockPassword,
	}
	got := BuildCreateHandOffReport(handOffData)

	if !strings.Contains(got.String(), mockUsername) {
		t.Errorf("built buffer doesn't contain %q", mockUsername)
	}

	if !strings.Contains(got.String(), mockPassword) {
		t.Errorf("built buffer doesn't contain %q", mockPassword)
	}
}
