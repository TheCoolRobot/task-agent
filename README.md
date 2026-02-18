# ⚡ task-agent

**YOLO AI Task Executor for Asana** — Bubble Tea TUI, zero Python, one binary.

> Pick an Asana task → AI executes it autonomously → real files written to disk.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## TUI Layout

```
⚡ task-agent  anthropic / claude-sonnet-4-6  ⠋
──────────────────────────────────────────────────────────────
┌──────────────────────┐ ┌───────────────────────────────────┐
│ Tasks (3)            │ │ Task Details                      │
│                      │ │                                   │
│ ⏳ 🔴 Fix JWT bug    │ │ Fix JWT token expiry bug          │
│ ⏳ 🟡 Write blog     │ │                                   │
│ ✅    Deploy v2      │ │ ID: task-001                      │
│                      │ │ Status: ⏳ Incomplete             │
│                      │ │ Priority: high                    │
│                      │ │ Due: 2026-03-01                   │
│                      │ │ Assignee: Alice Dev               │
│                      │ │                                   │
│                      │ │ Description:                      │
│                      │ │   Tokens expiring 30min early     │
│                      │ │   due to timezone mismatch...     │
│                      ├─┼───────────────────────────────────┤
│                      │ │ Model › Providers [Tab=models]    │
│                      │ │                                   │
│                      │ │ ▶ Anthropic                       │
│                      │ │   OpenAI                          │
│                      │ │   Groq                            │
│                      │ │   Ollama (Local)                  │
└──────────────────────┘ └───────────────────────────────────┘
 ↑↓ nav · Enter execute · Tab switch pane · / search · q quit
✅ Loaded 3 tasks
```

---

## Installation

### Prerequisites

- Go 1.22+
- [`asana-cli`](https://github.com/TheCoolRobot/asana-cli) in your PATH
- `ASANA_TOKEN` environment variable
- At least one AI provider API key

### Build

```bash
git clone https://github.com/you/task-agent.git
cd task-agent

# Download dependencies
go mod download

# Build
make build
# Binary: ./bin/task-agent

# Or install globally
make install
```

### Quick install with Go

```bash
go install github.com/you/task-agent/cmd/task-agent@latest
```

---

## Quick Start

```bash
# 1. First-time setup (workspace, project, API keys)
task-agent config

# 2. Launch TUI (default command)
task-agent
# or explicitly:
task-agent tui

# 3. Run a task directly (no TUI)
task-agent run <task-gid>

# 4. With a specific provider/model
task-agent run <task-gid> --provider groq --model llama-3.3-70b-versatile
```

---

## TUI Controls

### Main view

| Key | Action |
|-----|--------|
| `↑` / `↓` or `j` / `k` | Navigate tasks or model list |
| `Enter` | **Execute task** (tasks pane) or **confirm selection** (model pane) |
| `Tab` | Cycle panes: Tasks → Model Providers → Model Names → Log |
| `/` | Search tasks (Asana API or local filter) |
| `C` | Open full **Config screen** |
| `L` | Switch to execution log |
| `r` | Refresh task list from Asana |
| `q` | Quit (config auto-saved) |

### Config screen (`C`)

| Key | Action |
|-----|--------|
| `Tab` / `Shift-Tab` | Move between fields |
| `←` / `→` or `Enter` | Cycle option fields (provider) |
| Type | Edit text fields (GIDs, API keys, output dir) |
| `Ctrl-S` | **Save config** and return |
| `Esc` | Cancel without saving |

---

## AI Providers

| Provider | Models | API Key |
|----------|--------|---------|
| **Anthropic** | claude-opus-4-6, claude-sonnet-4-6, claude-haiku-4-5 | `ANTHROPIC_API_KEY` |
| **OpenAI** | gpt-4o, gpt-4o-mini, o1, o3-mini | `OPENAI_API_KEY` |
| **Groq** | llama-3.3-70b-versatile, mixtral-8x7b, gemma2-9b | `GROQ_API_KEY` |
| **Ollama** | llama3.3, qwen2.5-coder, mistral, codellama | *(none, runs locally)* |

```bash
# Set via environment
export ANTHROPIC_API_KEY=sk-ant-...
export OPENAI_API_KEY=sk-...
export GROQ_API_KEY=gsk_...

# Or store in config (encrypted at rest is your responsibility)
task-agent config
```

---

## YOLO Mode

When you hit `Enter` on a task, the agent:

1. Formats the full task as markdown (name, description, priority, due date, tags, assignee)
2. Sends it to your selected AI with a system prompt: *"execute completely, produce real files, make decisions, don't ask for permission"*
3. AI responds with structured JSON containing file paths + content
4. Files are written to `./task-outputs/<timestamp>_<task-name>/`

### Output structure

```
task-outputs/
└── 20260217_143022_Fix_JWT_token_expiry_bug/
    ├── AGENT_MANIFEST.md       ← Summary, file index, notes
    ├── auth/
    │   ├── jwt.go              ← Generated fix
    │   └── jwt_test.go         ← Tests
    └── README.md               ← Task-specific docs
```

The AI decides the output type (`markdown`, `code_folder`, or `mixed`) based on the task content.

---

## CLI Reference

```bash
task-agent                          # Launch TUI (default)
task-agent tui                      # Launch TUI explicitly
task-agent run <gid>                # Execute task by GID
task-agent run <gid> -p openai -m gpt-4o  # With specific model
task-agent list                     # List tasks (table)
task-agent list --json              # List tasks (JSON)
task-agent search "auth bug"        # Search tasks
task-agent search "fix" -w <ws>    # Search in workspace
task-agent config                   # Interactive setup
task-agent providers                # Show providers + key status
```

---

## Configuration

Stored at `~/.task-agent/config.json` (permissions: `0600`):

```json
{
  "workspace_gid": "12345678",
  "project_gid":   "87654321",
  "provider":      "anthropic",
  "model":         "claude-sonnet-4-6",
  "output_dir":    "./task-outputs",
  "api_keys": {
    "anthropic": "sk-ant-...",
    "openai":    "sk-..."
  }
}
```

Environment variables always take precedence over stored keys.

---

## Project Structure

```
task-agent/
├── cmd/task-agent/main.go    ← Cobra CLI entry point
├── internal/
│   ├── ai/providers.go       ← Anthropic + OpenAI-compat HTTP clients
│   ├── asana/client.go       ← asana-cli subprocess wrapper
│   ├── config/config.go      ← ~/.task-agent/config.json
│   ├── output/writer.go      ← Writes AI result files to disk
│   └── tui/
│       ├── model.go          ← Bubble Tea Model + Update
│       ├── view.go           ← Bubble Tea View rendering
│       └── styles.go         ← Lip Gloss styles
├── go.mod
└── Makefile
```

---

## License

MIT
