//! Fixture builders shared between the criterion benches.
//!
//! The values mirror `benchmark_test.go::makeCramberry*` / `makeProtobuf*`
//! so the wire output, allocation counts, and decode work are directly
//! comparable to the Go side.

use std::collections::HashMap;

use crate::{cramgen as cg, pbgen as pb};

// ---------------------------------------------------------------------------
// Cramberry fixtures
// ---------------------------------------------------------------------------

pub fn cram_small_message() -> cg::SmallMessage {
    cg::SmallMessage {
        id: 12345,
        name: "test-item".into(),
        active: true,
    }
}

pub fn cram_point() -> cg::Point {
    cg::Point { x: 123.456, y: 789.012, z: 345.678 }
}

pub fn cram_timestamp() -> cg::Timestamp {
    cg::Timestamp { seconds: 1705900800, nanos: 123456789 }
}

pub fn cram_metrics() -> cg::Metrics {
    cg::Metrics {
        count: 1_000_000,
        sum: 12_345_678.90,
        min: 0.001,
        max: 99_999.99,
        avg: 12_345.67,
        p50: 10_000.0,
        p95: 50_000.0,
        p99: 90_000.0,
        total_bytes: 1_073_741_824,
        error_count: 42,
    }
}

pub fn cram_address() -> cg::Address {
    cg::Address {
        street1: "123 Main Street".into(),
        street2: Some("Suite 100".into()),
        city: "San Francisco".into(),
        state: "CA".into(),
        postal_code: "94105".into(),
        country: "USA".into(),
        coordinates: Some(cram_point()),
    }
}

pub fn cram_contact_info() -> cg::ContactInfo {
    cg::ContactInfo {
        email: "john.doe@example.com".into(),
        phone: Some("+1-555-123-4567".into()),
        mobile: Some("+1-555-987-6543".into()),
        fax: None,
        mailing_address: Some(cram_address()),
        billing_address: None,
    }
}

pub fn cram_person() -> cg::Person {
    cg::Person {
        id: 1001,
        first_name: "John".into(),
        last_name: "Doe".into(),
        middle_name: Some("Robert".into()),
        date_of_birth: Some(cram_timestamp()),
        contact: cram_contact_info(),
        status: cg::Status::Active,
        created_at: cram_timestamp(),
        updated_at: Some(cram_timestamp()),
    }
}

pub fn cram_document() -> cg::Document {
    let mut metadata = HashMap::new();
    metadata.insert("source".into(), "import".into());
    metadata.insert("encoding".into(), "utf-8".into());
    metadata.insert("version".into(), "1.0".into());

    cg::Document {
        id: 2001,
        title: "Important Document Title".into(),
        content: "This is the document content with some meaningful text that would typically be much longer in a real application.".into(),
        author_id: 1001,
        status: cg::Status::Active,
        priority: cg::Priority::High,
        tags: vec![
            cg::Tag { key: "category".into(), value: "technical".into(), color: None },
            cg::Tag { key: "status".into(),   value: "reviewed".into(),  color: None },
            cg::Tag { key: "version".into(),  value: "2.0".into(),       color: None },
        ],
        attachments: vec![cg::Attachment {
            id: "att-001".into(),
            filename: "report.pdf".into(),
            mime_type: "application/pdf".into(),
            size_bytes: 1_048_576,
            checksum: vec![0xde, 0xad, 0xbe, 0xef],
            url: None,
            uploaded_at: cram_timestamp(),
        }],
        comments: vec![cg::Comment {
            id: 3001,
            author_id: 1002,
            content: "Great document!".into(),
            created_at: cram_timestamp(),
            edited_at: None,
            reactions: vec![1001, 1003, 1004],
        }],
        metadata,
        collaborators: vec![1001, 1002, 1003],
        created_at: cram_timestamp(),
        updated_at: Some(cram_timestamp()),
        published_at: Some(cram_timestamp()),
    }
}

pub fn cram_event() -> cg::Event {
    let mut attributes = HashMap::new();
    attributes.insert("user_id".into(), "1001".into());
    attributes.insert("action".into(), "create".into());

    cg::Event {
        id: "evt-001".into(),
        r#type: cg::EventType::Created,
        entity_type: "document".into(),
        entity_id: "doc-2001".into(),
        source: cg::EventSource {
            service: "document-service".into(),
            instance: "prod-01".into(),
            version: "1.2.3".into(),
            region: Some("us-west-2".into()),
        },
        timestamp: cram_timestamp(),
        attributes,
        payload: Some(b"{\"action\":\"click\",\"element\":\"button\"}".to_vec()),
        correlation_id: Some("corr-123".into()),
        causation_id: Some("caus-456".into()),
    }
}

pub fn cram_batch_request(size: usize) -> cg::BatchRequest {
    let mut items = Vec::with_capacity(size);
    for i in 0..size {
        items.push(cg::SmallMessage {
            id: i as i64,
            name: "batch-item".into(),
            active: i % 2 == 0,
        });
    }

    let mut headers = HashMap::new();
    headers.insert("Content-Type".into(), "application/x-cramberry".into());
    headers.insert("X-Request-Id".into(), "req-123".into());

    cg::BatchRequest {
        request_id: "batch-001".into(),
        items,
        headers,
        submitted_at: cram_timestamp(),
        timeout: None,
        priority: cg::Priority::Medium,
    }
}

// ---------------------------------------------------------------------------
// Protobuf (prost) fixtures
// ---------------------------------------------------------------------------

pub fn pb_small_message() -> pb::SmallMessage {
    pb::SmallMessage { id: 12345, name: "test-item".into(), active: true }
}

pub fn pb_point() -> pb::Point {
    pb::Point { x: 123.456, y: 789.012, z: 345.678 }
}

pub fn pb_timestamp() -> pb::Timestamp {
    pb::Timestamp { seconds: 1705900800, nanos: 123456789 }
}

pub fn pb_metrics() -> pb::Metrics {
    pb::Metrics {
        count: 1_000_000,
        sum: 12_345_678.90,
        min: 0.001,
        max: 99_999.99,
        avg: 12_345.67,
        p50: 10_000.0,
        p95: 50_000.0,
        p99: 90_000.0,
        total_bytes: 1_073_741_824,
        error_count: 42,
    }
}

pub fn pb_address() -> pb::Address {
    pb::Address {
        street1: "123 Main Street".into(),
        street2: Some("Suite 100".into()),
        city: "San Francisco".into(),
        state: "CA".into(),
        postal_code: "94105".into(),
        country: "USA".into(),
        coordinates: Some(pb_point()),
    }
}

pub fn pb_contact_info() -> pb::ContactInfo {
    pb::ContactInfo {
        email: "john.doe@example.com".into(),
        phone: Some("+1-555-123-4567".into()),
        mobile: Some("+1-555-987-6543".into()),
        fax: None,
        mailing_address: Some(pb_address()),
        billing_address: None,
    }
}

pub fn pb_person() -> pb::Person {
    pb::Person {
        id: 1001,
        first_name: "John".into(),
        last_name: "Doe".into(),
        middle_name: Some("Robert".into()),
        date_of_birth: Some(pb_timestamp()),
        contact: Some(pb_contact_info()),
        status: pb::Status::Active as i32,
        created_at: Some(pb_timestamp()),
        updated_at: Some(pb_timestamp()),
    }
}

pub fn pb_document() -> pb::Document {
    let mut metadata = HashMap::new();
    metadata.insert("source".into(), "import".into());
    metadata.insert("encoding".into(), "utf-8".into());
    metadata.insert("version".into(), "1.0".into());

    pb::Document {
        id: 2001,
        title: "Important Document Title".into(),
        content: "This is the document content with some meaningful text that would typically be much longer in a real application.".into(),
        author_id: 1001,
        status: pb::Status::Active as i32,
        priority: pb::Priority::High as i32,
        tags: vec![
            pb::Tag { key: "category".into(), value: "technical".into(), color: None },
            pb::Tag { key: "status".into(),   value: "reviewed".into(),  color: None },
            pb::Tag { key: "version".into(),  value: "2.0".into(),       color: None },
        ],
        attachments: vec![pb::Attachment {
            id: "att-001".into(),
            filename: "report.pdf".into(),
            mime_type: "application/pdf".into(),
            size_bytes: 1_048_576,
            checksum: bytes::Bytes::from_static(&[0xde, 0xad, 0xbe, 0xef]),
            url: None,
            uploaded_at: Some(pb_timestamp()),
        }],
        comments: vec![pb::Comment {
            id: 3001,
            author_id: 1002,
            content: "Great document!".into(),
            created_at: Some(pb_timestamp()),
            edited_at: None,
            reactions: vec![1001, 1003, 1004],
        }],
        metadata,
        collaborators: vec![1001, 1002, 1003],
        created_at: Some(pb_timestamp()),
        updated_at: Some(pb_timestamp()),
        published_at: Some(pb_timestamp()),
    }
}

pub fn pb_event() -> pb::Event {
    let mut attributes = HashMap::new();
    attributes.insert("user_id".into(), "1001".into());
    attributes.insert("action".into(), "create".into());

    pb::Event {
        id: "evt-001".into(),
        r#type: pb::EventType::Created as i32,
        entity_type: "document".into(),
        entity_id: "doc-2001".into(),
        source: Some(pb::EventSource {
            service: "document-service".into(),
            instance: "prod-01".into(),
            version: "1.2.3".into(),
            region: Some("us-west-2".into()),
        }),
        timestamp: Some(pb_timestamp()),
        attributes,
        payload: Some(bytes::Bytes::from_static(b"{\"action\":\"click\",\"element\":\"button\"}")),
        correlation_id: Some("corr-123".into()),
        causation_id: Some("caus-456".into()),
    }
}

pub fn pb_batch_request(size: usize) -> pb::BatchRequest {
    let mut items = Vec::with_capacity(size);
    for i in 0..size {
        items.push(pb::SmallMessage {
            id: i as i64,
            name: "batch-item".into(),
            active: i % 2 == 0,
        });
    }

    let mut headers = HashMap::new();
    headers.insert("Content-Type".into(), "application/x-cramberry".into());
    headers.insert("X-Request-Id".into(), "req-123".into());

    pb::BatchRequest {
        request_id: "batch-001".into(),
        items,
        headers,
        submitted_at: Some(pb_timestamp()),
        timeout: None,
        priority: pb::Priority::Medium as i32,
    }
}
