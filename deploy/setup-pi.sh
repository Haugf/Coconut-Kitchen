#!/usr/bin/env bash
# Run once on the Pi (after the first `make deploy`):  bash ~/mirror/setup-pi.sh
# Assumes 64-bit Raspberry Pi OS (Bookworm or newer) with the default labwc desktop.
set -euo pipefail

MIRROR_DIR="$HOME/mirror"
ROTATION="${ROTATION:-normal}"   # normal = horizontal; 90 or 270 once the monitor is turned vertical
# 720p because the current micro HDMI cable flickers at 1080p.
# With a better cable, rerun with MODE=1920x1080@60Hz.
MODE="${MODE:-1280x720@60Hz}"

echo "Installing backend service"
sudo tee /etc/systemd/system/mirror.service >/dev/null <<UNIT
[Unit]
Description=Mirror backend
After=network-online.target
Wants=network-online.target

[Service]
User=$USER
WorkingDirectory=$MIRROR_DIR
ExecStart=$MIRROR_DIR/mirror-server -config config.json -static web
Restart=always
RestartSec=3

[Install]
WantedBy=multi-user.target
UNIT
sudo systemctl daemon-reload
sudo systemctl enable --now mirror.service

echo "Turning off screen blanking"
sudo raspi-config nonint do_blanking 1 || true

echo "Writing kiosk launcher"
# Written beside the old one and renamed, so a kiosk.sh that's already
# running keeps working until the next boot.
cat > "$MIRROR_DIR/kiosk.sh.new" <<KIOSK
#!/usr/bin/env bash
OUTPUT=\$(wlr-randr | awk '/^HDMI/ {print \$1; exit}')
if [ -n "\$OUTPUT" ]; then
  wlr-randr --output "\$OUTPUT" --mode $MODE || true
  wlr-randr --output "\$OUTPUT" --transform $ROTATION
fi
until curl -fs http://localhost:8080 >/dev/null; do sleep 1; done
BROWSER=\$(command -v chromium || command -v chromium-browser)
# Relaunch the browser if it ever crashes. Wait for any leftover
# Chromium to fully exit first, or the new one quits straight away.
while true; do
  while pgrep -x chromium >/dev/null || pgrep -x chromium-browser >/dev/null; do sleep 1; done
  "\$BROWSER" --disable-session-crashed-bubble --kiosk --noerrdialogs --disable-infobars --no-first-run \\
    --password-store=basic --check-for-update-interval=31536000 \\
    --app=http://localhost:8080
  sleep 2
done
KIOSK
chmod +x "$MIRROR_DIR/kiosk.sh.new"
mv -f "$MIRROR_DIR/kiosk.sh.new" "$MIRROR_DIR/kiosk.sh"

mkdir -p "$HOME/.config/labwc"
AUTOSTART="$HOME/.config/labwc/autostart"
touch "$AUTOSTART"
grep -q "mirror/kiosk.sh" "$AUTOSTART" || echo "$MIRROR_DIR/kiosk.sh &" >> "$AUTOSTART"

SRC_DIR="${SRC_DIR:-$HOME/mirror-src}"
if [ -d "$SRC_DIR/.git" ]; then
  echo "Turning on auto-update from GitHub (checks every 5 minutes)"
  sudo tee /etc/systemd/system/mirror-update.service >/dev/null <<UNIT
[Unit]
Description=Pull the mirror from GitHub and reinstall if it changed
After=network-online.target
Wants=network-online.target

[Service]
Type=oneshot
User=$USER
WorkingDirectory=$SRC_DIR
Environment=HOME=$HOME
ExecStart=/usr/bin/env bash $SRC_DIR/update.sh
UNIT
  sudo tee /etc/systemd/system/mirror-update.timer >/dev/null <<UNIT
[Unit]
Description=Check GitHub for mirror updates

[Timer]
OnBootSec=2min
OnUnitActiveSec=5min

[Install]
WantedBy=timers.target
UNIT
  sudo systemctl daemon-reload
  sudo systemctl enable --now mirror-update.timer
fi

echo "Setup done. On a first install, reboot to start the mirror:  sudo reboot"
