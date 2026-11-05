# CLVX Keyboard (Omarchy)

Omarchy plugin + resident daemon for the Clevetura CLVX / TouchOnKeys
keyboard, over USB and Bluetooth. No profiles, no AI training — just
customization: touch sliders, touchpad zones and effects, per-zone
backlight color, F-row, device info, and safe reset/restart.

## What it is

- **Bar widget** — battery %, charging, transport (USB/BLE); click opens the panel.
- **Panel** — sliders (left/right sensitivity + Volume / Screen Brightness /
  Keyboard Brightness), touchpad effect + preset colors, 4-zone keyboard map
  with per-zone color, per-setting reset buttons backed by factory defaults.
- **`clvxd` daemon (Go)** — 100 ms heartbeat, auto-reconnect across USB/BLE,
  Unix-socket JSON API (`status`, `get_settings`, `set_global`,
  `get_profile_settings`, `set_profile_settings`, `device_info`,
  `default_settings`, `set_os_mode`, `statistics`, `diagnostics`,
  `restart`, guarded `factory_reset`). Typed protobuf codecs preserve unknown
  fields byte-exact on every write.

## Requirements

- Omarchy (quickshell shell) with third-party plugins enabled.
- Go toolchain (`sudo pacman -S go`) to build the daemon.
- The keyboard paired over Bluetooth (`bluetoothctl`) and/or USB.
- `plugdev` group membership for `/dev/hidraw` USB access (needs relogin).

## Install

```sh
./install.sh            # builds clvxd/clvx, installs user service, enables plugin deps
```

Privileged USB setup runs separately (the script tells you how —
it must run from your own session, then requires logout/login):

```sh
./install-udev.sh       # needs sudo: udev rule + plugdev group
```

Then restart the shell once so it picks up the new QML
(`omarchy plugin disable/enable` is not enough — quickshell caches
compiled panels for the life of the process).

## Socket API

`$XDG_RUNTIME_DIR/clvx/clvx.sock`, newline-delimited JSON
`{"id":N,"method":"...","params":{...}}`.

| method | params | result |
|---|---|---|
| `status` | — | connection, battery, transport, power/backlight mode |
| `get_settings` | — | `AppSettings` (global + counter) |
| `set_global` | `{"global":{...partial...}}` | merged settings (read-modify-write) |
| `get_profile_settings` | `{"profileId":0}` | gestures, touchZone, backlight, keyboard + `raw` |
| `set_profile_settings` | `{"settings":{...}}` | typed edits merged into raw bytes |
| `device_info` | — | CLVX S details, fw version, serial, kbm info |
| `default_settings` | — | factory defaults (read-only, reset source) |
| `set_os_mode` | `{"mode":"LINUX"}` | OS mode status |
| `restart` / `factory_reset` | `{"confirm":true}` for reset | status |
| `statistics` / `diagnostics` | — | counters (firmware may not answer) |

## Repo layout

- `BarWidget.qml`, `Panel.qml`, `Service.qml`, `manifest.json` — the plugin.
- `go/` — vendored daemon source (`clvxd`, `clvx` CLI).
- `config/clvxd.service`, `config/99-clvx-hidraw.rules` — systemd unit, udev rule.
- `install.sh`, `install-udev.sh`, `uninstall.sh`.

Wire protocol notes (USB framing, BLE framing, field maps, firmware-update
map) live in the reverse-engineering notes alongside `go/`.

## Troubleshooting

- Panel shows old UI after update: restart the shell (QML cache).
- `clvxd socket is unavailable`: `systemctl --user status clvxd`.
- USB permission errors: check `groups` for `plugdev`; relogin after joining.
- Volume/brightness keys feel laggy: the companion `clvx-volume` and
  `clvx-brightness-display` helpers in `~/.local/bin` are the fast path
  (see your Hyprland `bindings.lua` overrides).

## License

MIT
