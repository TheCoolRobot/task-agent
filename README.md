# ⚡ task-agent

**YOLO AI Task Executor for Asana** — Bubble Tea TUI, zero Python, one binary.

> Pick an Asana task → AI executes it autonomously → real files written to disk.

Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) + [Lip Gloss](https://github.com/charmbracelet/lipgloss).

---

## TUI Layout

```
⚡ task-agent  anthropic / claude-sonnet-4-6  ⠋
──────────────────────────────────────────────────────────────────────
┌────────────────────────┐ ┌─────────────────────────────────────────┐
│ Tasks (3)              │ │ Task Details                            │
│                        │ │                                         │
│ ⏳ 🔴 Fix JWT bug      │ │ Fix JWT token expiry bug                │
│ ⏳ 🟡 Write blog post  │ │                                         │
│ ✅    Deploy v2        │ │ ID:       task-001                      │
│                        │ │ Status:   ⏳ Incomplete                 │
│                        │ │ Priority: high                          │
│                        │ │ Due:      2026-03-01                    │
│                        │ │ Assignee: Alice Dev                     │
│                        │ │                                         │
│                        │ │ Description:                            │
│                        │ │   Tokens expiring 30min early due to    │
│                        │ │   timezone mismatch in JWT validation   │
│                        ├─┼─────────────────────────────────────────┤
│                        │ │ Model › Providers  [Tab→models]         │
│                        │ │                                         │
│                        │ │ ▶ Anthropic                             │
│                        │ │   OpenAI                                │
│                        │ │   Groq                                  │
│                        │ │   Moonshot (Kimi)                       │
│                        │ │   Ollama (Local)                        │
└────────────────────────┘ └─────────────────────────────────────────┘
 ↑↓/jk nav · Enter execute · Tab pane · / search · C config · T theme · q quit
✅ Loaded 3 tasks
```

---

## Installation

### Prerequisites

- Go 1.22+
- [`asana-cli`](https://github.com/TheCoolRobot/asana-cli) in your PATH
- `ASANA_TOKEN` environment variable set
- At least one AI provider API key

### Homebrew (recommended)

```bash
brew tap TheCoolRobot/task-agent
brew install task-agent
# asana-cli is installed automatically as a dependency
```

### Build from source

```bash
git clone https://github.com/TheCoolRobot/task-agent.git
cd task-agent

# One-shot: resolves deps, generates go.sum, builds binary
./bootstrap.sh
# Binary: ./bin/task-agent

# Or step by step:
go mod tidy     # resolves deps + writes go.sum
make build      # compiles binary
make install    # copies to /usr/local/bin
```

---

## Quick Start

```bash
# 1. First-time setup (workspace GID, project GID, API keys, theme)
task-agent config

# 2. Launch TUI
task-agent

# 3. Run a task directly without the TUI
task-agent run <task-gid>

# 4. Override provider/model for a single run
task-agent run <task-gid> --provider groq --model llama-3.3-70b-versatile
```

---

## TUI Controls

### Main view

| Key | Action |
|-----|--------|
| `↑` `↓` or `j` `k` | Navigate tasks / provider / model list |
| `Enter` | **Execute task** (tasks pane) · confirm selection (model pane) |
| `Tab` | Cycle panes: Tasks → Providers → Models → Log |
| `/` | Search tasks (Asana API, falls back to local filter) |
| `C` | Open **Config screen** |
| `T` | Cycle through themes live |
| `L` | View execution log |
| `r` | Refresh task list from Asana |
| `Esc` | Return to tasks pane |
| `q` | Quit (config auto-saved) |

### Config screen (`C`)

| Key | Action |
|-----|--------|
| `Tab` / `Shift-Tab` | Move between fields |
| `←` `→` | Cycle through options (provider, theme) |
| Type | Edit text fields (GIDs, API keys, output dir) |
| `Ctrl-S` | **Save and close** |
| `Esc` | Discard changes |

---

## Themes

Switch themes live with `T`, or set permanently in the config screen.

| Theme | Description |
|-------|-------------|
| `dark` | GitHub dark *(default)* |
| `light` | GitHub light |
| `homebrew` | Deep amber on near-black |
| `dracula` | Purple & pink on charcoal |
| `solarized` | Classic Ethan Schoonover dark |
| `nord` | Arctic blue-grey |
| `monokai` | Sublime Text classic |

Persisted to `~/.task-agent/config.json` as `"theme": "nord"`.

---

## AI Providers

| Provider | Models | API Key Env Var |
|----------|--------|-----------------|
| **Anthropic** | claude-opus-4-6, claude-sonnet-4-6, claude-haiku-4-5 | `ANTHROPIC_API_KEY` |
| **OpenAI** | gpt-4o, gpt-4o-mini, gpt-4-turbo, o1, o3-mini | `OPENAI_API_KEY` |
| **Groq** | llama-3.3-70b-versatile, llama-3.1-8b-instant, mixtral-8x7b, gemma2-9b | `GROQ_API_KEY` |
| **Moonshot (Kimi)** | kimi-k2-0711-preview, moonshot-v1-8k/32k/128k | `MOONSHOT_API_KEY` |
| **Ollama** | llama3.3, llama3.1, qwen2.5-coder, mistral, codellama, phi4 | *(none — local)* |

Environment variables always take precedence over keys stored in config.

```bash
export ANTHROPIC_API_KEY=sk-ant-...
export OPENAI_API_KEY=sk-...
export GROQ_API_KEY=gsk_...
export MOONSHOT_API_KEY=sk-...
```

Switch providers and models interactively with `Tab` in the TUI, or pass flags at the CLI:

```bash
task-agent run <gid> -p moonshot -m kimi-k2-0711-preview
```

---

## YOLO Mode

When you press `Enter` on a task, the agent:

1. Formats the full task as markdown (name, description, priority, due date, tags, assignee)
2. Sends it to the selected AI with the system prompt: *"execute completely, produce real files, make decisions, don't ask for permission"*
3. Streams live progress into the log panel as the AI responds
4. Parses the structured JSON response (output type, files, summary, notes)
5. Writes all files to `./task-outputs/<timestamp>_<task-name>/`

### Output structure

```
task-outputs/
└── 20260301_143022_Fix_JWT_token_expiry_bug/
    ├── AGENT_MANIFEST.md       ← Summary, file index, agent notes
    ├── auth/
    │   ├── jwt.go              ← Generated fix
    │   └── jwt_test.go         ← Tests included automatically
    └── README.md
```

The AI picks the output type (`markdown`, `code_folder`, or `mixed`) based on the task.

---

## CLI Reference

```bash
task-agent                              # Launch TUI (default)
task-agent tui                          # Launch TUI explicitly
task-agent run <gid>                    # Execute task by GID
task-agent run <gid> -p openai -m gpt-4o
task-agent list                         # List tasks (table)
task-agent list --json                  # List tasks (JSON)
task-agent search "auth bug"            # Search tasks
task-agent search "fix" -w <ws-gid>    # Search in specific workspace
task-agent config                       # Interactive setup wizard
task-agent providers                    # Show all providers + API key status
```

---

## Configuration

Stored at `~/.task-agent/config.json` (permissions `0600`):

```json
{
  "workspace_gid": "12345678",
  "project_gid":   "87654321",
  "provider":      "anthropic",
  "model":         "claude-sonnet-4-6",
  "output_dir":    "./task-outputs",
  "theme":         "dark",
  "api_keys": {
    "anthropic": "sk-ant-...",
    "openai":    "sk-...",
    "groq":      "gsk_...",
    "moonshot":  "sk-..."
  }
}
```

---

## Project Structure

```
task-agent/
├── cmd/task-agent/main.go        ← Cobra CLI entry point
├── internal/
│   ├── ai/providers.go           ← Multi-provider HTTP clients (Anthropic + OpenAI-compat)
│   ├── asana/client.go           ← asana-cli subprocess wrapper
│   ├── config/config.go          ← ~/.task-agent/config.json
│   ├── output/writer.go          ← Writes AI result files + manifest to disk
│   └── tui/
│       ├── model.go              ← Bubble Tea Model, Update, key handlers
│       ├── view.go               ← Bubble Tea View rendering
│       └── styles.go             ← Lip Gloss theme system (7 themes)
├── .github/
│   ├── workflows/
│   │   ├── ci.yml                ← Build matrix + GoReleaser dry run
│   │   └── release.yml           ← Tag-triggered release + Homebrew tap update
│   └── HOMEBREW_SETUP.md         ← Tap wiring guide
├── .goreleaser.yaml              ← Cross-platform build + brew formula
├── bootstrap.sh                  ← First-time dependency + build script
├── go.mod
└── Makefile
```

---

## Homebrew Tap Setup

After a tagged release, the formula in `TheCoolRobot/homebrew-task-agent` is updated automatically by GoReleaser. See [`.github/HOMEBREW_SETUP.md`](.github/HOMEBREW_SETUP.md) for the one-time wiring instructions (tap repo creation + `TAP_GITHUB_TOKEN` secret).

---

## License

MIT