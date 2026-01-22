// Example: Basic Cramberry serialization
//
// This example demonstrates basic struct serialization and deserialization.
package main

import (
	"fmt"
	"log"

	"github.com/cramberry/cramberry-go/pkg/cramberry"
)

// User represents a simple user record.
type User struct {
	ID    int64  `cramberry:"1,required"`
	Name  string `cramberry:"2"`
	Email string `cramberry:"3"`
	Age   int32  `cramberry:"4,omitempty"`
}

// Address represents a physical address.
type Address struct {
	Street  string `cramberry:"1"`
	City    string `cramberry:"2"`
	Country string `cramberry:"3"`
	ZipCode string `cramberry:"4"`
}

// Profile combines user info with address.
type Profile struct {
	User    User     `cramberry:"1"`
	Address *Address `cramberry:"2"`
	Tags    []string `cramberry:"3"`
}

func main() {
	// Create a profile
	profile := Profile{
		User: User{
			ID:    12345,
			Name:  "Alice Smith",
			Email: "alice@example.com",
			Age:   30,
		},
		Address: &Address{
			Street:  "123 Main St",
			City:    "San Francisco",
			Country: "USA",
			ZipCode: "94102",
		},
		Tags: []string{"developer", "golang", "cramberry"},
	}

	// Marshal to binary
	data, err := cramberry.Marshal(profile)
	if err != nil {
		log.Fatalf("Marshal failed: %v", err)
	}

	fmt.Printf("Serialized size: %d bytes\n", len(data))

	// Compare with approximate JSON size
	// JSON would be ~200 bytes for this data
	fmt.Printf("Approximate compression vs JSON: %.0f%%\n",
		(1.0-float64(len(data))/200.0)*100)

	// Unmarshal back
	var decoded Profile
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		log.Fatalf("Unmarshal failed: %v", err)
	}

	// Print decoded data
	fmt.Println("\nDecoded Profile:")
	fmt.Printf("  User ID: %d\n", decoded.User.ID)
	fmt.Printf("  Name: %s\n", decoded.User.Name)
	fmt.Printf("  Email: %s\n", decoded.User.Email)
	fmt.Printf("  Age: %d\n", decoded.User.Age)
	if decoded.Address != nil {
		fmt.Printf("  Address: %s, %s, %s %s\n",
			decoded.Address.Street,
			decoded.Address.City,
			decoded.Address.Country,
			decoded.Address.ZipCode)
	}
	fmt.Printf("  Tags: %v\n", decoded.Tags)

	// Demonstrate Size function (no allocation)
	size := cramberry.Size(profile)
	fmt.Printf("\nPre-computed size: %d bytes\n", size)

	// Demonstrate MarshalAppend (reuse buffer)
	buf := make([]byte, 0, size)
	buf, err = cramberry.MarshalAppend(buf, profile)
	if err != nil {
		log.Fatalf("MarshalAppend failed: %v", err)
	}
	fmt.Printf("MarshalAppend result: %d bytes\n", len(buf))
}
