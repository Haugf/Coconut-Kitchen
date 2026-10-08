#!/usr/bin/env bash
# Run by the mirror-update timer. Installs whatever is on GitHub's main
# branch if it isn't what's installed yet. Compares against what was
# actually installed (not just pulled), so a failed install is retried.
set -euo pipefail
cd "$(dirname "$0")"
git fetch -q origin main
target=$(git rev-parse origin/main)
installed=$(cat "$HOME/mirror/.installed" 2>/dev/null || true)
if [ "$target" != "$installed" ]; then
  echo "Installing $target"
  git reset -q --hard origin/main
  exec bash install.sh
fi
