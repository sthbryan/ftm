#!/usr/bin/env bash
# Writes shell completions and the man page for an ftm binary into a folder.
# Usage: scripts/gen-completions.sh <path-to-ftm> <out-dir>
set -euo pipefail

bin=${1:?path to the ftm binary}
out=${2:?output folder}

mkdir -p "$out/completions" "$out/man"
"$bin" completion bash > "$out/completions/ftm.bash"
"$bin" completion zsh > "$out/completions/_ftm"
"$bin" completion fish > "$out/completions/ftm.fish"
"$bin" completion powershell > "$out/completions/ftm.ps1"
"$bin" man > "$out/man/ftm.1"
