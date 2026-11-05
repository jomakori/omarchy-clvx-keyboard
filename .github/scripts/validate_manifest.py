#!/usr/bin/env python3
"""Validate manifest.json: schema essentials + semver + referenced files exist."""
import json
import re
import sys
from pathlib import Path

root = Path(__file__).resolve().parent.parent.parent
manifest = json.loads((root / "manifest.json").read_text())

errors = []


def err(msg):
    errors.append(msg)


for key in ("schemaVersion", "id", "name", "version", "kinds", "entryPoints"):
    if key not in manifest:
        err(f"missing key: {key}")

if not re.fullmatch(r"\d+\.\d+\.\d+", str(manifest.get("version", ""))):
    err(f"version is not strict semver X.Y.Z: {manifest.get('version')!r}")

if not re.fullmatch(r"[a-z0-9]+(\.[a-z0-9]+)+", str(manifest.get("id", ""))):
    err(f"id is not reverse-DNS: {manifest.get('id')!r}")

valid_kinds = {"bar-widget", "panel", "overlay", "menu", "service", "bar"}
for kind in manifest.get("kinds", []):
    if kind not in valid_kinds:
        err(f"unknown kind: {kind}")

for _, entry in manifest.get("entryPoints", {}).items():
    if not (root / entry).is_file():
        err(f"entry point missing: {entry}")

if errors:
    print("\n".join(errors), file=sys.stderr)
    sys.exit(1)
print("manifest OK:", manifest["id"], manifest["version"])
