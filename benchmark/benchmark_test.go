// Package benchmark provides comprehensive performance comparisons between
// Cramberry, Protocol Buffers, and JSON serialization formats.
package benchmark

import (
	"encoding/json"
	"testing"

	cramgen "github.com/blockberries/cramberry/benchmark/gen/cramberry"
	pb "github.com/blockberries/cramberry/benchmark/gen/protobuf"
	"google.golang.org/protobuf/proto"
)

// ============================================================================
// Test Data Construction - Cramberry Types
// ============================================================================

func makeCramberrySmallMessage() *cramgen.SmallMessage {
	return &cramgen.SmallMessage{
		Id:     12345,
		Name:   "test-item",
		Active: true,
	}
}

func makeCramberryPoint() *cramgen.Point {
	return &cramgen.Point{
		X: 123.456,
		Y: 789.012,
		Z: 345.678,
	}
}

func makeCramberryTimestamp() *cramgen.Timestamp {
	return &cramgen.Timestamp{
		Seconds: 1705900800,
		Nanos:   123456789,
	}
}

func makeCramberryMetrics() *cramgen.Metrics {
	return &cramgen.Metrics{
		Count:      1000000,
		Sum:        12345678.90,
		Min:        0.001,
		Max:        99999.99,
		Avg:        12345.67,
		P50:        10000.0,
		P95:        50000.0,
		P99:        90000.0,
		TotalBytes: 1073741824,
		ErrorCount: 42,
	}
}

func makeCramberryAddress() *cramgen.Address {
	street2 := "Suite 100"
	coords := makeCramberryPoint()
	return &cramgen.Address{
		Street1:     "123 Main Street",
		Street2:     &street2,
		City:        "San Francisco",
		State:       "CA",
		PostalCode:  "94105",
		Country:     "USA",
		Coordinates: coords,
	}
}

func makeCramberryContactInfo() *cramgen.ContactInfo {
	phone := "+1-555-123-4567"
	mobile := "+1-555-987-6543"
	addr := makeCramberryAddress()
	return &cramgen.ContactInfo{
		Email:          "john.doe@example.com",
		Phone:          &phone,
		Mobile:         &mobile,
		MailingAddress: addr,
	}
}

func makeCramberryPerson() *cramgen.Person {
	middle := "Robert"
	dob := makeCramberryTimestamp()
	updated := makeCramberryTimestamp()
	return &cramgen.Person{
		Id:          1001,
		FirstName:   "John",
		LastName:    "Doe",
		MiddleName:  &middle,
		DateOfBirth: dob,
		Contact:     *makeCramberryContactInfo(),
		Status:      cramgen.StatusActive,
		CreatedAt:   *makeCramberryTimestamp(),
		UpdatedAt:   updated,
	}
}

func makeCramberryDocument() *cramgen.Document {
	updated := makeCramberryTimestamp()
	published := makeCramberryTimestamp()
	return &cramgen.Document{
		Id:       2001,
		Title:    "Important Document Title",
		Content:  "This is the document content with some meaningful text that would typically be much longer in a real application.",
		AuthorId: 1001,
		Status:   cramgen.StatusActive,
		Priority: cramgen.PriorityHigh,
		Tags: []cramgen.Tag{
			{Key: "category", Value: "technical"},
			{Key: "status", Value: "reviewed"},
			{Key: "version", Value: "2.0"},
		},
		Attachments: []cramgen.Attachment{
			{
				Id:         "att-001",
				Filename:   "report.pdf",
				MimeType:   "application/pdf",
				SizeBytes:  1048576,
				Checksum:   []byte{0xde, 0xad, 0xbe, 0xef},
				UploadedAt: *makeCramberryTimestamp(),
			},
		},
		Comments: []cramgen.Comment{
			{
				Id:        3001,
				AuthorId:  1002,
				Content:   "Great document!",
				CreatedAt: *makeCramberryTimestamp(),
				Reactions: []int64{1001, 1003, 1004},
			},
		},
		Metadata: map[string]string{
			"source":   "import",
			"encoding": "utf-8",
			"version":  "1.0",
		},
		Collaborators: []int64{1001, 1002, 1003},
		CreatedAt:     *makeCramberryTimestamp(),
		UpdatedAt:     updated,
		PublishedAt:   published,
	}
}

func makeCramberryEvent() *cramgen.Event {
	payload := []byte(`{"action":"click","element":"button"}`)
	corrId := "corr-123"
	causId := "caus-456"
	region := "us-west-2"
	return &cramgen.Event{
		Id:         "evt-001",
		Type:       cramgen.EventTypeCreated,
		EntityType: "document",
		EntityId:   "doc-2001",
		Source: cramgen.EventSource{
			Service:  "document-service",
			Instance: "prod-01",
			Version:  "1.2.3",
			Region:   &region,
		},
		Timestamp: *makeCramberryTimestamp(),
		Attributes: map[string]string{
			"user_id": "1001",
			"action":  "create",
		},
		Payload:       &payload,
		CorrelationId: &corrId,
		CausationId:   &causId,
	}
}

func makeCramberryBatchRequest(size int) *cramgen.BatchRequest {
	items := make([]cramgen.SmallMessage, size)
	for i := 0; i < size; i++ {
		items[i] = cramgen.SmallMessage{
			Id:     int64(i),
			Name:   "batch-item",
			Active: i%2 == 0,
		}
	}
	return &cramgen.BatchRequest{
		RequestId: "batch-001",
		Items:     items,
		Headers: map[string]string{
			"Content-Type": "application/x-cramberry",
			"X-Request-Id": "req-123",
		},
		SubmittedAt: *makeCramberryTimestamp(),
		Priority:    cramgen.PriorityMedium,
	}
}

// ============================================================================
// Test Data Construction - Protobuf Types
// ============================================================================

func makeProtobufSmallMessage() *pb.SmallMessage {
	return &pb.SmallMessage{
		Id:     12345,
		Name:   "test-item",
		Active: true,
	}
}

func makeProtobufPoint() *pb.Point {
	return &pb.Point{
		X: 123.456,
		Y: 789.012,
		Z: 345.678,
	}
}

func makeProtobufTimestamp() *pb.Timestamp {
	return &pb.Timestamp{
		Seconds: 1705900800,
		Nanos:   123456789,
	}
}

func makeProtobufMetrics() *pb.Metrics {
	return &pb.Metrics{
		Count:      1000000,
		Sum:        12345678.90,
		Min:        0.001,
		Max:        99999.99,
		Avg:        12345.67,
		P50:        10000.0,
		P95:        50000.0,
		P99:        90000.0,
		TotalBytes: 1073741824,
		ErrorCount: 42,
	}
}

func makeProtobufAddress() *pb.Address {
	street2 := "Suite 100"
	return &pb.Address{
		Street1:     "123 Main Street",
		Street2:     &street2,
		City:        "San Francisco",
		State:       "CA",
		PostalCode:  "94105",
		Country:     "USA",
		Coordinates: makeProtobufPoint(),
	}
}

func makeProtobufContactInfo() *pb.ContactInfo {
	phone := "+1-555-123-4567"
	mobile := "+1-555-987-6543"
	return &pb.ContactInfo{
		Email:          "john.doe@example.com",
		Phone:          &phone,
		Mobile:         &mobile,
		MailingAddress: makeProtobufAddress(),
	}
}

func makeProtobufPerson() *pb.Person {
	middle := "Robert"
	return &pb.Person{
		Id:          1001,
		FirstName:   "John",
		LastName:    "Doe",
		MiddleName:  &middle,
		DateOfBirth: makeProtobufTimestamp(),
		Contact:     makeProtobufContactInfo(),
		Status:      pb.Status_STATUS_ACTIVE,
		CreatedAt:   makeProtobufTimestamp(),
		UpdatedAt:   makeProtobufTimestamp(),
	}
}

func makeProtobufDocument() *pb.Document {
	return &pb.Document{
		Id:       2001,
		Title:    "Important Document Title",
		Content:  "This is the document content with some meaningful text that would typically be much longer in a real application.",
		AuthorId: 1001,
		Status:   pb.Status_STATUS_ACTIVE,
		Priority: pb.Priority_PRIORITY_HIGH,
		Tags: []*pb.Tag{
			{Key: "category", Value: "technical"},
			{Key: "status", Value: "reviewed"},
			{Key: "version", Value: "2.0"},
		},
		Attachments: []*pb.Attachment{
			{
				Id:         "att-001",
				Filename:   "report.pdf",
				MimeType:   "application/pdf",
				SizeBytes:  1048576,
				Checksum:   []byte{0xde, 0xad, 0xbe, 0xef},
				UploadedAt: makeProtobufTimestamp(),
			},
		},
		Comments: []*pb.Comment{
			{
				Id:        3001,
				AuthorId:  1002,
				Content:   "Great document!",
				CreatedAt: makeProtobufTimestamp(),
				Reactions: []int64{1001, 1003, 1004},
			},
		},
		Metadata: map[string]string{
			"source":   "import",
			"encoding": "utf-8",
			"version":  "1.0",
		},
		Collaborators: []int64{1001, 1002, 1003},
		CreatedAt:     makeProtobufTimestamp(),
		UpdatedAt:     makeProtobufTimestamp(),
		PublishedAt:   makeProtobufTimestamp(),
	}
}

func makeProtobufEvent() *pb.Event {
	payload := []byte(`{"action":"click","element":"button"}`)
	corrId := "corr-123"
	causId := "caus-456"
	region := "us-west-2"
	return &pb.Event{
		Id:         "evt-001",
		Type:       pb.EventType_EVENT_TYPE_CREATED,
		EntityType: "document",
		EntityId:   "doc-2001",
		Source: &pb.EventSource{
			Service:  "document-service",
			Instance: "prod-01",
			Version:  "1.2.3",
			Region:   &region,
		},
		Timestamp: makeProtobufTimestamp(),
		Attributes: map[string]string{
			"user_id": "1001",
			"action":  "create",
		},
		Payload:       payload,
		CorrelationId: &corrId,
		CausationId:   &causId,
	}
}

func makeProtobufBatchRequest(size int) *pb.BatchRequest {
	items := make([]*pb.SmallMessage, size)
	for i := 0; i < size; i++ {
		items[i] = &pb.SmallMessage{
			Id:     int64(i),
			Name:   "batch-item",
			Active: i%2 == 0,
		}
	}
	return &pb.BatchRequest{
		RequestId: "batch-001",
		Items:     items,
		Headers: map[string]string{
			"Content-Type": "application/x-cramberry",
			"X-Request-Id": "req-123",
		},
		SubmittedAt: makeProtobufTimestamp(),
		Priority:    pb.Priority_PRIORITY_MEDIUM,
	}
}

// ============================================================================
// JSON Types (mirrors the Cramberry types for fair comparison)
// ============================================================================

type JSONSmallMessage struct {
	Id     int64  `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type JSONPoint struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

type JSONTimestamp struct {
	Seconds int64 `json:"seconds"`
	Nanos   int32 `json:"nanos"`
}

type JSONMetrics struct {
	Count      int64   `json:"count"`
	Sum        float64 `json:"sum"`
	Min        float64 `json:"min"`
	Max        float64 `json:"max"`
	Avg        float64 `json:"avg"`
	P50        float64 `json:"p50"`
	P95        float64 `json:"p95"`
	P99        float64 `json:"p99"`
	TotalBytes int64   `json:"total_bytes"`
	ErrorCount int64   `json:"error_count"`
}

type JSONAddress struct {
	Street1     string     `json:"street1"`
	Street2     *string    `json:"street2,omitempty"`
	City        string     `json:"city"`
	State       string     `json:"state"`
	PostalCode  string     `json:"postal_code"`
	Country     string     `json:"country"`
	Coordinates *JSONPoint `json:"coordinates,omitempty"`
}

type JSONContactInfo struct {
	Email          string       `json:"email"`
	Phone          *string      `json:"phone,omitempty"`
	Mobile         *string      `json:"mobile,omitempty"`
	MailingAddress *JSONAddress `json:"mailing_address,omitempty"`
}

type JSONPerson struct {
	Id          int64            `json:"id"`
	FirstName   string           `json:"first_name"`
	LastName    string           `json:"last_name"`
	MiddleName  *string          `json:"middle_name,omitempty"`
	DateOfBirth *JSONTimestamp   `json:"date_of_birth,omitempty"`
	Contact     *JSONContactInfo `json:"contact"`
	Status      int32            `json:"status"`
	CreatedAt   JSONTimestamp    `json:"created_at"`
	UpdatedAt   *JSONTimestamp   `json:"updated_at,omitempty"`
}

type JSONTag struct {
	Key   string  `json:"key"`
	Value string  `json:"value"`
	Color *string `json:"color,omitempty"`
}

type JSONAttachment struct {
	Id         string        `json:"id"`
	Filename   string        `json:"filename"`
	MimeType   string        `json:"mime_type"`
	SizeBytes  int64         `json:"size_bytes"`
	Checksum   []byte        `json:"checksum"`
	URL        *string       `json:"url,omitempty"`
	UploadedAt JSONTimestamp `json:"uploaded_at"`
}

type JSONComment struct {
	Id        int64          `json:"id"`
	AuthorId  int64          `json:"author_id"`
	Content   string         `json:"content"`
	CreatedAt JSONTimestamp  `json:"created_at"`
	EditedAt  *JSONTimestamp `json:"edited_at,omitempty"`
	Reactions []int64        `json:"reactions"`
}

type JSONDocument struct {
	Id            int64             `json:"id"`
	Title         string            `json:"title"`
	Content       string            `json:"content"`
	AuthorId      int64             `json:"author_id"`
	Status        int32             `json:"status"`
	Priority      int32             `json:"priority"`
	Tags          []JSONTag         `json:"tags"`
	Attachments   []JSONAttachment  `json:"attachments"`
	Comments      []JSONComment     `json:"comments"`
	Metadata      map[string]string `json:"metadata"`
	Collaborators []int64           `json:"collaborators"`
	CreatedAt     JSONTimestamp     `json:"created_at"`
	UpdatedAt     *JSONTimestamp    `json:"updated_at,omitempty"`
	PublishedAt   *JSONTimestamp    `json:"published_at,omitempty"`
}

type JSONEventSource struct {
	Service  string  `json:"service"`
	Instance string  `json:"instance"`
	Version  string  `json:"version"`
	Region   *string `json:"region,omitempty"`
}

type JSONEvent struct {
	Id            string            `json:"id"`
	Type          int32             `json:"type"`
	EntityType    string            `json:"entity_type"`
	EntityId      string            `json:"entity_id"`
	Source        JSONEventSource   `json:"source"`
	Timestamp     JSONTimestamp     `json:"timestamp"`
	Attributes    map[string]string `json:"attributes"`
	Payload       []byte            `json:"payload,omitempty"`
	CorrelationId *string           `json:"correlation_id,omitempty"`
	CausationId   *string           `json:"causation_id,omitempty"`
}

type JSONBatchRequest struct {
	RequestId   string             `json:"request_id"`
	Items       []JSONSmallMessage `json:"items"`
	Headers     map[string]string  `json:"headers"`
	SubmittedAt JSONTimestamp      `json:"submitted_at"`
	Priority    int32              `json:"priority"`
}

func makeJSONSmallMessage() *JSONSmallMessage {
	return &JSONSmallMessage{
		Id:     12345,
		Name:   "test-item",
		Active: true,
	}
}

func makeJSONTimestamp() *JSONTimestamp {
	return &JSONTimestamp{
		Seconds: 1705900800,
		Nanos:   123456789,
	}
}

func makeJSONPoint() *JSONPoint {
	return &JSONPoint{
		X: 123.456,
		Y: 789.012,
		Z: 345.678,
	}
}

func makeJSONMetrics() *JSONMetrics {
	return &JSONMetrics{
		Count:      1000000,
		Sum:        12345678.90,
		Min:        0.001,
		Max:        99999.99,
		Avg:        12345.67,
		P50:        10000.0,
		P95:        50000.0,
		P99:        90000.0,
		TotalBytes: 1073741824,
		ErrorCount: 42,
	}
}

func makeJSONAddress() *JSONAddress {
	street2 := "Suite 100"
	return &JSONAddress{
		Street1:     "123 Main Street",
		Street2:     &street2,
		City:        "San Francisco",
		State:       "CA",
		PostalCode:  "94105",
		Country:     "USA",
		Coordinates: makeJSONPoint(),
	}
}

func makeJSONContactInfo() *JSONContactInfo {
	phone := "+1-555-123-4567"
	mobile := "+1-555-987-6543"
	return &JSONContactInfo{
		Email:          "john.doe@example.com",
		Phone:          &phone,
		Mobile:         &mobile,
		MailingAddress: makeJSONAddress(),
	}
}

func makeJSONPerson() *JSONPerson {
	middle := "Robert"
	return &JSONPerson{
		Id:          1001,
		FirstName:   "John",
		LastName:    "Doe",
		MiddleName:  &middle,
		DateOfBirth: makeJSONTimestamp(),
		Contact:     makeJSONContactInfo(),
		Status:      2, // ACTIVE
		CreatedAt:   *makeJSONTimestamp(),
		UpdatedAt:   makeJSONTimestamp(),
	}
}

func makeJSONDocument() *JSONDocument {
	return &JSONDocument{
		Id:       2001,
		Title:    "Important Document Title",
		Content:  "This is the document content with some meaningful text that would typically be much longer in a real application.",
		AuthorId: 1001,
		Status:   2,
		Priority: 2,
		Tags: []JSONTag{
			{Key: "category", Value: "technical"},
			{Key: "status", Value: "reviewed"},
			{Key: "version", Value: "2.0"},
		},
		Attachments: []JSONAttachment{
			{
				Id:         "att-001",
				Filename:   "report.pdf",
				MimeType:   "application/pdf",
				SizeBytes:  1048576,
				Checksum:   []byte{0xde, 0xad, 0xbe, 0xef},
				UploadedAt: *makeJSONTimestamp(),
			},
		},
		Comments: []JSONComment{
			{
				Id:        3001,
				AuthorId:  1002,
				Content:   "Great document!",
				CreatedAt: *makeJSONTimestamp(),
				Reactions: []int64{1001, 1003, 1004},
			},
		},
		Metadata: map[string]string{
			"source":   "import",
			"encoding": "utf-8",
			"version":  "1.0",
		},
		Collaborators: []int64{1001, 1002, 1003},
		CreatedAt:     *makeJSONTimestamp(),
		UpdatedAt:     makeJSONTimestamp(),
		PublishedAt:   makeJSONTimestamp(),
	}
}

func makeJSONEvent() *JSONEvent {
	payload := []byte(`{"action":"click","element":"button"}`)
	corrId := "corr-123"
	causId := "caus-456"
	region := "us-west-2"
	return &JSONEvent{
		Id:         "evt-001",
		Type:       0, // CREATED
		EntityType: "document",
		EntityId:   "doc-2001",
		Source: JSONEventSource{
			Service:  "document-service",
			Instance: "prod-01",
			Version:  "1.2.3",
			Region:   &region,
		},
		Timestamp: *makeJSONTimestamp(),
		Attributes: map[string]string{
			"user_id": "1001",
			"action":  "create",
		},
		Payload:       payload,
		CorrelationId: &corrId,
		CausationId:   &causId,
	}
}

func makeJSONBatchRequest(size int) *JSONBatchRequest {
	items := make([]JSONSmallMessage, size)
	for i := 0; i < size; i++ {
		items[i] = JSONSmallMessage{
			Id:     int64(i),
			Name:   "batch-item",
			Active: i%2 == 0,
		}
	}
	return &JSONBatchRequest{
		RequestId: "batch-001",
		Items:     items,
		Headers: map[string]string{
			"Content-Type": "application/x-cramberry",
			"X-Request-Id": "req-123",
		},
		SubmittedAt: *makeJSONTimestamp(),
		Priority:    1,
	}
}

// ============================================================================
// Benchmarks - Small Message (Baseline)
// ============================================================================

func BenchmarkSmallMessage_Cramberry_Encode(b *testing.B) {
	msg := makeCramberrySmallMessage()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkSmallMessage_Cramberry_Decode(b *testing.B) {
	msg := makeCramberrySmallMessage()
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.SmallMessage
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkSmallMessage_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufSmallMessage()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkSmallMessage_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufSmallMessage()
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.SmallMessage
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkSmallMessage_JSON_Encode(b *testing.B) {
	msg := makeJSONSmallMessage()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkSmallMessage_JSON_Decode(b *testing.B) {
	msg := makeJSONSmallMessage()
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONSmallMessage
		_ = json.Unmarshal(data, &result)
	}
}

// ============================================================================
// Benchmarks - Metrics (Scalar-heavy)
// ============================================================================

func BenchmarkMetrics_Cramberry_Encode(b *testing.B) {
	msg := makeCramberryMetrics()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkMetrics_Cramberry_Decode(b *testing.B) {
	msg := makeCramberryMetrics()
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.Metrics
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkMetrics_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufMetrics()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkMetrics_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufMetrics()
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.Metrics
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkMetrics_JSON_Encode(b *testing.B) {
	msg := makeJSONMetrics()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkMetrics_JSON_Decode(b *testing.B) {
	msg := makeJSONMetrics()
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONMetrics
		_ = json.Unmarshal(data, &result)
	}
}

// ============================================================================
// Benchmarks - Person (Nested Messages)
// ============================================================================

func BenchmarkPerson_Cramberry_Encode(b *testing.B) {
	msg := makeCramberryPerson()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkPerson_Cramberry_Decode(b *testing.B) {
	msg := makeCramberryPerson()
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.Person
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkPerson_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufPerson()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkPerson_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufPerson()
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.Person
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkPerson_JSON_Encode(b *testing.B) {
	msg := makeJSONPerson()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkPerson_JSON_Decode(b *testing.B) {
	msg := makeJSONPerson()
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONPerson
		_ = json.Unmarshal(data, &result)
	}
}

// ============================================================================
// Benchmarks - Document (Complex with Arrays/Maps)
// ============================================================================

func BenchmarkDocument_Cramberry_Encode(b *testing.B) {
	msg := makeCramberryDocument()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkDocument_Cramberry_Decode(b *testing.B) {
	msg := makeCramberryDocument()
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.Document
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkDocument_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufDocument()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkDocument_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufDocument()
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.Document
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkDocument_JSON_Encode(b *testing.B) {
	msg := makeJSONDocument()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkDocument_JSON_Decode(b *testing.B) {
	msg := makeJSONDocument()
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONDocument
		_ = json.Unmarshal(data, &result)
	}
}

// ============================================================================
// Benchmarks - Event (Maps and Optional Fields)
// ============================================================================

func BenchmarkEvent_Cramberry_Encode(b *testing.B) {
	msg := makeCramberryEvent()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkEvent_Cramberry_Decode(b *testing.B) {
	msg := makeCramberryEvent()
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.Event
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkEvent_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufEvent()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkEvent_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufEvent()
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.Event
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkEvent_JSON_Encode(b *testing.B) {
	msg := makeJSONEvent()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkEvent_JSON_Decode(b *testing.B) {
	msg := makeJSONEvent()
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONEvent
		_ = json.Unmarshal(data, &result)
	}
}

// ============================================================================
// Benchmarks - Batch Request (Large Arrays)
// ============================================================================

func BenchmarkBatch100_Cramberry_Encode(b *testing.B) {
	msg := makeCramberryBatchRequest(100)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkBatch100_Cramberry_Decode(b *testing.B) {
	msg := makeCramberryBatchRequest(100)
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.BatchRequest
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkBatch100_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufBatchRequest(100)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkBatch100_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufBatchRequest(100)
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.BatchRequest
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkBatch100_JSON_Encode(b *testing.B) {
	msg := makeJSONBatchRequest(100)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkBatch100_JSON_Decode(b *testing.B) {
	msg := makeJSONBatchRequest(100)
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONBatchRequest
		_ = json.Unmarshal(data, &result)
	}
}

func BenchmarkBatch1000_Cramberry_Encode(b *testing.B) {
	msg := makeCramberryBatchRequest(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = msg.MarshalCramberry()
	}
}

func BenchmarkBatch1000_Cramberry_Decode(b *testing.B) {
	msg := makeCramberryBatchRequest(1000)
	data, _ := msg.MarshalCramberry()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result cramgen.BatchRequest
		_ = result.UnmarshalCramberry(data)
	}
}

func BenchmarkBatch1000_Protobuf_Encode(b *testing.B) {
	msg := makeProtobufBatchRequest(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = proto.Marshal(msg)
	}
}

func BenchmarkBatch1000_Protobuf_Decode(b *testing.B) {
	msg := makeProtobufBatchRequest(1000)
	data, _ := proto.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result pb.BatchRequest
		_ = proto.Unmarshal(data, &result)
	}
}

func BenchmarkBatch1000_JSON_Encode(b *testing.B) {
	msg := makeJSONBatchRequest(1000)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(msg)
	}
}

func BenchmarkBatch1000_JSON_Decode(b *testing.B) {
	msg := makeJSONBatchRequest(1000)
	data, _ := json.Marshal(msg)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var result JSONBatchRequest
		_ = json.Unmarshal(data, &result)
	}
}

// ============================================================================
// Size Comparison Tests
// ============================================================================

func TestEncodedSizes(t *testing.T) {
	tests := []struct {
		name string
		cram func() ([]byte, error)
		pb   func() ([]byte, error)
		json func() ([]byte, error)
	}{
		{
			name: "SmallMessage",
			cram: func() ([]byte, error) { return makeCramberrySmallMessage().MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufSmallMessage()) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONSmallMessage()) },
		},
		{
			name: "Metrics",
			cram: func() ([]byte, error) { return makeCramberryMetrics().MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufMetrics()) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONMetrics()) },
		},
		{
			name: "Person",
			cram: func() ([]byte, error) { return makeCramberryPerson().MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufPerson()) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONPerson()) },
		},
		{
			name: "Document",
			cram: func() ([]byte, error) { return makeCramberryDocument().MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufDocument()) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONDocument()) },
		},
		{
			name: "Event",
			cram: func() ([]byte, error) { return makeCramberryEvent().MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufEvent()) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONEvent()) },
		},
		{
			name: "Batch100",
			cram: func() ([]byte, error) { return makeCramberryBatchRequest(100).MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufBatchRequest(100)) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONBatchRequest(100)) },
		},
		{
			name: "Batch1000",
			cram: func() ([]byte, error) { return makeCramberryBatchRequest(1000).MarshalCramberry() },
			pb:   func() ([]byte, error) { return proto.Marshal(makeProtobufBatchRequest(1000)) },
			json: func() ([]byte, error) { return json.Marshal(makeJSONBatchRequest(1000)) },
		},
	}

	t.Log("\n=== Encoded Size Comparison ===")
	t.Log("| Message       | Cramberry | Protobuf | JSON    | Cram/PB | JSON/PB |")
	t.Log("|---------------|-----------|----------|---------|---------|---------|")

	for _, tt := range tests {
		cramData, err := tt.cram()
		if err != nil {
			t.Errorf("%s: cramberry encode failed: %v", tt.name, err)
			continue
		}
		pbData, err := tt.pb()
		if err != nil {
			t.Errorf("%s: protobuf encode failed: %v", tt.name, err)
			continue
		}
		jsonData, err := tt.json()
		if err != nil {
			t.Errorf("%s: json encode failed: %v", tt.name, err)
			continue
		}

		cramPbRatio := float64(len(cramData)) / float64(len(pbData))
		jsonPbRatio := float64(len(jsonData)) / float64(len(pbData))

		t.Logf("| %-13s | %9d | %8d | %7d | %7.2fx | %7.2fx |",
			tt.name, len(cramData), len(pbData), len(jsonData), cramPbRatio, jsonPbRatio)
	}
}
