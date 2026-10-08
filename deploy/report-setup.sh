#!/usr/bin/env bash
# One-time: lets the Pi publish its status to GitHub (branch pi-status).
#   bash ~/mirror/report-setup.sh
set -euo pipefail
KEY="$HOME/.ssh/mirror_status"
mkdir -p "$HOME/.ssh"
if [ ! -f "$KEY" ]; then
  ssh-keygen -q -t ed25519 -N "" -C "mirror-status" -f "$KEY"
fi
echo
echo "1. Open: https://github.com/Haugf/coconut-kitchen/settings/keys/new"
echo "2. Title: mirror status"
echo "3. Paste this key, tick 'Allow write access', and click Add key:"
echo
cat "$KEY.pub"
echo
read -r -p "Press Enter once the key is added... " _
if bash "$HOME/mirror/push-status.sh"; then
  echo "Status published. It refreshes every 5 minutes on the pi-status branch."
else
  echo "Publishing failed. Check that the key was added with write access, then run this again."
fi
