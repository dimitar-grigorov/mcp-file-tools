#!/usr/bin/env bash
# Lint .github/workflows with pinned, checksummed actionlint + shellcheck. Linux only.
set -euo pipefail

ACTIONLINT_VERSION=1.7.12
ACTIONLINT_SHA256=8aca8db96f1b94770f1b0d72b6dddcb1ebb8123cb3712530b08cc387b349a3d8
SHELLCHECK_VERSION=0.11.0
SHELLCHECK_SHA256=b7af85e41cc99489dcc21d66c6d5f3685138f06d34651e6d34b42ec6d54fe6f6

tools_dir="$(mktemp -d)"
trap 'rm -rf "$tools_dir"' EXIT

actionlint_archive="$tools_dir/actionlint.tar.gz"
shellcheck_archive="$tools_dir/shellcheck.tar.gz"

curl -fSL --retry 3 --retry-all-errors \
  "https://github.com/rhysd/actionlint/releases/download/v${ACTIONLINT_VERSION}/actionlint_${ACTIONLINT_VERSION}_linux_amd64.tar.gz" \
  -o "$actionlint_archive"
printf '%s  %s\n' "$ACTIONLINT_SHA256" "$actionlint_archive" | sha256sum -c -

curl -fSL --retry 3 --retry-all-errors \
  "https://github.com/koalaman/shellcheck/releases/download/v${SHELLCHECK_VERSION}/shellcheck-v${SHELLCHECK_VERSION}.linux.x86_64.tar.gz" \
  -o "$shellcheck_archive"
printf '%s  %s\n' "$SHELLCHECK_SHA256" "$shellcheck_archive" | sha256sum -c -

mkdir -p "$tools_dir/actionlint"
tar -xzf "$actionlint_archive" -C "$tools_dir/actionlint"
tar -xzf "$shellcheck_archive" -C "$tools_dir"

PATH="$tools_dir/actionlint:$tools_dir/shellcheck-v${SHELLCHECK_VERSION}:$PATH" actionlint
