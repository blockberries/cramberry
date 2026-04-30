// Compile the protobuf benchmark schema with prost-build.
// Requires `protoc` on PATH at build time.

fn main() {
    println!("cargo:rerun-if-changed=../schemas/messages.proto");
    let mut config = prost_build::Config::new();
    config.bytes(["."]);
    config
        .compile_protos(&["../schemas/messages.proto"], &["../schemas"])
        .expect("prost-build failed; ensure protoc is installed (brew install protobuf)");
}
