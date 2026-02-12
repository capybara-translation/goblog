package domain

import (
	"strings"
	"testing"
)

func TestParsePostStatus(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    PostStatus
		expectError bool
	}{
		{
			name:        "Valid value: draft",
			input:       "draft",
			expected:    PostStatusDraft,
			expectError: false,
		},
		{
			name:        "Valid value: published",
			input:       "published",
			expected:    PostStatusPublished,
			expectError: false,
		},
		{
			name:        "Invalid value: invalid",
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid value: empty string",
			input:       "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid value: Draft (uppercase)",
			input:       "Draft",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid value: PUBLISHED (uppercase)",
			input:       "PUBLISHED",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid value: publised (typo)",
			input:       "publised",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid value: random string",
			input:       "random-status",
			expected:    "",
			expectError: true,
		},
		{
			name:        "Invalid value: with spaces",
			input:       " draft ",
			expected:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePostStatus(tt.input)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error for input %q, but got nil", tt.input)
				}
				// Verify that the error message contains the input value
				if err != nil && tt.input != "" {
					expectedMsg := "invalid status: '" + tt.input + "'"
					if err.Error()[:len(expectedMsg)] != expectedMsg {
						t.Errorf("expected error message to start with %q, got %q", expectedMsg, err.Error())
					}
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for input %q: %v", tt.input, err)
				}
				if result != tt.expected {
					t.Errorf("expected %q for input %q, got %q", tt.expected, tt.input, result)
				}
			}
		})
	}
}

func TestParsePostStatus_ErrorMessage(t *testing.T) {
	t.Run("Error message contains valid values", func(t *testing.T) {
		_, err := ParsePostStatus("invalid")
		if err == nil {
			t.Fatal("expected error, got nil")
		}

		errMsg := err.Error()
		expectedSubstrings := []string{
			"invalid status",
			"'invalid'",
			"Must be",
			"'draft'",
			"'published'",
		}

		for _, substr := range expectedSubstrings {
			if !strings.Contains(errMsg, substr) {
				t.Errorf("expected error message to contain %q, got: %s", substr, errMsg)
			}
		}
	})
}
