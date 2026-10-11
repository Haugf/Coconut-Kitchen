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
cp deploy/setup-pi.sh deploy/mirror-status.sh deploy/push-status.sh deploy/report-setup.sh deploy/mirror.sh "$HOME/mirror/"
chmod +x "$HOME/mirror/mirror.sh" "$HOME/mirror/mirror-status.sh"
# The `mirror` command. ~/.local/bin is on the PATH on Raspberry Pi OS
# (from the next login once the folder exists); no sudo needed.
mkdir -p "$HOME/.local/bin"
ln -sf "$HOME/mirror/mirror.sh" "$HOME/.local/bin/mirror"
if [ ! -f "$HOME/mirror/config.json" ]; then
  cp backend/config.example.json "$HOME/mirror/config.json"
  echo "Created ~/mirror/config.json. Add your calendar address there."
fi

# Automatic updates run without a terminal, so they can't answer a sudo
# password prompt. Routine updates only need to restart the server, which
# setup-pi.sh allows without a password. The full setup (services, kiosk,
# timers) needs sudo, so it only runs when setup-pi.sh itself changed, and
# only when sudo can work: from a terminal, or with passwordless sudo.
setup_hash=$(sha256sum deploy/setup-pi.sh | cut -d' ' -f1)
if [ "$setup_hash" != "$(cat "$HOME/mirror/.setup-hash" 2>/dev/null)" ]; then
  if [ -t 0 ] || sudo -n true 2>/dev/null; then
    bash "$HOME/mirror/setup-pi.sh"
    echo "$setup_hash" > "$HOME/mirror/.setup-hash"
  else
    echo "NOTE: setup changed and needs your password once. Run:  bash ~/mirror-src/update.sh"
  fi
fi

if sudo -n systemctl restart mirror.service 2>/dev/null; then
  :
elif [ -t 0 ]; then
  sudo systemctl restart mirror.service
else
  echo "ERROR: can't restart the server without a password. Run once from a terminal:  bash ~/mirror-src/update.sh"
  exit 1
fi
# No need to touch the browser: the page notices the new build within a
# minute and reloads itself.
echo "Installed $(git rev-parse --short HEAD 2>/dev/null). The screen refreshes within a minute."

# Record what's installed, so update.sh retries if this run failed.
git rev-parse HEAD > "$HOME/mirror/.installed" 2>/dev/null || true
