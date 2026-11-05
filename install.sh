#!/usr/bin/env bash
# Build and install the CLVX daemon for the invoking user. Safe to re-run.
set -euo pipefail
here="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
say() { printf '\033[1;32m==>\033[0m %s\n' "$*"; }
die() { printf '\033[1;31m==> error:\033[0m %s\n' "$*" >&2; exit 1; }

(( EUID != 0 )) || die "run as your regular user; only install-udev.sh should run as root"
command -v go >/dev/null 2>&1 || die "Go is required (pacman package 'go'); install it with: sudo pacman -S go"
command -v systemctl >/dev/null 2>&1 || die "systemctl is required"
[[ -f "$here/go/go.mod" ]] || die "vendored Go sources are missing from $here/go"

user="$(id -un)"
needs_root=0
cmp -s "$here/config/99-clvx-hidraw.rules" /etc/udev/rules.d/99-clvx-hidraw.rules || needs_root=1
getent group plugdev >/dev/null 2>&1 || needs_root=1
[[ " $(id -nG "$user") " == *" plugdev "* ]] || needs_root=1
if (( needs_root )); then
  say "privileged udev/group setup is required; run this exact command in a tmux session:"
  printf 'sudo bash %q\n' "$here/install-udev.sh"
  printf 'Then log out and back in, and re-run %q.\n' "$here/install.sh"
  exit 1
fi
if [[ " $(id -nG) " != *" plugdev "* ]]; then
  die "plugdev membership is configured but absent from this session; log out and back in, then re-run install.sh"
fi

mkdir -p "$HOME/.local/bin"
say "building clvxd and clvx from $here/go"
(
  cd "$here/go"
  go build -o "$HOME/.local/bin/clvxd" ./cmd/clvxd
  go build -o "$HOME/.local/bin/clvx" ./cmd/clvx
)

say "installing the clvxd user service"
install -Dm644 "$here/config/clvxd.service" "$HOME/.config/systemd/user/clvxd.service"
systemctl --user daemon-reload
systemctl --user enable --now clvxd.service

say "done; daemon status: systemctl --user status clvxd.service"
