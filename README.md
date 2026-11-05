# Omarchy CLVX plugin

Omarchy Quattro integration for the Clevetura CLVX keyboard. The `clvxd` user daemon maintains the keyboard heartbeat and exposes connection, battery, backlight, and active-profile state over a Unix socket for the plugin UI. The `clvx` CLI provides device and diagnostic commands.

The keyboard supports **USB or Bluetooth, one at a time**. Switch its mode manually; this plugin and daemon never switch the keyboard mode. The daemon uses `--transport auto` to discover whichever mode is active.

## Requirements

- Omarchy Quattro with systemd user services.
- Go (Arch/pacman package `go`) to build the daemon and CLI.
- A CLVX connected in either USB mode or Bluetooth mode.
- Membership in `plugdev`. The privileged installer adds this; **log out of the desktop session and back in** before starting the daemon.

The Go module is vendored in this repository under `go/` (`go.mod`, `go.sum`, `cmd/`, and `internal/`). No separate source checkout or network fetch of the CLVX source is needed at install time.

## Install

Omarchy's plugin installer copies and validates plugins but **does not run `install.sh`**. Run it manually from this checkout:

```sh
./install.sh
```

If udev rules or group membership need setup, the installer stops and prints the exact command required. Run that command in a **tmux session**; it has this form (use the path printed by the installer):

```sh
sudo bash /path/to/omarchy-plugin-clvx/install-udev.sh
```

The root helper creates `plugdev` if needed, adds the invoking user, installs `/etc/udev/rules.d/99-clvx-hidraw.rules`, reloads udev, and triggers hidraw devices. It never builds or starts the daemon as root.

Log out and back in after being added to `plugdev`, then re-run `./install.sh`. It builds `clvxd` and `clvx` into `~/.local/bin`, installs `~/.config/systemd/user/clvxd.service`, and enables/starts that user unit. Installation is safe to re-run.

## Status and troubleshooting

Check the service and recent daemon errors:

```sh
systemctl --user status clvxd.service
journalctl --user -u clvxd.service -n 50
clvx status
```

Inspect the daemon status JSON's `error` field. The socket is normally `$XDG_RUNTIME_DIR/clvx/clvx.sock` (fallback: `/tmp/clvx-$USER.sock`); if it is missing, check that `clvxd.service` is active and inspect its journal.

- **`EACCES` / permission denied:** group membership usually is not active in the current desktop session. Confirm `id -nG` contains `plugdev`, then log out and back in (restarting only the daemon is not enough). Also verify `/etc/udev/rules.d/99-clvx-hidraw.rules` exists.
- **No device, timeout, or vendor-characteristic error:** check which mode the keyboard is physically in. Use USB mode for hidraw USB discovery or pair/connect it in Bluetooth mode for BLE. It cannot use both simultaneously; reconnect after manually changing modes.
- **Wrong transport selected:** the installed unit uses `--transport auto`. For a direct USB diagnostic use `clvx --transport usb heartbeat`; for Bluetooth use `clvx --transport ble heartbeat`.

## Uninstall

As the regular user:

```sh
./uninstall.sh
```

This disables and removes `clvxd.service` and deletes `~/.local/bin/clvxd` and `~/.local/bin/clvx`. It deliberately leaves the system udev rule, `plugdev` group, and group memberships unchanged.

To remove device access as a separate root step, run:

```sh
sudo rm -f /etc/udev/rules.d/99-clvx-hidraw.rules
sudo udevadm control --reload-rules
sudo udevadm trigger --subsystem-match=hidraw --action=add
```
