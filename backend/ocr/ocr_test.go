package ocr

import (
	"testing"
)

func TestResolveChatEndpoint(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{
			input:    "https://api.openai.com/v1",
			expected: "https://api.openai.com/v1/chat/completions",
		},
		{
			input:    "https://generativelanguage.googleapis.com/v1beta/openai",
			expected: "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			input:    "https://generativelanguage.googleapis.com/v1beta",
			expected: "https://generativelanguage.googleapis.com/v1beta/openai/chat/completions",
		},
		{
			input:    "https://openrouter.ai/api/v1/chat/completions",
			expected: "https://openrouter.ai/api/v1/chat/completions",
		},
	}

	for _, c := range cases {
		actual := ResolveChatEndpoint(c.input)
		if actual != c.expected {
			t.Errorf("for %q, expected %q, got %q", c.input, c.expected, actual)
		}
	}
}
