#!/usr/bin/env bash
# Privileged CLVX udev/group setup. Run through sudo, never from install.sh.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if (( EUID != 0 )); then
  printf 'Run as root: sudo bash %q\n' "$here/install-udev.sh" >&2
  exit 1
fi

target_user="${SUDO_USER:-}"
if [[ -z "$target_user" || "$target_user" == root ]] || ! id "$target_user" >/dev/null 2>&1; then
  printf 'Run with sudo from the regular user session so SUDO_USER identifies that user.\n' >&2
  exit 1
fi

getent group plugdev >/dev/null 2>&1 || groupadd -r plugdev
usermod -aG plugdev "$target_user"
install -m644 "$here/config/99-clvx-hidraw.rules" /etc/udev/rules.d/99-clvx-hidraw.rules
udevadm control --reload-rules
udevadm trigger --subsystem-match=hidraw --action=add

printf 'udev rules installed and %s added to plugdev.\n' "$target_user"
printf 'Log out of the desktop session and back in, then re-run install.sh as %s.\n' "$target_user"
