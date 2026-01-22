package cramberry

import (
	"bytes"
	"io"
	"testing"
)

func TestStreamWriterBasicTypes(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	// Write various types
	sw.WriteBool(true)
	sw.WriteBool(false)
	sw.WriteUint8(42)
	sw.WriteInt8(-42)
	sw.WriteUint16(1000)
	sw.WriteInt16(-1000)
	sw.WriteUint32(100000)
	sw.WriteInt32(-100000)
	sw.WriteUint64(1000000000)
	sw.WriteInt64(-1000000000)
	sw.WriteFloat32(3.14)
	sw.WriteFloat64(2.718281828)

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if sw.Err() != nil {
		t.Fatalf("write error: %v", sw.Err())
	}

	// Read back with StreamReader
	sr := NewStreamReader(&buf)

	if v := sr.ReadBool(); v != true {
		t.Errorf("expected true, got %v", v)
	}
	if v := sr.ReadBool(); v != false {
		t.Errorf("expected false, got %v", v)
	}
	if v := sr.ReadUint8(); v != 42 {
		t.Errorf("expected 42, got %v", v)
	}
	if v := sr.ReadInt8(); v != -42 {
		t.Errorf("expected -42, got %v", v)
	}
	if v := sr.ReadUint16(); v != 1000 {
		t.Errorf("expected 1000, got %v", v)
	}
	if v := sr.ReadInt16(); v != -1000 {
		t.Errorf("expected -1000, got %v", v)
	}
	if v := sr.ReadUint32(); v != 100000 {
		t.Errorf("expected 100000, got %v", v)
	}
	if v := sr.ReadInt32(); v != -100000 {
		t.Errorf("expected -100000, got %v", v)
	}
	if v := sr.ReadUint64(); v != 1000000000 {
		t.Errorf("expected 1000000000, got %v", v)
	}
	if v := sr.ReadInt64(); v != -1000000000 {
		t.Errorf("expected -1000000000, got %v", v)
	}
	if v := sr.ReadFloat32(); v != 3.14 {
		t.Errorf("expected 3.14, got %v", v)
	}
	if v := sr.ReadFloat64(); v != 2.718281828 {
		t.Errorf("expected 2.718281828, got %v", v)
	}

	if sr.Err() != nil {
		t.Fatalf("read error: %v", sr.Err())
	}
}

func TestStreamWriterStringAndBytes(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	sw.WriteString("hello, world!")
	sw.WriteBytes([]byte{1, 2, 3, 4, 5})

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	sr := NewStreamReader(&buf)

	if v := sr.ReadString(); v != "hello, world!" {
		t.Errorf("expected 'hello, world!', got %q", v)
	}
	if v := sr.ReadBytes(); !bytes.Equal(v, []byte{1, 2, 3, 4, 5}) {
		t.Errorf("expected [1,2,3,4,5], got %v", v)
	}

	if sr.Err() != nil {
		t.Fatalf("read error: %v", sr.Err())
	}
}

func TestStreamWriterVarint(t *testing.T) {
	tests := []struct {
		name  string
		value uint64
	}{
		{"zero", 0},
		{"one", 1},
		{"small", 127},
		{"medium", 16383},
		{"large", 1<<21 - 1},
		{"huge", 1<<63 - 1},
		{"max", ^uint64(0)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			sw := NewStreamWriter(&buf)
			sw.WriteUvarint(tt.value)
			if err := sw.Flush(); err != nil {
				t.Fatalf("flush error: %v", err)
			}

			sr := NewStreamReader(&buf)
			if v := sr.ReadUvarint(); v != tt.value {
				t.Errorf("expected %d, got %d", tt.value, v)
			}
			if sr.Err() != nil {
				t.Fatalf("read error: %v", sr.Err())
			}
		})
	}
}

func TestStreamWriterSvarint(t *testing.T) {
	tests := []struct {
		name  string
		value int64
	}{
		{"zero", 0},
		{"positive", 100},
		{"negative", -100},
		{"large_positive", 1 << 62},
		{"large_negative", -(1 << 62)},
		{"max", 1<<63 - 1},
		{"min", -1 << 63},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			sw := NewStreamWriter(&buf)
			sw.WriteSvarint(tt.value)
			if err := sw.Flush(); err != nil {
				t.Fatalf("flush error: %v", err)
			}

			sr := NewStreamReader(&buf)
			if v := sr.ReadSvarint(); v != tt.value {
				t.Errorf("expected %d, got %d", tt.value, v)
			}
			if sr.Err() != nil {
				t.Fatalf("read error: %v", sr.Err())
			}
		})
	}
}

func TestStreamWriterFixed(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	sw.WriteFixed32(0x12345678)
	sw.WriteFixed64(0x123456789ABCDEF0)

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	sr := NewStreamReader(&buf)

	if v := sr.ReadFixed32(); v != 0x12345678 {
		t.Errorf("expected 0x12345678, got %#x", v)
	}
	if v := sr.ReadFixed64(); v != 0x123456789ABCDEF0 {
		t.Errorf("expected 0x123456789ABCDEF0, got %#x", v)
	}

	if sr.Err() != nil {
		t.Fatalf("read error: %v", sr.Err())
	}
}

func TestStreamWriterComplex(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	sw.WriteComplex64(complex(1.5, 2.5))
	sw.WriteComplex128(complex(3.5, 4.5))

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	sr := NewStreamReader(&buf)

	if v := sr.ReadComplex64(); v != complex(1.5, 2.5) {
		t.Errorf("expected (1.5+2.5i), got %v", v)
	}
	if v := sr.ReadComplex128(); v != complex(3.5, 4.5) {
		t.Errorf("expected (3.5+4.5i), got %v", v)
	}

	if sr.Err() != nil {
		t.Fatalf("read error: %v", sr.Err())
	}
}

func TestStreamWriterTag(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	sw.WriteTag(1, WireVarint)
	sw.WriteTag(100, WireBytes)
	sw.WriteTag(1000, WireFixed64)

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	sr := NewStreamReader(&buf)

	fn, wt := sr.ReadTag()
	if fn != 1 || wt != WireVarint {
		t.Errorf("expected (1, WireVarint), got (%d, %d)", fn, wt)
	}

	fn, wt = sr.ReadTag()
	if fn != 100 || wt != WireBytes {
		t.Errorf("expected (100, WireBytes), got (%d, %d)", fn, wt)
	}

	fn, wt = sr.ReadTag()
	if fn != 1000 || wt != WireFixed64 {
		t.Errorf("expected (1000, WireFixed64), got (%d, %d)", fn, wt)
	}

	if sr.Err() != nil {
		t.Fatalf("read error: %v", sr.Err())
	}
}

func TestStreamWriterHeaders(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	sw.WriteArrayHeader(10)
	sw.WriteMapHeader(5)
	sw.WriteTypeID(TypeID(128))

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	sr := NewStreamReader(&buf)

	if v := sr.ReadArrayHeader(); v != 10 {
		t.Errorf("expected 10, got %d", v)
	}
	if v := sr.ReadMapHeader(); v != 5 {
		t.Errorf("expected 5, got %d", v)
	}
	if v := sr.ReadTypeID(); v != TypeID(128) {
		t.Errorf("expected 128, got %d", v)
	}

	if sr.Err() != nil {
		t.Fatalf("read error: %v", sr.Err())
	}
}

func TestStreamDelimitedMessages(t *testing.T) {
	type Message struct {
		ID   int32  `cramberry:"1"`
		Name string `cramberry:"2"`
	}

	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	// Write multiple delimited messages
	messages := []Message{
		{ID: 1, Name: "first"},
		{ID: 2, Name: "second"},
		{ID: 3, Name: "third"},
	}

	for _, msg := range messages {
		if err := sw.WriteDelimited(&msg); err != nil {
			t.Fatalf("write delimited error: %v", err)
		}
	}

	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	// Read them back
	sr := NewStreamReader(&buf)
	for i, expected := range messages {
		var msg Message
		if err := sr.ReadDelimited(&msg); err != nil {
			t.Fatalf("read delimited %d error: %v", i, err)
		}
		if msg.ID != expected.ID || msg.Name != expected.Name {
			t.Errorf("message %d: expected %+v, got %+v", i, expected, msg)
		}
	}
}

func TestMessageIterator(t *testing.T) {
	type Message struct {
		ID int32 `cramberry:"1"`
	}

	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	// Write multiple messages
	for i := int32(1); i <= 5; i++ {
		if err := sw.WriteDelimited(&Message{ID: i}); err != nil {
			t.Fatalf("write delimited error: %v", err)
		}
	}
	if err := sw.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	// Read with iterator
	it := NewMessageIterator(&buf)
	count := 0
	var msg Message
	for it.Next(&msg) {
		count++
		if msg.ID != int32(count) {
			t.Errorf("expected ID %d, got %d", count, msg.ID)
		}
	}

	if it.Err() != nil {
		t.Fatalf("iterator error: %v", it.Err())
	}
	if count != 5 {
		t.Errorf("expected 5 messages, got %d", count)
	}
}

func TestStreamWriterClose(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	sw.WriteUint32(42)
	if err := sw.Close(); err != nil {
		t.Fatalf("close error: %v", err)
	}

	// Writing after close should fail
	sw.WriteUint32(100)
	if sw.Err() == nil {
		t.Error("expected error writing after close")
	}
}

func TestStreamReaderEOF(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteUint32(42)
	sw.Flush()

	sr := NewStreamReader(&buf)
	sr.ReadUint32() // Read the value

	// Reading more should fail with EOF
	sr.ReadUint32()
	if sr.Err() == nil {
		t.Error("expected EOF error")
	}
}

func TestStreamWriterLimits(t *testing.T) {
	opts := DefaultOptions
	opts.Limits.MaxStringLength = 10

	var buf bytes.Buffer
	sw := NewStreamWriterWithOptions(&buf, opts)

	sw.WriteString("short")
	if sw.Err() != nil {
		t.Fatalf("unexpected error for short string: %v", sw.Err())
	}

	sw.WriteString("this is too long")
	if sw.Err() == nil {
		t.Error("expected error for long string")
	}
}

func TestStreamReaderLimits(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteString("this is a long string")
	sw.Flush()

	opts := DefaultOptions
	opts.Limits.MaxStringLength = 10

	sr := NewStreamReaderWithOptions(&buf, opts)
	sr.ReadString()
	if sr.Err() == nil {
		t.Error("expected error for long string")
	}
}

func TestStreamWriterPool(t *testing.T) {
	var buf bytes.Buffer

	// Get from pool
	sw := GetStreamWriter(&buf)
	sw.WriteUint32(42)
	sw.Flush()

	// Return to pool
	PutStreamWriter(sw)

	// Verify data was written
	sr := NewStreamReader(&buf)
	if v := sr.ReadUint32(); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}
}

func TestStreamReaderPool(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteUint32(42)
	sw.Flush()

	// Get from pool
	sr := GetStreamReader(&buf)
	if v := sr.ReadUint32(); v != 42 {
		t.Errorf("expected 42, got %d", v)
	}

	// Return to pool
	PutStreamReader(sr)
}

func TestStreamWriterReset(t *testing.T) {
	var buf1, buf2 bytes.Buffer
	sw := NewStreamWriter(&buf1)

	sw.WriteUint32(1)
	sw.Flush()

	sw.Reset(&buf2)
	sw.WriteUint32(2)
	sw.Flush()

	sr1 := NewStreamReader(&buf1)
	sr2 := NewStreamReader(&buf2)

	if v := sr1.ReadUint32(); v != 1 {
		t.Errorf("expected 1, got %d", v)
	}
	if v := sr2.ReadUint32(); v != 2 {
		t.Errorf("expected 2, got %d", v)
	}
}

func TestStreamReaderSkipMessage(t *testing.T) {
	type Message struct {
		ID int32 `cramberry:"1"`
	}

	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteDelimited(&Message{ID: 1})
	sw.WriteDelimited(&Message{ID: 2})
	sw.WriteDelimited(&Message{ID: 3})
	sw.Flush()

	sr := NewStreamReader(&buf)

	// Skip first message
	sr.SkipMessage()
	if sr.Err() != nil {
		t.Fatalf("skip error: %v", sr.Err())
	}

	// Read second message
	var msg Message
	sr.ReadDelimited(&msg)
	if msg.ID != 2 {
		t.Errorf("expected ID 2, got %d", msg.ID)
	}

	// Skip third message
	sr.SkipMessage()
	if sr.Err() != nil {
		t.Fatalf("skip error: %v", sr.Err())
	}
}

func TestStreamRawBytes(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	sw.WriteRawBytes(data)
	sw.Flush()

	sr := NewStreamReader(&buf)
	result := sr.ReadRawBytes(8)
	if !bytes.Equal(result, data) {
		t.Errorf("expected %v, got %v", data, result)
	}
}

func TestStreamWriterBufferSize(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriterSize(&buf, 16) // Small buffer

	// Write more than buffer size
	for i := 0; i < 100; i++ {
		sw.WriteUint32(uint32(i))
	}
	sw.Flush()

	sr := NewStreamReader(&buf)
	for i := 0; i < 100; i++ {
		if v := sr.ReadUint32(); v != uint32(i) {
			t.Errorf("at %d: expected %d, got %d", i, i, v)
		}
	}
}

func TestStreamReaderPeek(t *testing.T) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteBytes([]byte{1, 2, 3, 4, 5})
	sw.Flush()

	sr := NewStreamReader(&buf)

	// Peek should not advance position
	peeked, err := sr.Peek(2)
	if err != nil {
		t.Fatalf("peek error: %v", err)
	}
	if len(peeked) < 2 {
		t.Fatalf("expected at least 2 bytes, got %d", len(peeked))
	}

	// Read should still work
	data := sr.ReadBytes()
	if !bytes.Equal(data, []byte{1, 2, 3, 4, 5}) {
		t.Errorf("expected [1,2,3,4,5], got %v", data)
	}
}

func BenchmarkStreamWriter(b *testing.B) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		sw.Reset(&buf)
		sw.WriteUint64(uint64(i))
		sw.WriteString("benchmark test string")
		sw.WriteBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8})
		sw.Flush()
	}
}

func BenchmarkStreamReader(b *testing.B) {
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteUint64(12345678)
	sw.WriteString("benchmark test string")
	sw.WriteBytes([]byte{1, 2, 3, 4, 5, 6, 7, 8})
	sw.Flush()
	data := buf.Bytes()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sr := NewStreamReader(bytes.NewReader(data))
		sr.ReadUint64()
		sr.ReadString()
		sr.ReadBytes()
	}
}

func BenchmarkStreamDelimited(b *testing.B) {
	type Message struct {
		ID   int32  `cramberry:"1"`
		Name string `cramberry:"2"`
	}

	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	msg := Message{ID: 42, Name: "test message"}

	b.Run("Write", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buf.Reset()
			sw.Reset(&buf)
			sw.WriteDelimited(&msg)
			sw.Flush()
		}
	})

	// Prepare data for read benchmark
	buf.Reset()
	sw.Reset(&buf)
	sw.WriteDelimited(&msg)
	sw.Flush()
	data := buf.Bytes()

	b.Run("Read", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			sr := NewStreamReader(bytes.NewReader(data))
			var m Message
			sr.ReadDelimited(&m)
		}
	})
}

type slowReader struct {
	data   []byte
	offset int
}

func (r *slowReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	// Return only 1 byte at a time
	p[0] = r.data[r.offset]
	r.offset++
	return 1, nil
}

func TestStreamReaderSlowSource(t *testing.T) {
	// Create data
	var buf bytes.Buffer
	sw := NewStreamWriter(&buf)
	sw.WriteString("hello world")
	sw.WriteUint64(12345678901234567890)
	sw.Flush()

	// Read from slow reader (1 byte at a time)
	slow := &slowReader{data: buf.Bytes()}
	sr := NewStreamReader(slow)

	if v := sr.ReadString(); v != "hello world" {
		t.Errorf("expected 'hello world', got %q", v)
	}
	if v := sr.ReadUint64(); v != 12345678901234567890 {
		t.Errorf("expected 12345678901234567890, got %d", v)
	}
	if sr.Err() != nil {
		t.Fatalf("unexpected error: %v", sr.Err())
	}
}
