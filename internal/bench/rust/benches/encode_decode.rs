//! Criterion benchmarks for the Rust port of Cramberry, with prost (Google
//! protobuf) and serde_json as baselines.
//!
//! Layout mirrors `internal/bench/benchmark_test.go`:
//!   - SmallMessage  — minimal three-field baseline.
//!   - Metrics       — scalar-heavy numeric record.
//!   - Person        — deeply nested with optional fields.
//!   - Document      — arrays + maps + nested messages.
//!   - Event         — maps + bytes payload.
//!   - Batch100/1000 — large repeated arrays.
//!
//! Run with:
//!     cargo bench --bench encode_decode
//!
//! Or filter:
//!     cargo bench --bench encode_decode -- SmallMessage/encode

use criterion::{black_box, criterion_group, criterion_main, BatchSize, Criterion};
use cramberry::{Reader, Writer};
use cramberry_bench::{cramgen as cg, fixtures as fx, pbgen as pb};
use prost::Message;

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

fn cram_encode<F>(w: &mut Writer, mut f: F)
where
    F: FnMut(&mut Writer),
{
    w.reset();
    f(w);
}

// ---------------------------------------------------------------------------
// SmallMessage
// ---------------------------------------------------------------------------

fn bench_small_message(c: &mut Criterion) {
    let mut group = c.benchmark_group("SmallMessage");

    // Cramberry codegen — encode
    {
        let msg = fx::cram_small_message();
        let mut w = Writer::new();
        group.bench_function("cramberry/encode", |b| {
            b.iter(|| {
                cram_encode(&mut w, |w| {
                    cg::encode_small_message(w, black_box(&msg)).unwrap();
                });
                black_box(w.as_bytes());
            })
        });
    }

    // Cramberry codegen — decode
    {
        let msg = fx::cram_small_message();
        let mut w = Writer::new();
        cg::encode_small_message(&mut w, &msg).unwrap();
        let bytes = w.into_bytes();
        group.bench_function("cramberry/decode", |b| {
            b.iter(|| {
                let mut r = Reader::new(black_box(&bytes));
                black_box(cg::decode_small_message(&mut r).unwrap());
            })
        });
    }

    // Protobuf — encode
    {
        let msg = fx::pb_small_message();
        let mut buf = Vec::with_capacity(64);
        group.bench_function("protobuf/encode", |b| {
            b.iter(|| {
                buf.clear();
                black_box(&msg).encode(&mut buf).unwrap();
                black_box(&buf);
            })
        });
    }

    // Protobuf — decode
    {
        let msg = fx::pb_small_message();
        let mut buf = Vec::with_capacity(64);
        msg.encode(&mut buf).unwrap();
        group.bench_function("protobuf/decode", |b| {
            b.iter(|| {
                let decoded = pb::SmallMessage::decode(black_box(&buf[..])).unwrap();
                black_box(decoded);
            })
        });
    }

    // serde_json — encode
    {
        let msg = fx::cram_small_message();
        group.bench_function("json/encode", |b| {
            b.iter(|| {
                black_box(serde_json::to_vec(black_box(&msg)).unwrap());
            })
        });
    }

    // serde_json — decode
    {
        let msg = fx::cram_small_message();
        let bytes = serde_json::to_vec(&msg).unwrap();
        group.bench_function("json/decode", |b| {
            b.iter(|| {
                let v: cg::SmallMessage = serde_json::from_slice(black_box(&bytes)).unwrap();
                black_box(v);
            })
        });
    }

    group.finish();
}

// ---------------------------------------------------------------------------
// Metrics
// ---------------------------------------------------------------------------

fn bench_metrics(c: &mut Criterion) {
    let mut group = c.benchmark_group("Metrics");

    let msg = fx::cram_metrics();
    let mut w = Writer::new();
    group.bench_function("cramberry/encode", |b| {
        b.iter(|| {
            cram_encode(&mut w, |w| {
                cg::encode_metrics(w, black_box(&msg)).unwrap();
            });
            black_box(w.as_bytes());
        })
    });

    let mut w2 = Writer::new();
    cg::encode_metrics(&mut w2, &msg).unwrap();
    let bytes = w2.into_bytes();
    group.bench_function("cramberry/decode", |b| {
        b.iter(|| {
            let mut r = Reader::new(black_box(&bytes));
            black_box(cg::decode_metrics(&mut r).unwrap());
        })
    });

    let pb_msg = fx::pb_metrics();
    let mut buf = Vec::with_capacity(128);
    group.bench_function("protobuf/encode", |b| {
        b.iter(|| {
            buf.clear();
            black_box(&pb_msg).encode(&mut buf).unwrap();
            black_box(&buf);
        })
    });

    let mut pbuf = Vec::with_capacity(128);
    pb_msg.encode(&mut pbuf).unwrap();
    group.bench_function("protobuf/decode", |b| {
        b.iter(|| {
            black_box(pb::Metrics::decode(black_box(&pbuf[..])).unwrap());
        })
    });

    group.bench_function("json/encode", |b| {
        b.iter(|| black_box(serde_json::to_vec(black_box(&msg)).unwrap()))
    });
    let json_bytes = serde_json::to_vec(&msg).unwrap();
    group.bench_function("json/decode", |b| {
        b.iter(|| {
            let v: cg::Metrics = serde_json::from_slice(black_box(&json_bytes)).unwrap();
            black_box(v);
        })
    });

    group.finish();
}

// ---------------------------------------------------------------------------
// Person
// ---------------------------------------------------------------------------

fn bench_person(c: &mut Criterion) {
    let mut group = c.benchmark_group("Person");

    let msg = fx::cram_person();
    let mut w = Writer::new();
    group.bench_function("cramberry/encode", |b| {
        b.iter(|| {
            cram_encode(&mut w, |w| {
                cg::encode_person(w, black_box(&msg)).unwrap();
            });
            black_box(w.as_bytes());
        })
    });

    let mut w2 = Writer::new();
    cg::encode_person(&mut w2, &msg).unwrap();
    let bytes = w2.into_bytes();
    group.bench_function("cramberry/decode", |b| {
        b.iter(|| {
            let mut r = Reader::new(black_box(&bytes));
            black_box(cg::decode_person(&mut r).unwrap());
        })
    });

    let pb_msg = fx::pb_person();
    let mut buf = Vec::with_capacity(256);
    group.bench_function("protobuf/encode", |b| {
        b.iter(|| {
            buf.clear();
            black_box(&pb_msg).encode(&mut buf).unwrap();
            black_box(&buf);
        })
    });
    let mut pbuf = Vec::with_capacity(256);
    pb_msg.encode(&mut pbuf).unwrap();
    group.bench_function("protobuf/decode", |b| {
        b.iter(|| {
            black_box(pb::Person::decode(black_box(&pbuf[..])).unwrap());
        })
    });

    group.bench_function("json/encode", |b| {
        b.iter(|| black_box(serde_json::to_vec(black_box(&msg)).unwrap()))
    });
    let json_bytes = serde_json::to_vec(&msg).unwrap();
    group.bench_function("json/decode", |b| {
        b.iter(|| {
            let v: cg::Person = serde_json::from_slice(black_box(&json_bytes)).unwrap();
            black_box(v);
        })
    });

    group.finish();
}

// ---------------------------------------------------------------------------
// Document
// ---------------------------------------------------------------------------

fn bench_document(c: &mut Criterion) {
    let mut group = c.benchmark_group("Document");

    let msg = fx::cram_document();
    let mut w = Writer::new();
    group.bench_function("cramberry/encode", |b| {
        b.iter(|| {
            cram_encode(&mut w, |w| {
                cg::encode_document(w, black_box(&msg)).unwrap();
            });
            black_box(w.as_bytes());
        })
    });

    let mut w2 = Writer::new();
    cg::encode_document(&mut w2, &msg).unwrap();
    let bytes = w2.into_bytes();
    group.bench_function("cramberry/decode", |b| {
        b.iter(|| {
            let mut r = Reader::new(black_box(&bytes));
            black_box(cg::decode_document(&mut r).unwrap());
        })
    });

    let pb_msg = fx::pb_document();
    let mut buf = Vec::with_capacity(512);
    group.bench_function("protobuf/encode", |b| {
        b.iter(|| {
            buf.clear();
            black_box(&pb_msg).encode(&mut buf).unwrap();
            black_box(&buf);
        })
    });
    let mut pbuf = Vec::with_capacity(512);
    pb_msg.encode(&mut pbuf).unwrap();
    group.bench_function("protobuf/decode", |b| {
        b.iter(|| {
            black_box(pb::Document::decode(black_box(&pbuf[..])).unwrap());
        })
    });

    group.bench_function("json/encode", |b| {
        b.iter(|| black_box(serde_json::to_vec(black_box(&msg)).unwrap()))
    });
    let json_bytes = serde_json::to_vec(&msg).unwrap();
    group.bench_function("json/decode", |b| {
        b.iter(|| {
            let v: cg::Document = serde_json::from_slice(black_box(&json_bytes)).unwrap();
            black_box(v);
        })
    });

    group.finish();
}

// ---------------------------------------------------------------------------
// Event
// ---------------------------------------------------------------------------

fn bench_event(c: &mut Criterion) {
    let mut group = c.benchmark_group("Event");

    let msg = fx::cram_event();
    let mut w = Writer::new();
    group.bench_function("cramberry/encode", |b| {
        b.iter(|| {
            cram_encode(&mut w, |w| {
                cg::encode_event(w, black_box(&msg)).unwrap();
            });
            black_box(w.as_bytes());
        })
    });

    let mut w2 = Writer::new();
    cg::encode_event(&mut w2, &msg).unwrap();
    let bytes = w2.into_bytes();
    group.bench_function("cramberry/decode", |b| {
        b.iter(|| {
            let mut r = Reader::new(black_box(&bytes));
            black_box(cg::decode_event(&mut r).unwrap());
        })
    });

    let pb_msg = fx::pb_event();
    let mut buf = Vec::with_capacity(256);
    group.bench_function("protobuf/encode", |b| {
        b.iter(|| {
            buf.clear();
            black_box(&pb_msg).encode(&mut buf).unwrap();
            black_box(&buf);
        })
    });
    let mut pbuf = Vec::with_capacity(256);
    pb_msg.encode(&mut pbuf).unwrap();
    group.bench_function("protobuf/decode", |b| {
        b.iter(|| {
            black_box(pb::Event::decode(black_box(&pbuf[..])).unwrap());
        })
    });

    group.bench_function("json/encode", |b| {
        b.iter(|| black_box(serde_json::to_vec(black_box(&msg)).unwrap()))
    });
    let json_bytes = serde_json::to_vec(&msg).unwrap();
    group.bench_function("json/decode", |b| {
        b.iter(|| {
            let v: cg::Event = serde_json::from_slice(black_box(&json_bytes)).unwrap();
            black_box(v);
        })
    });

    group.finish();
}

// ---------------------------------------------------------------------------
// Batch{100,1000}
// ---------------------------------------------------------------------------

fn bench_batch(c: &mut Criterion, size: usize) {
    let mut group = c.benchmark_group(format!("Batch{}", size));

    // Long-running benches: shrink sample size so the suite stays under
    // a reasonable wall-clock budget. Criterion's defaults oversample
    // for small workloads.
    if size >= 1000 {
        group.sample_size(20);
    } else {
        group.sample_size(50);
    }

    let msg = fx::cram_batch_request(size);
    let mut w = Writer::new();
    group.bench_function("cramberry/encode", |b| {
        b.iter(|| {
            cram_encode(&mut w, |w| {
                cg::encode_batch_request(w, black_box(&msg)).unwrap();
            });
            black_box(w.as_bytes());
        })
    });

    let mut w2 = Writer::new();
    cg::encode_batch_request(&mut w2, &msg).unwrap();
    let bytes = w2.into_bytes();
    group.bench_function("cramberry/decode", |b| {
        b.iter_batched_ref(
            || (),
            |_| {
                let mut r = Reader::new(black_box(&bytes));
                black_box(cg::decode_batch_request(&mut r).unwrap());
            },
            BatchSize::SmallInput,
        )
    });

    let pb_msg = fx::pb_batch_request(size);
    let mut buf = Vec::with_capacity(bytes.len() + 64);
    group.bench_function("protobuf/encode", |b| {
        b.iter(|| {
            buf.clear();
            black_box(&pb_msg).encode(&mut buf).unwrap();
            black_box(&buf);
        })
    });
    let mut pbuf = Vec::with_capacity(bytes.len() + 64);
    pb_msg.encode(&mut pbuf).unwrap();
    group.bench_function("protobuf/decode", |b| {
        b.iter(|| {
            black_box(pb::BatchRequest::decode(black_box(&pbuf[..])).unwrap());
        })
    });

    group.bench_function("json/encode", |b| {
        b.iter(|| black_box(serde_json::to_vec(black_box(&msg)).unwrap()))
    });
    let json_bytes = serde_json::to_vec(&msg).unwrap();
    group.bench_function("json/decode", |b| {
        b.iter(|| {
            let v: cg::BatchRequest = serde_json::from_slice(black_box(&json_bytes)).unwrap();
            black_box(v);
        })
    });

    group.finish();
}

fn bench_batch_100(c: &mut Criterion) { bench_batch(c, 100); }
fn bench_batch_1000(c: &mut Criterion) { bench_batch(c, 1000); }

criterion_group!(
    benches,
    bench_small_message,
    bench_metrics,
    bench_person,
    bench_document,
    bench_event,
    bench_batch_100,
    bench_batch_1000,
);
criterion_main!(benches);
