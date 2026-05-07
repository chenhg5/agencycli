package api

import (
	"testing"

	"github.com/chenhg5/agencycli/internal/entity"
)

func TestChatSSELineKeepsCodexTranscriptRaw(t *testing.T) {
	line := "codex"
	got := chatSSELine(line, entity.ModelCodex)
	if got != line {
		t.Fatalf("chatSSELine() = %q, want %q", got, line)
	}
}

func TestChatSSELineWrapsPlainGenericStatus(t *testing.T) {
	got := chatSSELine("plain status", entity.ModelClaudeCode)
	want := "=== plain status ==="
	if got != want {
		t.Fatalf("chatSSELine() = %q, want %q", got, want)
	}
}

func TestExtractAgentChatSessionID(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "json",
			line: `{"type":"system","session_id":"abc123"}`,
			want: "abc123",
		},
		{
			name: "codex label",
			line: "Session ID: sess-456",
			want: "sess-456",
		},
		{
			name: "multiline log",
			line: "header\nSession: sess-789\nfooter",
			want: "sess-789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractAgentChatSessionID(tt.line); got != tt.want {
				t.Fatalf("extractAgentChatSessionID() = %q, want %q", got, tt.want)
			}
		})
	}
}
