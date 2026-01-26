//go:build go1.18

package schema

import (
	"testing"
)

// FuzzSchemaParser tests that the schema parser never panics on arbitrary input.
func FuzzSchemaParser(f *testing.F) {
	// Seed corpus with valid schema snippets
	f.Add(`message Foo { bar: int32 = 1; }`)
	f.Add(`message Empty {}`)
	f.Add(`enum Status { UNKNOWN = 0; ACTIVE = 1; }`)
	f.Add(`interface Principal { User = 128; }`)
	f.Add(`package example;`)
	f.Add(`
package example;

message User {
    id: int64 = 1 [required];
    name: string = 2;
    tags: []string = 3;
    metadata: map[string]string = 4;
}
`)
	f.Add(`
interface Animal {
    Dog = 128;
    Cat = 129;
}
`)

	// Add edge cases
	f.Add(``)
	f.Add(`{`)
	f.Add(`}`)
	f.Add(`message`)
	f.Add(`message {`)
	f.Add(`message Foo`)
	f.Add(`message Foo {`)
	f.Add(`message Foo { bar }`)
	f.Add(`message Foo { bar: }`)
	f.Add(`message Foo { bar: int32 }`)
	f.Add(`message Foo { bar: int32 = }`)
	f.Add(`message Foo { bar: int32 = abc; }`)

	f.Fuzz(func(t *testing.T, input string) {
		// Parser should never panic on any input
		p := NewParser("fuzz.cram", input)
		_, _ = p.Parse()
	})
}

// FuzzLexer tests that the lexer never panics on arbitrary input.
func FuzzLexer(f *testing.F) {
	f.Add(`message Foo { bar: int32 = 1; }`)
	f.Add(`"hello world"`)
	f.Add(`123`)
	f.Add(`0x1234`)
	f.Add(`identifier`)
	f.Add(`// comment`)
	f.Add(`/* multi-line comment */`)

	f.Fuzz(func(t *testing.T, input string) {
		l := NewLexer("fuzz.cram", input)
		// Consume all tokens - should never panic
		for {
			tok := l.Next()
			if tok.Type == TokenEOF || tok.Type == TokenError {
				break
			}
		}
	})
}
