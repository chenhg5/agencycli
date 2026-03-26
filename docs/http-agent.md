# HTTP agent (`http-agent`)

The **HTTP agent** model lets agencycli run tasks by calling a **custom HTTP API** instead of a local CLI (Claude Code, Codex, etc.). It is aimed at **OpenAI-compatible chat completion** endpoints: Ollama, LM Studio, LocalAI, vLLM, OpenAI, Azure OpenAI-style proxies, or your own gateway.

Execution is **HTTP-only**: there is no subprocess, no Docker-wrapped CLI, and no automatic session file on disk. The scheduler, task queue, inbox, and heartbeats behave like any other agent; only the **inference step** is a `POST` to your URL.

---

## What gets sent

On each `agencycli run` / `agencycli exec`:

1. **System message** — contents of the agent’s merged context file. After hire, this is normally `context.md` in the agent directory (same layout as `generic-cli`: agency → team → role → project layers plus skills). If that file is missing, the runner falls back to `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `OPENCODE.md`, or `IFLOW.md` so older workspaces still work.
2. **User message** — the task prompt (for queued tasks) or your raw prompt (`exec`), plus agencycli’s **system metadata footer** (task id, `agencycli task done`, `confirm-request`, etc.).

The request body matches the common **chat completions** shape:

```json
{
  "model": "<from config>",
  "messages": [
    { "role": "system", "content": "..." },
    { "role": "user", "content": "..." }
  ],
  "stream": true
}
```

Headers:

- `Content-Type: application/json`
- `Accept: text/event-stream` when streaming, otherwise `application/json`
- `Authorization: Bearer <token>` when an API key is configured (see below)
- Any **extra headers** from config (repeatable `--http-header` at hire time)

---

## Response format

### Non-streaming (`stream: false`)

agencycli expects a JSON body compatible with:

```json
{
  "choices": [
    { "message": { "role": "assistant", "content": "full reply text" } }
  ],
  "error": { "type": "...", "message": "..." }
}
```

The first choice’s `message.content` is treated as the full model output. Errors in `error` are surfaced to the user.

### Streaming (`stream: true`, default)

The server should return **Server-Sent Events**: lines `data: <json>` where each JSON object follows the usual delta shape, e.g.:

```json
{
  "choices": [
    { "delta": { "content": "token or fragment" } }
  ],
  "error": { "type": "...", "message": "..." }
}
```

Lines `data: [DONE]` end the stream. Malformed `data:` lines are skipped; `error` on a chunk fails the run.

---

## Hiring an HTTP agent

```bash
agencycli hire \
  --project my-api \
  --team engineering \
  --role developer \
  --model http-agent \
  --name local-llm \
  --http-url "http://localhost:11434/v1/chat/completions" \
  --http-model "llama3.2"
```

Common flags:

| Flag | Purpose |
|------|--------|
| `--http-url` | **Required.** Full URL to the chat completions endpoint. |
| `--http-model` | Model id passed in the JSON `model` field (server-specific). |
| `--http-api-key` | `Authorization: Bearer …` value. Omit for local servers with no auth. |
| `--http-timeout` | Per-request timeout (Go duration), default `10m`. |
| `--http-stream` | `true` / `false`; default `true` (SSE). |
| `--http-header` | Repeatable `Key: Value` extra headers. |

If you omit `--http-api-key`, you can set **`AGENCYCLI_HTTP_API_KEY`** in the environment when agencycli runs; the YAML value takes precedence if both are set.

After hire, metadata is stored in:

`projects/<project>/agents/<name>/.agencycli/agent.yaml`

Relevant excerpt:

```yaml
model: http-agent
http_agent:
  url: http://localhost:11434/v1/chat/completions
  model: llama3.2
  api_key: ""           # optional; prefer env for secrets
  timeout: 10m
  stream: true
  extra_headers: {}
```

You may edit `agent.yaml` by hand (e.g. add `extra_headers`) and keep context in sync with `agencycli sync --project <project> --name <name>`.

---

## Running tasks

```bash
agencycli run  --project my-api --agent local-llm
agencycli exec --project my-api --agent local-llm --prompt "Summarise the README."
```

- **Host vs Docker:** For `http-agent`, agencycli always performs the HTTP call from the **agencycli process** on the host. A `sandbox: docker` entry in `agent.yaml` does **not** wrap this model in a container for task execution.
- **Session IDs:** There is no CLI session file. If the model’s **text** output contains a line starting with `AGENCYCLI_SESSION_ID:`, agencycli records it like other models (optional, for custom tooling).

---

## Optional: human confirmation from the model

If the HTTP model’s reply includes a line starting with:

`AGENCYCLI_AWAIT_CONFIRM:`

the remainder of the line is treated as the confirmation summary and the task moves to **awaiting confirmation**, same as CLI agents. This only works if the **backend** (or a wrapper) emits that exact prefix in the assistant content.

---

## Building a minimal compatible server

Your server must:

1. Accept `POST` with JSON body as above.
2. Return `200` with either a full JSON completion or an SSE stream as described.
3. Put the assistant text in `choices[0].message.content` (non-stream) or stream deltas in `choices[0].delta.content`.

Pseudo-checklist for local testing:

- Point `--http-url` at `http://127.0.0.1:<port>/v1/chat/completions` (or your provider’s real path).
- Match `--http-model` to what the server expects (`llama3.2`, `gpt-4o`, etc.).
- Disable streaming (`--http-stream=false`) if your server only returns JSON.

---

## Troubleshooting

| Symptom | Things to check |
|--------|------------------|
| `http-agent: no http_agent config` | Re-hire with `--http-url` or fix `http_agent` in `agent.yaml`. |
| HTTP 4xx/5xx | URL path, model name, API key, and extra headers. |
| Empty reply | Server must return `choices[0].message.content` or stream non-empty deltas. |
| Stream hangs | Ensure the server sends `data: [DONE]` or closes the body; timeout is controlled by `timeout` in config. |
| SSL / corporate proxy | Use `HTTPS_PROXY` / `NO_PROXY` for the Go HTTP client (standard env vars). |

---

## Related

- [Command reference](./commands.md) — `hire`, `run`, `exec`, `task`
- [Workspace layout](./workspace-layout.md) — where agent files live
