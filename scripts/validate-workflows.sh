#!/usr/bin/env bash
# actionlint from tools/go.mod plus a pinned shellcheck. Linux only: shellcheck has no Go module.
set -euo pipefail

SHELLCHECK_VERSION=0.11.0
SHELLCHECK_SHA256=b7af85e41cc99489dcc21d66c6d5f3685138f06d34651e6d34b42ec6d54fe6f6

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

tools_dir="$(mktemp -d)"
trap 'rm -rf "$tools_dir"' EXIT

shellcheck_archive="$tools_dir/shellcheck.tar.gz"

curl -fSL --retry 3 --retry-all-errors \
  "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.gz" \
  -o "$shellcheck_archive"
printf '%s  %s\n' "$SHELLCHECK_SHA256" "$shellcheck_archive" | sha256sum -c -

tar -xzf "$shellcheck_archive" -C "$tools_dir"

cd "$repo_root"
PATH="$tools_dir/shellcheck-v${SHELLCHECK_VERSION}:$PATH" \
  go tool -modfile=tools/go.mod actionlint
