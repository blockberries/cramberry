// Example: Streaming Cramberry serialization
//
// This example demonstrates streaming multiple messages over a connection.
package main

import (
	"bytes"
	"fmt"
	"log"

	"github.com/cramberry/cramberry-go/pkg/cramberry"
)

// LogEntry represents a log message.
type LogEntry struct {
	Timestamp int64  `cramberry:"1"`
	Level     string `cramberry:"2"`
	Message   string `cramberry:"3"`
	Source    string `cramberry:"4"`
}

func main() {
	// Create sample log entries
	entries := []LogEntry{
		{Timestamp: 1700000001, Level: "INFO", Message: "Application started", Source: "main"},
		{Timestamp: 1700000002, Level: "DEBUG", Message: "Loading configuration", Source: "config"},
		{Timestamp: 1700000003, Level: "INFO", Message: "Database connected", Source: "db"},
		{Timestamp: 1700000004, Level: "WARN", Message: "High memory usage", Source: "monitor"},
		{Timestamp: 1700000005, Level: "ERROR", Message: "Request timeout", Source: "http"},
	}

	// Simulate a network connection with a buffer
	var buf bytes.Buffer

	// Create stream writer
	sw := cramberry.NewStreamWriter(&buf)

	// Write all entries as delimited messages
	fmt.Println("Writing log entries...")
	for i, entry := range entries {
		if err := sw.WriteDelimited(&entry); err != nil {
			log.Fatalf("Failed to write entry %d: %v", i, err)
		}
	}

	// Flush to ensure all data is written
	if err := sw.Flush(); err != nil {
		log.Fatalf("Failed to flush: %v", err)
	}

	fmt.Printf("Wrote %d entries in %d bytes\n", len(entries), buf.Len())
	fmt.Printf("Average bytes per entry: %.1f\n", float64(buf.Len())/float64(len(entries)))

	// Read entries back using iterator
	fmt.Println("\nReading log entries...")
	it := cramberry.NewMessageIterator(&buf)

	var entry LogEntry
	count := 0
	for it.Next(&entry) {
		fmt.Printf("[%d] %s | %-5s | %s: %s\n",
			entry.Timestamp,
			entry.Source,
			entry.Level,
			entry.Source,
			entry.Message)
		count++
	}

	if err := it.Err(); err != nil {
		log.Fatalf("Iterator error: %v", err)
	}

	fmt.Printf("\nRead %d entries\n", count)

	// Demonstrate stream with custom buffer size
	fmt.Println("\n--- Using Custom Buffer Size ---")

	var buf2 bytes.Buffer
	sw2 := cramberry.NewStreamWriterSize(&buf2, 4096) // 4KB buffer

	for _, entry := range entries {
		if err := sw2.WriteDelimited(&entry); err != nil {
			log.Fatalf("Failed to write: %v", err)
		}
	}
	sw2.Flush()

	fmt.Printf("Custom buffer writer: %d bytes\n", buf2.Len())
}
