#!/usr/bin/env bash
# Compile-check the TypeScript output of `cramberry generate -lang typescript`
# for every example / testdata schema. Catches generator regressions that
# Go-side unit tests don't see — e.g. emitting code that doesn't pass
# `tsc --strict` against the TS runtime.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$REPO_ROOT/bin/cramberry"
[[ -x "$BIN" ]] || { echo "$BIN not built; run make build" >&2; exit 1; }

WORK="$(mktemp -d -t cramberry-ts-codegen.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

# Reuse the typescript/ workspace's installed tsc + dependencies.
ln -s "$REPO_ROOT/typescript/node_modules" "$WORK/node_modules"

cat > "$WORK/tsconfig.json" <<EOF
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "paths": {
      "@cramberry/runtime": ["$REPO_ROOT/typescript/src/index.ts"]
    },
    "baseUrl": "."
  },
  "include": ["**/*.ts"]
}
EOF

for src in "$REPO_ROOT"/examples/schemas/*.cram "$REPO_ROOT"/testdata/schemas/*.cram; do
    [[ -e "$src" ]] || continue
    name="$(basename "$src" .cram)"
    out="$WORK/$name"
    mkdir -p "$out"
    "$BIN" generate -lang typescript -out "$out" "$src" >/dev/null
done

cd "$WORK"
if npx --no -- tsc -p tsconfig.json ; then
    for d in */ ; do echo "  OK  ${d%/}" ; done
else
    echo "FAIL: tsc errors in generated TypeScript output" >&2
    exit 1
fi
