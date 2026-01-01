package domain

import (
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
			name:        "有効な値: draft",
			input:       "draft",
			expected:    PostStatusDraft,
			expectError: false,
		},
		{
			name:        "有効な値: published",
			input:       "published",
			expected:    PostStatusPublished,
			expectError: false,
		},
		{
			name:        "無効な値: invalid",
			input:       "invalid",
			expected:    "",
			expectError: true,
		},
		{
			name:        "無効な値: 空文字列",
			input:       "",
			expected:    "",
			expectError: true,
		},
		{
			name:        "無効な値: Draft（大文字）",
			input:       "Draft",
			expected:    "",
			expectError: true,
		},
		{
			name:        "無効な値: PUBLISHED（大文字）",
			input:       "PUBLISHED",
			expected:    "",
			expectError: true,
		},
		{
			name:        "無効な値: publised（タイポ）",
			input:       "publised",
			expected:    "",
			expectError: true,
		},
		{
			name:        "無効な値: ランダムな文字列",
			input:       "random-status",
			expected:    "",
			expectError: true,
		},
		{
			name:        "無効な値: スペース付き",
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
				// エラーメッセージに入力値が含まれていることを確認
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
	t.Run("エラーメッセージに有効な値が含まれる", func(t *testing.T) {
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
			if !contains(errMsg, substr) {
				t.Errorf("expected error message to contain %q, got: %s", substr, errMsg)
			}
		}
	})
}

// ヘルパー関数
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) && containsAt(s, substr))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
