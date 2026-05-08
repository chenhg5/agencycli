package runner

import (
	"testing"

	"github.com/chenhg5/agencycli/internal/entity"
)

func TestCodexInvokerParseSessionID(t *testing.T) {
	invoker := &codexInvoker{}
	got := invoker.ParseSessionID("OpenAI Codex\nSession ID: sess-123\n")
	if got != "sess-123" {
		t.Fatalf("ParseSessionID() = %q, want %q", got, "sess-123")
	}
}

func TestCodexInvokerParseLowercaseSessionID(t *testing.T) {
	invoker := &codexInvoker{}
	got := invoker.ParseSessionID("OpenAI Codex\nsession id: 019e0262-618f-7d80-9a6d-fb5ed664ccaa\n")
	if got != "019e0262-618f-7d80-9a6d-fb5ed664ccaa" {
		t.Fatalf("ParseSessionID() = %q", got)
	}
}

func TestCodexResumeMissingRolloutError(t *testing.T) {
	output := "Error: thread/resume: thread/resume failed: no rollout found for thread id 019e0262-618f-7d80-9a6d-fb5ed664ccaa"
	if !isCodexResumeMissingRolloutError(output) {
		t.Fatal("expected missing rollout error to be detected")
	}
}

func TestDiscardSessionIDOnFailure(t *testing.T) {
	if !discardSessionIDOnFailure(entity.ModelCodex) {
		t.Fatal("expected codex failed sessions to be discarded")
	}
	if discardSessionIDOnFailure(entity.ModelClaudeCode) {
		t.Fatal("did not expect claude failed sessions to be discarded")
	}
}
