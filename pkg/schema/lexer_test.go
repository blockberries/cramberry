package schema

import (
	"testing"
)

func TestLexerKeywords(t *testing.T) {
	input := "package import as message enum interface option required repeated optional map true false deprecated"

	expected := []struct {
		typ   TokenType
		value string
	}{
		{TokenPackage, "package"},
		{TokenImport, "import"},
		{TokenAs, "as"},
		{TokenMessage, "message"},
		{TokenEnum, "enum"},
		{TokenInterface, "interface"},
		{TokenOption, "option"},
		{TokenRequired, "required"},
		{TokenRepeated, "repeated"},
		{TokenOptional, "optional"},
		{TokenMap, "map"},
		{TokenTrue, "true"},
		{TokenFalse, "false"},
		{TokenDeprecated, "deprecated"},
		{TokenEOF, ""},
	}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != exp.typ {
			t.Errorf("token %d: expected type %v, got %v", i, exp.typ, tok.Type)
		}
		if tok.Value != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexerIdentifiers(t *testing.T) {
	input := "foo Bar _private camelCase snake_case PascalCase"

	expected := []string{"foo", "Bar", "_private", "camelCase", "snake_case", "PascalCase"}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != TokenIdent {
			t.Errorf("token %d: expected Ident, got %v", i, tok.Type)
		}
		if tok.Value != exp {
			t.Errorf("token %d: expected %q, got %q", i, exp, tok.Value)
		}
	}
}

func TestLexerNumbers(t *testing.T) {
	tests := []struct {
		input string
		typ   TokenType
		value string
	}{
		{"0", TokenInt, "0"},
		{"123", TokenInt, "123"},
		{"999999", TokenInt, "999999"},
		{"-1", TokenInt, "-1"},
		{"-123", TokenInt, "-123"},
		{"3.14", TokenFloat, "3.14"},
		{"0.5", TokenFloat, "0.5"},
		{"-3.14", TokenFloat, "-3.14"},
		{"1e10", TokenFloat, "1e10"},
		{"1E10", TokenFloat, "1E10"},
		{"1.5e10", TokenFloat, "1.5e10"},
		{"1e-10", TokenFloat, "1e-10"},
		{"1e+10", TokenFloat, "1e+10"},
	}

	for _, tt := range tests {
		lexer := NewLexer("test.cramberry", tt.input)
		tok := lexer.Next()
		if tok.Type != tt.typ {
			t.Errorf("input %q: expected type %v, got %v", tt.input, tt.typ, tok.Type)
		}
		if tok.Value != tt.value {
			t.Errorf("input %q: expected value %q, got %q", tt.input, tt.value, tok.Value)
		}
	}
}

func TestLexerStrings(t *testing.T) {
	tests := []struct {
		input string
		value string
	}{
		{`"hello"`, "hello"},
		{`"hello world"`, "hello world"},
		{`""`, ""},
		{`"with\nnewline"`, "with\nnewline"},
		{`"with\ttab"`, "with\ttab"},
		{`"with\\backslash"`, "with\\backslash"},
		{`"with\"quote"`, "with\"quote"},
		{`"with\rcarriage"`, "with\rcarriage"},
		{`"with\0null"`, "with\x00null"},
	}

	for _, tt := range tests {
		lexer := NewLexer("test.cramberry", tt.input)
		tok := lexer.Next()
		if tok.Type != TokenString {
			t.Errorf("input %q: expected String, got %v", tt.input, tok.Type)
		}
		if tok.Value != tt.value {
			t.Errorf("input %q: expected value %q, got %q", tt.input, tt.value, tok.Value)
		}
	}
}

func TestLexerStringErrors(t *testing.T) {
	tests := []struct {
		input string
		err   string
	}{
		{`"unterminated`, "unterminated string"},
		{`"with\x escape"`, "unknown escape sequence"},
		{"\"with\nnewline\"", "newline in string literal"},
	}

	for _, tt := range tests {
		lexer := NewLexer("test.cramberry", tt.input)
		tok := lexer.Next()
		if tok.Type != TokenError {
			t.Errorf("input %q: expected Error, got %v", tt.input, tok.Type)
		}
	}
}

func TestLexerPunctuation(t *testing.T) {
	input := "{}[]();:,=.*@"

	expected := []TokenType{
		TokenLBrace, TokenRBrace,
		TokenLBracket, TokenRBracket,
		TokenLParen, TokenRParen,
		TokenSemicolon, TokenColon, TokenComma,
		TokenEquals, TokenDot, TokenStar, TokenAt,
		TokenEOF,
	}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != exp {
			t.Errorf("token %d: expected %v, got %v", i, exp, tok.Type)
		}
	}
}

func TestLexerComments(t *testing.T) {
	tests := []struct {
		input   string
		typ     TokenType
		value   string
	}{
		{"// comment", TokenComment, "comment"},
		{"// comment with spaces", TokenComment, "comment with spaces"},
		{"//no space", TokenComment, "no space"},
		{"/// doc comment", TokenDocComment, "doc comment"},
		{"///doc comment", TokenDocComment, "doc comment"},
		{"// trailing   ", TokenComment, "trailing"},
	}

	for _, tt := range tests {
		lexer := NewLexer("test.cramberry", tt.input)
		tok := lexer.Next()
		if tok.Type != tt.typ {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.typ, tok.Type)
		}
		if tok.Value != tt.value {
			t.Errorf("input %q: expected value %q, got %q", tt.input, tt.value, tok.Value)
		}
	}
}

func TestLexerPositions(t *testing.T) {
	input := "package foo\nmessage Bar {\n  int32 x = 1;\n}"

	lexer := NewLexer("test.cramberry", input)

	tests := []struct {
		typ    TokenType
		line   int
		column int
	}{
		{TokenPackage, 1, 1},
		{TokenIdent, 1, 9},
		{TokenMessage, 2, 1},
		{TokenIdent, 2, 9},
		{TokenLBrace, 2, 13},
		{TokenIdent, 3, 3},     // int32
		{TokenIdent, 3, 9},     // x
		{TokenEquals, 3, 11},
		{TokenInt, 3, 13},
		{TokenSemicolon, 3, 14},
		{TokenRBrace, 4, 1},
	}

	for i, tt := range tests {
		tok := lexer.Next()
		if tok.Type != tt.typ {
			t.Errorf("token %d: expected type %v, got %v", i, tt.typ, tok.Type)
		}
		if tok.Position.Line != tt.line {
			t.Errorf("token %d: expected line %d, got %d", i, tt.line, tok.Position.Line)
		}
		if tok.Position.Column != tt.column {
			t.Errorf("token %d: expected column %d, got %d", i, tt.column, tok.Position.Column)
		}
	}
}

func TestLexerPeek(t *testing.T) {
	input := "foo bar baz"
	lexer := NewLexer("test.cramberry", input)

	// Peek should not advance
	tok1 := lexer.Peek()
	tok2 := lexer.Peek()
	if tok1.Value != tok2.Value {
		t.Errorf("Peek returned different values: %q vs %q", tok1.Value, tok2.Value)
	}
	if tok1.Value != "foo" {
		t.Errorf("expected 'foo', got %q", tok1.Value)
	}

	// Next should advance
	tok3 := lexer.Next()
	if tok3.Value != "foo" {
		t.Errorf("expected 'foo', got %q", tok3.Value)
	}

	tok4 := lexer.Peek()
	if tok4.Value != "bar" {
		t.Errorf("expected 'bar', got %q", tok4.Value)
	}
}

func TestLexerCompleteSchema(t *testing.T) {
	input := `
// Package declaration
package example;

import "other.cramberry";

/// User represents a user in the system.
message User {
  required int32 id = 1;
  optional string name = 2;
  repeated string tags = 3;
  map[string]int32 scores = 4;
  *Address address = 5;
}

enum Status {
  UNKNOWN = 0;
  ACTIVE = 1;
  INACTIVE = 2;
}

interface Animal {
  1 = Dog;
  2 = Cat;
}
`

	tokens := Tokenize("test.cramberry", input)

	// Verify we got some tokens without errors
	for _, tok := range tokens {
		if tok.Type == TokenError {
			t.Errorf("unexpected error: %s at line %d", tok.Value, tok.Position.Line)
		}
	}

	// Check we have reasonable number of tokens
	if len(tokens) < 50 {
		t.Errorf("expected at least 50 tokens, got %d", len(tokens))
	}

	// Verify last token is EOF
	if tokens[len(tokens)-1].Type != TokenEOF {
		t.Errorf("expected last token to be EOF, got %v", tokens[len(tokens)-1].Type)
	}
}

func TestLexerWhitespaceHandling(t *testing.T) {
	input := "  \t\n\n   foo   \n\t  bar  "

	lexer := NewLexer("test.cramberry", input)

	tok1 := lexer.Next()
	if tok1.Value != "foo" {
		t.Errorf("expected 'foo', got %q", tok1.Value)
	}

	tok2 := lexer.Next()
	if tok2.Value != "bar" {
		t.Errorf("expected 'bar', got %q", tok2.Value)
	}

	tok3 := lexer.Next()
	if tok3.Type != TokenEOF {
		t.Errorf("expected EOF, got %v", tok3.Type)
	}
}

func TestLexerUnexpectedCharacter(t *testing.T) {
	input := "foo $ bar"

	lexer := NewLexer("test.cramberry", input)

	tok1 := lexer.Next() // foo
	if tok1.Value != "foo" {
		t.Errorf("expected 'foo', got %q", tok1.Value)
	}

	tok2 := lexer.Next() // $
	if tok2.Type != TokenError {
		t.Errorf("expected Error for '$', got %v", tok2.Type)
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		typ TokenType
		str string
	}{
		{TokenError, "Error"},
		{TokenEOF, "EOF"},
		{TokenIdent, "Ident"},
		{TokenInt, "Int"},
		{TokenFloat, "Float"},
		{TokenString, "String"},
		{TokenPackage, "package"},
		{TokenMessage, "message"},
		{TokenLBrace, "{"},
		{TokenRBrace, "}"},
		{TokenDocComment, "DocComment"},
	}

	for _, tt := range tests {
		if tt.typ.String() != tt.str {
			t.Errorf("TokenType(%d).String() = %q, want %q", tt.typ, tt.typ.String(), tt.str)
		}
	}
}

func TestTokenString(t *testing.T) {
	tok := Token{Type: TokenIdent, Value: "foo"}
	s := tok.String()
	if s != `Ident("foo")` {
		t.Errorf("Token.String() = %q, want %q", s, `Ident("foo")`)
	}

	tok2 := Token{Type: TokenLBrace, Value: "{"}
	s2 := tok2.String()
	if s2 != `{("{")` {
		t.Errorf("Token.String() = %q, want %q", s2, `{("{")`)
	}
}

func TestLexerFilename(t *testing.T) {
	input := "foo"
	lexer := NewLexer("myfile.cramberry", input)
	tok := lexer.Next()
	if tok.Position.Filename != "myfile.cramberry" {
		t.Errorf("expected filename 'myfile.cramberry', got %q", tok.Position.Filename)
	}
}

func TestLexerMapType(t *testing.T) {
	// This tests lexing map[string]int32 which is a common pattern
	input := "map[string]int32"

	expected := []struct {
		typ   TokenType
		value string
	}{
		{TokenMap, "map"},
		{TokenLBracket, "["},
		{TokenIdent, "string"},
		{TokenRBracket, "]"},
		{TokenIdent, "int32"},
		{TokenEOF, ""},
	}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != exp.typ {
			t.Errorf("token %d: expected type %v, got %v", i, exp.typ, tok.Type)
		}
		if tok.Value != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexerPointerType(t *testing.T) {
	input := "*User"

	expected := []struct {
		typ   TokenType
		value string
	}{
		{TokenStar, "*"},
		{TokenIdent, "User"},
		{TokenEOF, ""},
	}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != exp.typ {
			t.Errorf("token %d: expected type %v, got %v", i, exp.typ, tok.Type)
		}
	}
}

func TestLexerArrayType(t *testing.T) {
	input := "[]string [5]byte"

	expected := []struct {
		typ   TokenType
		value string
	}{
		{TokenLBracket, "["},
		{TokenRBracket, "]"},
		{TokenIdent, "string"},
		{TokenLBracket, "["},
		{TokenInt, "5"},
		{TokenRBracket, "]"},
		{TokenIdent, "byte"},
		{TokenEOF, ""},
	}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != exp.typ {
			t.Errorf("token %d: expected type %v, got %v (value: %s)", i, exp.typ, tok.Type, tok.Value)
		}
	}
}

func TestLexerQualifiedType(t *testing.T) {
	input := "other.User"

	expected := []struct {
		typ   TokenType
		value string
	}{
		{TokenIdent, "other"},
		{TokenDot, "."},
		{TokenIdent, "User"},
		{TokenEOF, ""},
	}

	lexer := NewLexer("test.cramberry", input)
	for i, exp := range expected {
		tok := lexer.Next()
		if tok.Type != exp.typ {
			t.Errorf("token %d: expected type %v, got %v", i, exp.typ, tok.Type)
		}
		if tok.Value != exp.value {
			t.Errorf("token %d: expected value %q, got %q", i, exp.value, tok.Value)
		}
	}
}

func TestLexerUnicodeIdentifiers(t *testing.T) {
	input := "café Здравствуй 你好"

	lexer := NewLexer("test.cramberry", input)

	tok1 := lexer.Next()
	if tok1.Type != TokenIdent || tok1.Value != "café" {
		t.Errorf("expected Ident 'café', got %v %q", tok1.Type, tok1.Value)
	}

	tok2 := lexer.Next()
	if tok2.Type != TokenIdent || tok2.Value != "Здравствуй" {
		t.Errorf("expected Ident 'Здравствуй', got %v %q", tok2.Type, tok2.Value)
	}

	tok3 := lexer.Next()
	if tok3.Type != TokenIdent || tok3.Value != "你好" {
		t.Errorf("expected Ident '你好', got %v %q", tok3.Type, tok3.Value)
	}
}
