package telemetry

import (
	"testing"
)

func TestParseStreamJSONUsage(t *testing.T) {
	data := []byte(`{"type":"assistant","message":{"role":"assistant"}}
{"type":"result","usage":{"input_tokens":100,"output_tokens":50,"cache_read_input_tokens":20},"total_cost_usd":0.0123}
`)
	u := ParseStreamJSONUsage(data)
	if !u.SawResult {
		t.Fatal("expected SawResult")
	}
	if u.InputTokens != 100 || u.OutputTokens != 50 || u.CacheReadTokens != 20 {
		t.Fatalf("tokens: %+v", u)
	}
	if u.TotalCostUSD != 0.0123 {
		t.Fatalf("cost: %v", u.TotalCostUSD)
	}
}
