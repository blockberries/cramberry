#!/usr/bin/env bash
# Compile-check the Go output of `cramberry generate -lang go` for every
# example / testdata schema. The pkg/cramberry tests already cover one
# committed fixture; this script makes sure every other schema in the
# repo also produces compilable Go.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$REPO_ROOT/bin/cramberry"
[[ -x "$BIN" ]] || { echo "$BIN not built; run make build" >&2; exit 1; }

WORK="$(mktemp -d -t cramberry-go-codegen.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

fail=0
for src in "$REPO_ROOT"/examples/schemas/*.cram "$REPO_ROOT"/testdata/schemas/*.cram; do
    [[ -e "$src" ]] || continue
    name="$(basename "$src" .cram)"
    out="$WORK/$name"
    mkdir -p "$out"
    "$BIN" generate -lang go -out "$out" "$src" >/dev/null

    # Each generated file declares its own package; build it in
    # isolation so packages don't collide.
    cat > "$out/go.mod" <<EOF
module cramberry-go-codegen-check/$name

go 1.21

require github.com/blockberries/cramberry v0.0.0

replace github.com/blockberries/cramberry => $REPO_ROOT
EOF

    if (cd "$out" && go mod tidy >/dev/null 2>&1 && go build ./... 2>&1) ; then
        echo "  OK  $name"
    else
        echo "FAIL  $name" >&2
        fail=1
    fi
done

exit $fail
