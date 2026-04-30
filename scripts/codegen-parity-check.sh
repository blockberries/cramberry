#!/usr/bin/env bash
# End-to-end cross-language byte-parity check for code-generated
# encoders. Generates Go and Rust code from a schema, compiles both,
# encodes the same logical data, and asserts the byte streams are
# identical.
#
# Until this test existed there was no end-to-end coverage that
# `cramberry generate -lang go` and `cramberry generate -lang rust`
# produce wire-equivalent output for the same input. The existing
# integration tests use hand-rolled encoders, not generated code.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BIN="$REPO_ROOT/bin/cramberry"
[[ -x "$BIN" ]] || { echo "$BIN not built; run make build" >&2; exit 1; }

SCHEMA="$REPO_ROOT/scripts/parity_fixture.cram"
[[ -f "$SCHEMA" ]] || { echo "missing fixture schema: $SCHEMA" >&2; exit 1; }

WORK="$(mktemp -d -t cramberry-parity.XXXXXX)"
trap 'rm -rf "$WORK"' EXIT

mkdir -p "$WORK/go" "$WORK/rust/src"
"$BIN" generate -lang go -json=false -out "$WORK/go" "$SCHEMA" >/dev/null
"$BIN" generate -lang rust              -out "$WORK/rust/src" "$SCHEMA" >/dev/null

# --- Go side ---
gofmt -w "$WORK/go"/*.go
cat > "$WORK/go/go.mod" <<EOF
module parity

go 1.21

require github.com/blockberries/cramberry v0.0.0
replace github.com/blockberries/cramberry => $REPO_ROOT
EOF

# Move the generated file's `package` declaration into the binary.
genpkg="$(grep -E '^package ' "$WORK/go"/*.go | head -1 | awk '{print $2}')"

# Drop a small probe in the same package as the generated code; it
# encodes a hand-rolled fixture and prints the resulting bytes.
cat > "$WORK/go/probe.go" <<EOF
package $genpkg

import (
	"encoding/hex"
	"fmt"

	"github.com/blockberries/cramberry/pkg/cramberry"
)

func fixture() *Sample {
	return &Sample{
		Active:  true,
		Count:   -42,
		Amount:  123456789012,
		Name:    "hello, 世界!",
		Payload: []byte{0xde, 0xad, 0xbe, 0xef},
		Ratio:   3.14159,
		Tags:    []string{"alpha", "beta", "gamma"},
	}
}

// CodegenBytes encodes via the codegen-emitted EncodeTo method.
func CodegenBytes() string {
	s := fixture()
	w := cramberry.GetWriter()
	defer cramberry.PutWriter(w)
	s.EncodeTo(w)
	if w.Err() != nil {
		panic(fmt.Sprintf("codegen encode err: %v", w.Err()))
	}
	return hex.EncodeToString(w.BytesCopy())
}

// ReflectionBytes encodes via the reflection marshaller.
func ReflectionBytes() string {
	s := fixture()
	data, err := cramberry.Marshal(s)
	if err != nil {
		panic(fmt.Sprintf("reflection encode err: %v", err))
	}
	return hex.EncodeToString(data)
}
EOF

mkdir -p "$WORK/go/cmd"
cat > "$WORK/go/cmd/main.go" <<EOF
package main

import (
	"fmt"

	parity "parity"
)

func main() {
	fmt.Println("CODEGEN", parity.CodegenBytes())
	fmt.Println("REFLECT", parity.ReflectionBytes())
}
EOF

(cd "$WORK/go" && go mod tidy >/dev/null 2>&1 && go build -o probe ./cmd >/dev/null)
GO_OUT="$("$WORK/go/probe")"
GO_CODEGEN_BYTES="$(echo "$GO_OUT" | awk '/^CODEGEN/ {print $2}')"
GO_REFLECT_BYTES="$(echo "$GO_OUT" | awk '/^REFLECT/ {print $2}')"

# --- Rust side ---
cat > "$WORK/rust/Cargo.toml" <<EOF
[package]
name = "parity_rust"
version = "0.1.0"
edition = "2021"

[dependencies]
cramberry = { path = "$REPO_ROOT/rust" }
serde = { version = "1.0", features = ["derive"] }
serde_json = "1.0"
hex = "0.4"

[[bin]]
name = "probe"
path = "src/main.rs"
EOF

# Move the generated rust file to lib.rs so we can import its symbols.
mv "$WORK/rust/src"/*.rs "$WORK/rust/src/lib.rs"

cat > "$WORK/rust/src/main.rs" <<'EOF'
use cramberry::Writer;
use parity_rust::*;

fn main() {
    let s = Sample {
        active:  true,
        count:   -42,
        amount:  123456789012,
        name:    "hello, 世界!".to_string(),
        payload: vec![0xde, 0xad, 0xbe, 0xef],
        ratio:   3.14159,
        tags:    vec!["alpha".to_string(), "beta".to_string(), "gamma".to_string()],
    };
    let mut w = Writer::new();
    encode_sample(&mut w, &s).unwrap();
    println!("{}", hex::encode(w.as_bytes()));
}
EOF

(cd "$WORK/rust" && cargo build --bin probe --quiet 2>/dev/null)
RUST_BYTES="$("$WORK/rust/target/debug/probe")"

# --- Compare ---
fail=0
if [[ "$GO_REFLECT_BYTES" != "$GO_CODEGEN_BYTES" ]]; then
    echo "FAIL  Go reflection != Go codegen" >&2
    echo "  reflection: $GO_REFLECT_BYTES"   >&2
    echo "  codegen:    $GO_CODEGEN_BYTES"   >&2
    fail=1
fi
if [[ "$GO_CODEGEN_BYTES" != "$RUST_BYTES" ]]; then
    echo "FAIL  Go codegen != Rust codegen" >&2
    echo "  Go:   $GO_CODEGEN_BYTES"         >&2
    echo "  Rust: $RUST_BYTES"               >&2
    fail=1
fi
if [[ $fail -eq 0 ]]; then
    echo "  OK  Go reflection == Go codegen == Rust codegen"
    echo "      bytes: $GO_CODEGEN_BYTES"
fi
exit $fail
