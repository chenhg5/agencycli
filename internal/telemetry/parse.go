package telemetry

import (
	"bytes"
	"encoding/json"
)

// StreamUsage holds the last aggregate usage from Claude stream-json "result" lines
// in a run log or combined stdout buffer.
type StreamUsage struct {
	SawResult bool // a line with type "result" was seen (tokens/cost are API-reported)
	InputTokens     int64
	OutputTokens    int64
	CacheReadTokens int64
	TotalCostUSD float64
}

type streamResultUsage struct {
	InputTokens          int64 `json:"input_tokens"`
	OutputTokens         int64 `json:"output_tokens"`
	CacheReadInputTokens int64 `json:"cache_read_input_tokens"`
}

type streamResultLine struct {
	Type         string            `json:"type"`
	TotalCostUSD float64           `json:"total_cost_usd"`
	Usage        streamResultUsage `json:"usage"`
}

// ParseStreamJSONUsage scans newline-delimited JSON (e.g. Claude --output-format stream-json).
// The final line with type "result" wins (same semantics as task tokens parsing).
func ParseStreamJSONUsage(data []byte) StreamUsage {
	var out StreamUsage
	for _, line := range bytes.Split(data, []byte{'\n'}) {
		line = bytes.TrimSpace(line)
		if len(line) < 10 {
			continue
		}
		var rl streamResultLine
		if json.Unmarshal(line, &rl) != nil {
			continue
		}
		if rl.Type == "result" {
			out.SawResult = true
			out.InputTokens = rl.Usage.InputTokens
			out.OutputTokens = rl.Usage.OutputTokens
			out.CacheReadTokens = rl.Usage.CacheReadInputTokens
			out.TotalCostUSD = rl.TotalCostUSD
		}
	}
	return out
}
