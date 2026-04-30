//! Cross-language benchmark harness — Rust side.
//!
//! Exposes the Cramberry-generated codecs and the prost-generated protobuf
//! codecs as a single library so the criterion benches in `benches/` can
//! exercise both for the same logical data.

pub mod cramgen {
    #![allow(clippy::all)]
    #![allow(unused_imports, unused_mut, unused_variables, unused_assignments, unreachable_patterns, unused_parens)]
    include!("messages.rs");
}

pub mod pbgen {
    include!(concat!(env!("OUT_DIR"), "/benchmark.rs"));
}

pub mod fixtures;
