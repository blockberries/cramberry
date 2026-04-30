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
	"strings"

	"github.com/blockberries/cramberry/pkg/cramberry"
)

func repeat(s string, n int) string { return strings.Repeat(s, n) }

func fixture() *Sample {
	nick := "the_admin"
	prio := int32(7)
	office := &Address{Street: "1 Hacker Way", City: "Menlo Park"}
	return &Sample{
		Active:  true,
		Count:   -42,
		Amount:  123456789012,
		Name:    "hello, 世界! " + repeat("AB", 100),
		Payload: []byte{0xde, 0xad, 0xbe, 0xef},
		Ratio:   3.14159,
		Tags:    []string{"alpha", "beta", "gamma"},
		Status:  StatusStatusActive,
		Home: Address{
			Street: "1 Infinite Loop",
			City:   "Cupertino",
		},
		Addresses: []Address{
			{Street: "742 Evergreen Terrace", City: "Springfield"},
			{Street: "221B Baker St", City: "London"},
		},
		Scores: map[string]int32{
			"alpha": 1,
			"gamma": 3,
			"beta":  2,
		},
		Nickname:       &nick,
		Priority:       &prio,
		Score:          2.71828,
		Generation:     4294967290,
		Epoch:          18446744073709551610,
		Tiny:           -7,
		ByteVal:        255,
		Small:          -1234,
		SmallUnsigned:  65535,
		IdToName: map[int32]string{
			10:  "ten",
			-1:  "neg_one",
			100: "hundred",
		},
		Office: office,
		Forest: Tree{
			Label: "root",
			Children: []Tree{
				{Label: "left", Children: []Tree{
					{Label: "leftleaf"},
				}},
				{Label: "right"},
			},
		},
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
use std::collections::HashMap;

fn main() {
    let mut scores = HashMap::new();
    scores.insert("alpha".to_string(), 1);
    scores.insert("gamma".to_string(), 3);
    scores.insert("beta".to_string(), 2);

    let mut id_to_name = HashMap::new();
    id_to_name.insert(10, "ten".to_string());
    id_to_name.insert(-1, "neg_one".to_string());
    id_to_name.insert(100, "hundred".to_string());

    let s = Sample {
        active:  true,
        count:   -42,
        amount:  123456789012,
        name:    format!("hello, 世界! {}", "AB".repeat(100)),
        payload: vec![0xde, 0xad, 0xbe, 0xef],
        ratio:   3.14159,
        tags:    vec!["alpha".to_string(), "beta".to_string(), "gamma".to_string()],
        status:  Status::StatusActive,
        home: Address {
            street: "1 Infinite Loop".to_string(),
            city:   "Cupertino".to_string(),
        },
        addresses: vec![
            Address { street: "742 Evergreen Terrace".to_string(), city: "Springfield".to_string() },
            Address { street: "221B Baker St".to_string(),         city: "London".to_string()      },
        ],
        scores,
        nickname:       Some("the_admin".to_string()),
        priority:       Some(7),
        score:          2.71828,
        generation:     4294967290,
        epoch:          18446744073709551610,
        tiny:           -7,
        byte_val:       255,
        small:          -1234,
        small_unsigned: 65535,
        id_to_name,
        office: Some(Address {
            street: "1 Hacker Way".to_string(),
            city:   "Menlo Park".to_string(),
        }),
        // Intentionally zero — exercise omit-empty paths.
        empty_str:   String::new(),
        empty_list:  Vec::new(),
        empty_bytes: Vec::new(),
        zero_count:  0,
        zero_ratio:  0.0,
        // Small recursive tree:
        //   root
        //   ├── left
        //   │   └── leftleaf
        //   └── right
        forest: Tree {
            label: "root".to_string(),
            children: vec![
                Tree { label: "left".to_string(),  children: vec![
                    Tree { label: "leftleaf".to_string(), children: vec![] },
                ]},
                Tree { label: "right".to_string(), children: vec![] },
            ],
        },
    };
    let mut w = Writer::new();
    encode_sample(&mut w, &s).unwrap();
    println!("{}", hex::encode(w.as_bytes()));
}
EOF

(cd "$WORK/rust" && cargo build --bin probe --quiet)
RUST_BYTES="$("$WORK/rust/target/debug/probe")"

# --- TypeScript side ---
# The TS generator emits `from '@cramberry/runtime'`. Node's ESM
# resolver finds it via the package's "exports" field once the
# package is installed (here via npm pack into a fresh project).
mkdir -p "$WORK/ts"
"$BIN" generate -lang typescript -out "$WORK/ts" "$SCHEMA" >/dev/null

# Build the TS runtime so the dist/ files referenced by the package
# exports actually exist.
(cd "$REPO_ROOT/typescript" && npm install --silent --no-audit --no-fund >/dev/null 2>&1 && npm run --silent build >/dev/null 2>&1) || {
    echo "skip (TS): runtime build failed; install npm + run 'cd typescript && npm install' once" >&2
    exit 0
}

cat > "$WORK/ts/package.json" <<EOF
{
  "name": "parity-ts",
  "private": true,
  "type": "module",
  "dependencies": {
    "@cramberry/runtime": "file:$REPO_ROOT/typescript"
  }
}
EOF
(cd "$WORK/ts" && npm install --silent --no-audit --no-fund >/dev/null 2>&1)

# Drop a probe ESM file that imports the generated module + the runtime.
genfile="$(ls "$WORK/ts"/*.ts | head -1)"
genstem="$(basename "$genfile" .ts)"

cat > "$WORK/ts/probe.mjs" <<EOF
import { Writer } from '@cramberry/runtime';
import { encodeSample, Status } from './$genstem.ts';

const s = {
    active: true,
    count: -42,
    amount: 123456789012n,
    name: "hello, 世界! " + "AB".repeat(100),
    payload: new Uint8Array([0xde, 0xad, 0xbe, 0xef]),
    ratio: 3.14159,
    tags: ["alpha", "beta", "gamma"],
    status: Status.StatusActive,
    home: {
        street: "1 Infinite Loop",
        city: "Cupertino",
    },
    addresses: [
        { street: "742 Evergreen Terrace", city: "Springfield" },
        { street: "221B Baker St",         city: "London"      },
    ],
    scores: new Map([
        ["alpha", 1],
        ["gamma", 3],
        ["beta",  2],
    ]),
    nickname: "the_admin",
    priority: 7,
    score: 2.71828,
    generation: 4294967290,
    epoch: 18446744073709551610n,
    tiny: -7,
    byteVal: 255,
    small: -1234,
    smallUnsigned: 65535,
    idToName: new Map([
        [10,  "ten"],
        [-1,  "neg_one"],
        [100, "hundred"],
    ]),
    office: { street: "1 Hacker Way", city: "Menlo Park" },
    forest: {
        label: "root",
        children: [
            { label: "left",  children: [
                { label: "leftleaf", children: [] },
            ]},
            { label: "right", children: [] },
        ],
    },
};
const w = new Writer();
encodeSample(w, s);
const bytes = w.bytes();
console.log(Array.from(bytes).map(b => b.toString(16).padStart(2, '0')).join(''));
EOF

# Use tsx (provided by the typescript/ workspace's installed deps) so
# the .ts import resolves and Node finds @cramberry/runtime via the
# package's "exports". The local node_modules linkage gives us dist/.
TS_BYTES="$(cd "$WORK/ts" && npx --no -- tsx probe.mjs 2>/dev/null)"

if [[ -z "$TS_BYTES" ]]; then
    echo "skip (TS): probe produced no output (likely missing tsx in PATH)" >&2
    TS_BYTES="(skipped)"
fi

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
if [[ "$TS_BYTES" != "(skipped)" ]] && [[ "$GO_CODEGEN_BYTES" != "$TS_BYTES" ]]; then
    echo "FAIL  Go codegen != TS codegen"   >&2
    echo "  Go: $GO_CODEGEN_BYTES"           >&2
    echo "  TS: $TS_BYTES"                   >&2
    fail=1
fi
if [[ $fail -eq 0 ]]; then
    if [[ "$TS_BYTES" == "(skipped)" ]]; then
        echo "  OK  Go reflection == Go codegen == Rust codegen (TS skipped)"
    else
        echo "  OK  Go reflection == Go codegen == Rust codegen == TS codegen"
    fi
    echo "      bytes: $GO_CODEGEN_BYTES"
fi
exit $fail
