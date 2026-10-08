#!/usr/bin/env bash
# Builds and installs the mirror on the Pi, straight from this repo.
#   git clone https://github.com/Haugf/coconut-kitchen ~/mirror-src
#   bash ~/mirror-src/install.sh
# To update later:  cd ~/mirror-src && git pull && bash install.sh
set -euo pipefail
cd "$(dirname "$0")"

# The server needs Go 1.22 or newer. Use apt's Go if it's new enough,
# otherwise install the official arm64 build into /usr/local/go.
go_ok() { command -v go >/dev/null && go version | grep -Eq 'go1\.(2[2-9]|[3-9][0-9])'; }
export PATH="/usr/local/go/bin:$PATH"
if ! go_ok; then
  echo "Installing Go"
  sudo apt-get update -qq
  sudo apt-get install -y golang-go || true
fi
if ! go_ok; then
  echo "apt's Go is too old, installing Go 1.23 from go.dev"
  curl -fsSL https://go.dev/dl/go1.23.4.linux-arm64.tar.gz -o /tmp/go.tgz
  sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tgz
fi
go version

echo "Building the server"
(cd backend && go mod tidy && go build -o ../mirror-server .)

echo "Installing to ~/mirror"
mkdir -p "$HOME/mirror"
rm -rf "$HOME/mirror/web"
cp -r frontend/dist "$HOME/mirror/web"
# The server is running, so a plain copy fails with "Text file busy".
# Copy beside it and rename over it instead; the running process keeps
# the old file until the restart below.
cp mirror-server "$HOME/mirror/mirror-server.new"
mv -f "$HOME/mirror/mirror-server.new" "$HOME/mirror/mirror-server"
cp deploy/setup-pi.sh "$HOME/mirror/"
if [ ! -f "$HOME/mirror/config.json" ]; then
  cp backend/config.example.json "$HOME/mirror/config.json"
  echo "Created ~/mirror/config.json. Add your calendar address there."
fi

if systemctl list-unit-files mirror.service >/dev/null 2>&1 && [ -f /etc/systemd/system/mirror.service ]; then
  sudo systemctl restart mirror.service
  # Close the kiosk browser; kiosk.sh reopens it with the new version.
  pkill -f -- '--app=http://localhost:8080' || true
  echo "Updated and restarted."
else
  bash "$HOME/mirror/setup-pi.sh"
fi

# Record what's installed, so update.sh retries if this run failed.
git rev-parse HEAD > "$HOME/mirror/.installed" 2>/dev/null || true
