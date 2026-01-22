// Example: Polymorphic types with Cramberry
//
// This example demonstrates interface serialization using the type registry.
package main

import (
	"fmt"
	"log"

	"github.com/cramberry/cramberry-go/pkg/cramberry"
)

// Shape is an interface for different shapes.
type Shape interface {
	Area() float64
	Name() string
}

// Circle represents a circle.
type Circle struct {
	Radius float64 `cramberry:"1"`
}

func (c *Circle) Area() float64 { return 3.14159 * c.Radius * c.Radius }
func (c *Circle) Name() string  { return "Circle" }

// Rectangle represents a rectangle.
type Rectangle struct {
	Width  float64 `cramberry:"1"`
	Height float64 `cramberry:"2"`
}

func (r *Rectangle) Area() float64 { return r.Width * r.Height }
func (r *Rectangle) Name() string  { return "Rectangle" }

// Triangle represents a triangle.
type Triangle struct {
	Base   float64 `cramberry:"1"`
	Height float64 `cramberry:"2"`
}

func (t *Triangle) Area() float64 { return 0.5 * t.Base * t.Height }
func (t *Triangle) Name() string  { return "Triangle" }

// Drawing contains multiple shapes.
type Drawing struct {
	Title  string  `cramberry:"1"`
	Shapes []Shape `cramberry:"2"`
}

func init() {
	// Register all shape types for polymorphic serialization
	cramberry.MustRegister[Circle]()
	cramberry.MustRegister[Rectangle]()
	cramberry.MustRegister[Triangle]()
}

func main() {
	// Create a drawing with multiple shapes
	drawing := Drawing{
		Title: "My Shapes",
		Shapes: []Shape{
			&Circle{Radius: 5.0},
			&Rectangle{Width: 10.0, Height: 20.0},
			&Triangle{Base: 8.0, Height: 6.0},
			&Circle{Radius: 3.0},
		},
	}

	// Calculate total area before serialization
	totalArea := 0.0
	for _, shape := range drawing.Shapes {
		totalArea += shape.Area()
	}
	fmt.Printf("Original drawing: %q with %d shapes\n", drawing.Title, len(drawing.Shapes))
	fmt.Printf("Total area: %.2f\n", totalArea)

	// Marshal the drawing (with polymorphic shapes)
	data, err := cramberry.Marshal(drawing)
	if err != nil {
		log.Fatalf("Marshal failed: %v", err)
	}

	fmt.Printf("\nSerialized size: %d bytes\n", len(data))

	// Unmarshal back
	var decoded Drawing
	if err := cramberry.Unmarshal(data, &decoded); err != nil {
		log.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify the decoded data
	fmt.Printf("\nDecoded drawing: %q\n", decoded.Title)
	fmt.Println("Shapes:")

	decodedArea := 0.0
	for i, shape := range decoded.Shapes {
		fmt.Printf("  %d. %s - Area: %.2f\n", i+1, shape.Name(), shape.Area())
		decodedArea += shape.Area()
	}

	fmt.Printf("\nTotal area after decode: %.2f\n", decodedArea)

	if decodedArea == totalArea {
		fmt.Println("Areas match!")
	}

	// Show type information
	fmt.Println("\nType details:")
	for _, shape := range decoded.Shapes {
		switch s := shape.(type) {
		case *Circle:
			fmt.Printf("  Circle with radius %.2f\n", s.Radius)
		case *Rectangle:
			fmt.Printf("  Rectangle %.2f x %.2f\n", s.Width, s.Height)
		case *Triangle:
			fmt.Printf("  Triangle base %.2f, height %.2f\n", s.Base, s.Height)
		}
	}
}
