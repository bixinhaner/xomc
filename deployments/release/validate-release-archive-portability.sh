#!/usr/bin/env bash
set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
build_script="$script_dir/build-release.sh"

archive_command='^[[:space:]]*\([[:space:]]*cd "\$WORK"[[:space:]]*&&[[:space:]]*tar --no-xattrs \$TAR_OPT '
grep -Eq "$archive_command" "$build_script" || {
  echo "FAIL: release tar command must omit host extended attributes with --no-xattrs" >&2
  exit 1
}

probe_dir="$(mktemp -d)"
trap 'rm -rf "$probe_dir"' EXIT
mkdir "$probe_dir/payload"
: >"$probe_dir/payload/probe"
tar --no-xattrs -cf "$probe_dir/probe.tar" -C "$probe_dir" payload
tar -tf "$probe_dir/probe.tar" | grep -Fxq 'payload/probe'

echo "Release archive portability validation passed"
