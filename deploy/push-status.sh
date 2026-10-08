#!/usr/bin/env bash
# Publishes mirror-status to the repo's pi-status branch so it can be read
# remotely (by you, or by Claude) without SSH. Runs every 5 minutes once
# report-setup.sh has created a deploy key; does nothing before that.
# What's published: checks, counts and error messages. No event titles,
# calendar addresses or API keys.
set -euo pipefail
KEY="$HOME/.ssh/mirror_status"
[ -f "$KEY" ] || exit 0

DIR="$HOME/.mirror-status"
export GIT_SSH_COMMAND="ssh -i $KEY -o IdentitiesOnly=yes -o StrictHostKeyChecking=accept-new"
rm -rf "$DIR" && mkdir -p "$DIR" && cd "$DIR"
git init -q -b pi-status

bash "$HOME/mirror/mirror-status.sh" --json > status.json
bash "$HOME/mirror/mirror-status.sh" > status.txt
journalctl -u mirror -u mirror-update -n 80 --no-pager -o short-iso 2>/dev/null > recent.log || true

git add -A
git -c user.name="mirror" -c user.email="mirror@localhost" commit -q -m "Status $(date '+%Y-%m-%d %H:%M')"
# One commit, replaced each time, so the branch never grows.
git push -q -f git@github.com:Haugf/coconut-kitchen.git pi-status
