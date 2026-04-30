#!/usr/bin/env bash
# Compile-check the Rust output of `cramberry generate -lang rust` for
# every example / testdata schema. Catches generator regressions that
# unit tests on Go side don't notice (e.g. emitting code that doesn't
# typecheck against the Rust runtime).
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$REPO_ROOT/bin/cramberry"
[[ -x "$BIN" ]] || { echo "$BIN not built; run make build" >&2; exit 1; }

WORK="$(mktemp -d -t cramberry-rust-codegen.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

fail=0
for src in "$REPO_ROOT"/examples/schemas/*.cram "$REPO_ROOT"/testdata/schemas/*.cram; do
    [[ -e "$src" ]] || continue
    name="$(basename "$src" .cram)"
    out="$WORK/$name"
    mkdir -p "$out"
    "$BIN" generate -lang rust -out "$out" "$src" >/dev/null
    rsfile="$(ls "$out"/*.rs 2>/dev/null | head -1)"
    [[ -n "$rsfile" ]] || { echo "no .rs produced for $src" >&2; fail=1; continue; }

    cat > "$out/Cargo.toml" <<TOML
[package]
name = "rust_codegen_check_$(echo $name | tr -c 'a-z0-9' _)"
version = "0.1.0"
edition = "2021"
[dependencies]
cramberry = { path = "$REPO_ROOT/rust" }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
[lib]
path = "$(basename "$rsfile")"
TOML

    # The generated code is consumed via `pub mod cramgen { #![allow(...)] include!(...) }`
    # in real crates; here we compile the file as a top-level lib, so suppress
    # warnings that the generator structurally cannot avoid (`mut` locals only
    # written-then-overwritten on the decode path, etc.).
    if (cd "$out" && RUSTFLAGS="-A unused_imports -A unused_mut -A unused_variables -A unused_assignments -A unreachable_patterns -A unused_parens" cargo build --quiet 2>&1) ; then
        echo "  OK  $name"
    else
        echo "FAIL  $name" >&2
        fail=1
    fi
done

exit $fail
