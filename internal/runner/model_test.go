package runner

import "testing"

func TestCodexInvokerParseSessionID(t *testing.T) {
	invoker := &codexInvoker{}
	got := invoker.ParseSessionID("OpenAI Codex\nSession ID: sess-123\n")
	if got != "sess-123" {
		t.Fatalf("ParseSessionID() = %q, want %q", got, "sess-123")
	}
}
