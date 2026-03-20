package runner

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/chenhg5/agencycli/internal/entity"
)

// ── OpenAI Chat Completions wire types ────────────────────────────────────────

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model,omitempty"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// chatResponse is the non-streaming response body.
type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *chatError `json:"error,omitempty"`
}

// streamChunk is one delta from a server-sent events streaming response.
type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
	Error *chatError `json:"error,omitempty"`
}

type chatError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// ── Context file reader ────────────────────────────────────────────────────────

// readAgentContextFile reads the merged context file from agentDir.
// It tries context.md first (http-agent / generic-cli native), then falls back
// to the other model-specific names so that the http runner works even when the
// agent dir was originally provisioned for a different model.
func readAgentContextFile(agentDir string) string {
	for _, name := range []string{
		"context.md", "AGENTS.md", "CLAUDE.md", "GEMINI.md", "OPENCODE.md", "IFLOW.md",
	} {
		data, err := os.ReadFile(filepath.Join(agentDir, name))
		if err == nil {
			return string(data)
		}
	}
	return ""
}

// ── Core HTTP execution ────────────────────────────────────────────────────────

// httpExec sends a chat completions request to the configured endpoint and
// returns the model's full response as a string.
//
//   - systemPrompt: the agent's merged context file content; sent as the
//     "system" role message (may be empty for models that don't support it).
//   - userPrompt:   the task prompt + system meta footer; sent as "user".
//   - logWriter:    every token is written here (the run log file).
//   - toStdout:     when true, tokens are also streamed to os.Stdout in real time.
func httpExec(
	cfg *entity.HTTPAgentConfig,
	systemPrompt, userPrompt string,
	logWriter io.Writer,
	toStdout bool,
) (string, error) {
	// Resolve API key: flag value → environment variable fallback.
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("AGENCYCLI_HTTP_API_KEY")
	}

	// Build message list.
	messages := make([]chatMessage, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemPrompt})
	}
	messages = append(messages, chatMessage{Role: "user", Content: userPrompt})

	reqBody := chatRequest{
		Model:    cfg.Model,
		Messages: messages,
		Stream:   cfg.Stream,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("http-agent: marshal request: %w", err)
	}

	// Parse timeout (default 10 min).
	timeout := 10 * time.Minute
	if cfg.Timeout != "" {
		if d, err2 := time.ParseDuration(cfg.Timeout); err2 == nil && d > 0 {
			timeout = d
		}
	}

	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequest(http.MethodPost, cfg.URL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("http-agent: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if cfg.Stream {
		req.Header.Set("Accept", "text/event-stream")
	} else {
		req.Header.Set("Accept", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	for k, v := range cfg.ExtraHeaders {
		req.Header.Set(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http-agent: POST %s: %w", cfg.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		raw, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("http-agent: server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}

	var full strings.Builder

	if cfg.Stream {
		full, err = readSSEStream(resp.Body, logWriter, toStdout)
		if err != nil {
			return full.String(), err
		}
		if toStdout {
			fmt.Fprintln(os.Stdout) // trailing newline after streamed output
		}
	} else {
		var chatResp chatResponse
		if err2 := json.NewDecoder(resp.Body).Decode(&chatResp); err2 != nil {
			return "", fmt.Errorf("http-agent: decode response: %w", err2)
		}
		if chatResp.Error != nil {
			return "", fmt.Errorf("http-agent: API error (%s): %s", chatResp.Error.Type, chatResp.Error.Message)
		}
		if len(chatResp.Choices) > 0 {
			text := chatResp.Choices[0].Message.Content
			full.WriteString(text)
			if logWriter != nil {
				fmt.Fprint(logWriter, text)
			}
			if toStdout {
				fmt.Fprintln(os.Stdout, text)
			}
		}
	}

	return full.String(), nil
}

// readSSEStream reads a server-sent events body and accumulates the text delta.
// Each "data: <json>" line is decoded as a streamChunk.
func readSSEStream(body io.Reader, logWriter io.Writer, toStdout bool) (strings.Builder, error) {
	var full strings.Builder
	scanner := bufio.NewScanner(body)
	// Increase buffer for very long lines (some proxies send large chunks).
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
			// Skip malformed lines silently (some servers send comment lines).
			continue
		}
		if chunk.Error != nil {
			return full, fmt.Errorf("http-agent: stream error (%s): %s", chunk.Error.Type, chunk.Error.Message)
		}
		for _, choice := range chunk.Choices {
			text := choice.Delta.Content
			if text == "" {
				continue
			}
			full.WriteString(text)
			if logWriter != nil {
				fmt.Fprint(logWriter, text)
			}
			if toStdout {
				fmt.Print(text)
			}
		}
	}
	return full, scanner.Err()
}
