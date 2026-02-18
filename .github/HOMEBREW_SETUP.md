# Homebrew Tap Setup Guide

This document explains how to wire up the Homebrew tap so `brew install task-agent` works
after every tagged release.

---

## Overview

You need **two GitHub repos**:

```
github.com/<you>/task-agent            ← main app (this repo)
github.com/<you>/homebrew-task-agent   ← tap repo (formula lives here)
```

GoReleaser handles everything: it builds cross-platform binaries, creates the GitHub
Release, computes SHA256 checksums, and commits the updated formula to the tap repo.

---

## Step 1 — Create the tap repo

```bash
# On GitHub, create a new public repo named exactly:
homebrew-task-agent

# Clone it and add the Formula directory
git clone https://github.com/<you>/homebrew-task-agent
cd homebrew-task-agent
mkdir Formula
cp /path/to/task-agent/.github/homebrew/task-agent.rb.template Formula/task-agent.rb
# Edit the <you> placeholders in the template
git add . && git commit -m "chore: initial formula scaffold"
git push
```

---

## Step 2 — Create the TAP_GITHUB_TOKEN secret

GoReleaser needs write access to the tap repo to commit the updated formula.

1. Go to **GitHub → Settings → Developer settings → Personal access tokens → Fine-grained tokens**
2. Create a new token with:
   - **Resource owner**: your account
   - **Repository access**: Only `homebrew-task-agent`
   - **Permissions**: Contents → Read and Write
3. Copy the token

4. In the **main `task-agent` repo** → Settings → Secrets and variables → Actions:
   - Add secret: `TAP_GITHUB_TOKEN` = (the token you just copied)

> `GITHUB_TOKEN` is automatically available in Actions — you don't need to add it.

---

## Step 3 — Tag a release

```bash
git tag v1.0.0
git push origin v1.0.0
```

This triggers `.github/workflows/release.yml`, which:
1. Builds binaries for macOS (arm64, amd64), Linux (arm64, amd64), Windows (amd64)
2. Creates a GitHub Release with all archives + `checksums.txt`
3. Commits the updated `Formula/task-agent.rb` to `homebrew-task-agent`

---

## Step 4 — Verify the tap

```bash
brew tap <you>/task-agent
brew install task-agent
task-agent --version
```

---

## Workflow summary

```
git push origin v1.2.3
        │
        ▼
release.yml (GitHub Actions)
        │
        ├─► GoReleaser builds:
        │     task-agent_Darwin_arm64.tar.gz
        │     task-agent_Darwin_x86_64.tar.gz
        │     task-agent_Linux_arm64.tar.gz
        │     task-agent_Linux_x86_64.tar.gz
        │     task-agent_Windows_x86_64.zip
        │     checksums.txt
        │
        ├─► Creates GitHub Release (with changelog)
        │
        └─► Commits to homebrew-task-agent/Formula/task-agent.rb
              with correct URLs + SHA256 hashes
```

---

## Local release dry run

```bash
# Install GoReleaser
brew install goreleaser

# Snapshot build (no tag needed, no publish)
goreleaser build --snapshot --clean

# Full dry run (uses current tag, skips publish)
goreleaser release --skip=publish --clean
```

---

## Secrets reference

| Secret | Where to add | Purpose |
|--------|-------------|---------|
| `GITHUB_TOKEN` | Auto-provided | Create releases, upload assets |
| `TAP_GITHUB_TOKEN` | Repo → Settings → Secrets | Write formula to tap repo |