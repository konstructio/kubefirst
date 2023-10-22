package k3d

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
)

func TestGetFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("testStringFlag", "default", "test string flag")
	cmd.Flags().Bool("testBoolFlag", false, "test bool flag")

	tests := []struct {
		name        string
		flagConfig  FlagConfig
		expected    interface{}
		expectError bool
		errorText   string
	}{
		{"string flag", FlagConfig{"testStringFlag", "string"}, "default", false, ""},
		{"bool flag", FlagConfig{"testBoolFlag", "bool"}, false, false, ""},
		{"unknown flag type", FlagConfig{"testUnknownFlag", "unknown"}, nil, true, "unknown flag type: unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getFlag(cmd, tt.flagConfig)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if err.Error() != tt.errorText {
					t.Errorf("Expected error message '%v', got '%v'", tt.errorText, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestValidateFlags(t *testing.T) {
	tests := []struct {
		name        string
		flagConfigs []FlagConfig
		expected    map[string]interface{}
		expectError bool
	}{
		{
			"valid string and bool flags",
			[]FlagConfig{
				{"testStringFlag", "string"},
				{"testBoolFlag", "bool"},
			},
			map[string]interface{}{
				"testStringFlag": "default",
				"testBoolFlag":   false,
			},
			false,
		},
		{
			"unknown flag type",
			[]FlagConfig{
				{"testUnknownFlag", "unknown"},
			},
			nil,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.Flags().String("testStringFlag", "default", "test string flag")
			cmd.Flags().Bool("testBoolFlag", false, "test bool flag")

			flagValues := make(map[string]interface{})

			err := validateFlags(cmd, flagValues, tt.flagConfigs)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(flagValues, tt.expected) {
				t.Errorf("Expected flag values to be %v, got %v", tt.expected, flagValues)
			}
		})
	}
}
