#!/usr/bin/env bash
# One-line setup on a fresh Pi:
#   curl -fsSL https://raw.githubusercontent.com/Haugf/coconut-kitchen/main/bootstrap.sh | bash
set -euo pipefail
command -v git >/dev/null || { sudo apt-get update -qq && sudo apt-get install -y git; }
if [ -d "$HOME/mirror-src/.git" ]; then
  git -C "$HOME/mirror-src" pull -q
else
  git clone -q https://github.com/Haugf/coconut-kitchen "$HOME/mirror-src"
fi
bash "$HOME/mirror-src/install.sh"
echo
echo "All set. Rebooting in 5 seconds to start the mirror."
sleep 5
sudo reboot
