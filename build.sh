#!/usr/bin/env bash
# Local IMGE build. Usage: ./build.sh [OUTPUT]
#
# OUTPUT is the path of the built CLI binary; it defaults to ./imge.
#
# The runtime version is "<version file>[-<dev suffix>]":
#   * a release build (HEAD is tagged AND the working tree is clean) keeps the plain
#     "vX.Y.Z" from the version file;
#   * anything else — an untagged commit, or uncommitted/untracked changes on top of a
#     tagged commit — is a local dev build and gets a short random hex suffix, so each
#     rebuild is distinguishable even when the semantic version has not been bumped.
set -euo pipefail
cd "$(dirname "$0")"

out="${1:-imge}"

if git describe --exact-match --tags HEAD >/dev/null 2>&1 && [ -z "$(git status --porcelain)" ]; then
	echo "release build (tagged $(git describe --tags HEAD), clean tree) -> $out"
	go build -o "$out" ./cmd/imge
else
	suffix="$(printf '%05x' $(( (RANDOM << 15 | RANDOM) & 0xFFFFF )))"
	echo "dev build (suffix -${suffix}) -> $out"
	go build -ldflags "-X github.com/EnesBaytekin/imge.buildTag=${suffix}" -o "$out" ./cmd/imge
fi
