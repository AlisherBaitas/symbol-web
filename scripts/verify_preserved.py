#!/usr/bin/env python3
"""Проверяет неизменность файлов исходной команды, кроме заполненного README."""

import hashlib
from pathlib import Path
import subprocess
import sys

BASE = "76b2ff96c1ded7eaa04df02fec4c457a0b09d0f5"
root = Path(__file__).resolve().parents[1]
tree = subprocess.check_output(["git", "ls-tree", "-rz", BASE], cwd=root)
failed = []
checked = 0
for entry in tree.split(b"\0"):
    if not entry:
        continue
    metadata, raw_path = entry.split(b"\t", 1)
    mode, kind, expected = metadata.split()
    path = raw_path.decode()
    if path == "README.md":
        continue
    target = root / path
    if kind != b"blob" or not target.is_file() or target.is_symlink():
        failed.append(path)
        continue
    data = target.read_bytes()
    actual = hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest().encode()
    if actual != expected:
        failed.append(path)
    checked += 1
if failed:
    print("ИЗМЕНЕНЫ ИСХОДНЫЕ ФАЙЛЫ:", ", ".join(failed))
    sys.exit(1)
print(f"OK: {checked} исходных файлов сохранены побайтно; README заполнен по запросу.")
