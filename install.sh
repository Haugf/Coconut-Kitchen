#!/usr/bin/env bash
# Builds and installs the mirror on the Pi, straight from this repo.
#   git clone https://github.com/Haugf/coconut-kitchen ~/mirror-src
#   bash ~/mirror-src/install.sh
# To update later:  cd ~/mirror-src && git pull && bash install.sh
set -euo pipefail
cd "$(dirname "$0")"

if ! command -v go >/dev/null; then
  echo "Installing Go"
  sudo apt-get update -qq
  sudo apt-get install -y golang-go
fi

echo "Building the server"
(cd backend && go mod tidy && go build -o ../mirror-server .)

echo "Installing to ~/mirror"
mkdir -p "$HOME/mirror"
rm -rf "$HOME/mirror/web"
cp -r frontend/dist "$HOME/mirror/web"
cp mirror-server deploy/setup-pi.sh "$HOME/mirror/"
if [ ! -f "$HOME/mirror/config.json" ]; then
  cp backend/config.example.json "$HOME/mirror/config.json"
  echo "Created ~/mirror/config.json. Add your calendar address there."
fi

if systemctl list-unit-files mirror.service >/dev/null 2>&1 && [ -f /etc/systemd/system/mirror.service ]; then
  sudo systemctl restart mirror.service
  echo "Updated and restarted."
else
  bash "$HOME/mirror/setup-pi.sh"
fi
