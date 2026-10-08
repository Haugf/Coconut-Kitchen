#!/usr/bin/env bash
# Run by the mirror-update timer. Pulls from GitHub and reinstalls only
# when there's something new.
set -euo pipefail
cd "$(dirname "$0")"
git fetch -q origin main
if [ "$(git rev-parse HEAD)" != "$(git rev-parse origin/main)" ]; then
  echo "New version found, updating"
  git reset -q --hard origin/main
  bash install.sh
fi
