#!/usr/bin/env bash
# Remove the per-user CLVX daemon. The system udev rule is intentionally kept.
set -euo pipefail

if (( EUID == 0 )); then
  printf 'Run uninstall.sh as the regular user, not root.\n' >&2
  exit 1
fi

systemctl --user disable --now clvxd.service 2>/dev/null || true
rm -f "$HOME/.config/systemd/user/clvxd.service"
systemctl --user daemon-reload
rm -f "$HOME/.local/bin/clvxd" "$HOME/.local/bin/clvx"

printf 'Removed clvxd.service and the clvxd/clvx binaries.\n'
printf 'The udev rule and plugdev membership remain; see README.md for optional removal.\n'
