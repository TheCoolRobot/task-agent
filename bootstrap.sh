#!/usr/bin/env bash
# bootstrap.sh — Run once to resolve deps and build task-agent
set -e

echo "🔧 Bootstrapping task-agent..."
echo ""

# Check Go is installed
if ! command -v go &>/dev/null; then
  echo "❌  Go is not installed."
  echo "    Install from: https://go.dev/dl/"
  echo "    Or via Homebrew: brew install go"
  exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED="1.22"
echo "✅  Go $GO_VERSION found"

# Check asana-cli
if command -v asana-cli &>/dev/null; then
  echo "✅  asana-cli found at $(which asana-cli)"
else
  echo "⚠️   asana-cli not found in PATH"
  echo "    Install: https://github.com/TheCoolRobot/asana-cli"
  echo "    Or brew install thecoolrobot/asana-cli/asana-cli"
  echo "    (You can still build task-agent — just won't be able to fetch real tasks)"
fi

echo ""
echo "📦  Resolving dependencies..."

# Remove stale go.sum if present
rm -f go.sum

# Let go mod tidy figure out the correct indirect deps + generate go.sum
go mod tidy

echo "✅  Dependencies resolved"
echo ""
echo "🔨  Building..."

mkdir -p bin
go build -ldflags="-s -w" -o bin/task-agent ./cmd/task-agent

echo "✅  Built: ./bin/task-agent"
echo ""
echo "🚀  Next steps:"
echo "    ./bin/task-agent config   # First-time setup (workspace, API keys)"
echo "    ./bin/task-agent          # Launch TUI"
echo ""
echo "    # Optional: install globally"
echo "    sudo cp bin/task-agent /usr/local/bin/task-agent"
echo "    # or: make install"