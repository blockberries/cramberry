//! Cramberry cross-runtime interoperability tests for Rust.
//!
//! These tests verify that Rust runtime produces identical binary
//! encodings to Go and can decode Go-generated golden files.

pub mod interop;

#[cfg(test)]
mod interop_test;

pub use interop::*;
