package schema

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TokenType represents the type of a token.
type TokenType int

// Token types.
const (
	TokenError TokenType = iota
	TokenEOF

	// Literals
	TokenIdent  // identifier
	TokenInt    // integer literal
	TokenFloat  // float literal
	TokenString // string literal

	// Keywords
	TokenPackage    // package
	TokenImport     // import
	TokenAs         // as
	TokenMessage    // message
	TokenEnum       // enum
	TokenInterface  // interface
	TokenOption     // option
	TokenRequired   // required
	TokenRepeated   // repeated
	TokenOptional   // optional
	TokenMap        // map
	TokenTrue       // true
	TokenFalse      // false
	TokenDeprecated // deprecated

	// Punctuation
	TokenLBrace    // {
	TokenRBrace    // }
	TokenLBracket  // [
	TokenRBracket  // ]
	TokenLParen    // (
	TokenRParen    // )
	TokenSemicolon // ;
	TokenColon     // :
	TokenComma     // ,
	TokenEquals    // =
	TokenDot       // .
	TokenStar      // *
	TokenAt        // @

	// Comments
	TokenComment    // // comment
	TokenDocComment // /// doc comment
)

// String returns a human-readable name for the token type.
func (t TokenType) String() string {
	switch t {
	case TokenError:
		return "Error"
	case TokenEOF:
		return "EOF"
	case TokenIdent:
		return "Ident"
	case TokenInt:
		return "Int"
	case TokenFloat:
		return "Float"
	case TokenString:
		return "String"
	case TokenPackage:
		return "package"
	case TokenImport:
		return "import"
	case TokenAs:
		return "as"
	case TokenMessage:
		return "message"
	case TokenEnum:
		return "enum"
	case TokenInterface:
		return "interface"
	case TokenOption:
		return "option"
	case TokenRequired:
		return "required"
	case TokenRepeated:
		return "repeated"
	case TokenOptional:
		return "optional"
	case TokenMap:
		return "map"
	case TokenTrue:
		return "true"
	case TokenFalse:
		return "false"
	case TokenDeprecated:
		return "deprecated"
	case TokenLBrace:
		return "{"
	case TokenRBrace:
		return "}"
	case TokenLBracket:
		return "["
	case TokenRBracket:
		return "]"
	case TokenLParen:
		return "("
	case TokenRParen:
		return ")"
	case TokenSemicolon:
		return ";"
	case TokenColon:
		return ":"
	case TokenComma:
		return ","
	case TokenEquals:
		return "="
	case TokenDot:
		return "."
	case TokenStar:
		return "*"
	case TokenAt:
		return "@"
	case TokenComment:
		return "Comment"
	case TokenDocComment:
		return "DocComment"
	default:
		return fmt.Sprintf("Token(%d)", t)
	}
}

// Token represents a lexical token.
type Token struct {
	Type     TokenType
	Value    string
	Position Position
}

// String returns a string representation of the token.
func (t Token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return t.Type.String()
}

// keywords maps keyword strings to their token types.
var keywords = map[string]TokenType{
	"package":    TokenPackage,
	"import":     TokenImport,
	"as":         TokenAs,
	"message":    TokenMessage,
	"enum":       TokenEnum,
	"interface":  TokenInterface,
	"option":     TokenOption,
	"required":   TokenRequired,
	"repeated":   TokenRepeated,
	"optional":   TokenOptional,
	"map":        TokenMap,
	"true":       TokenTrue,
	"false":      TokenFalse,
	"deprecated": TokenDeprecated,
}

// Lexer tokenizes schema source code.
type Lexer struct {
	filename string
	input    string
	pos      int      // current position in input
	line     int      // current line number (1-based)
	column   int      // current column number (1-based)
	start    int      // start position of current token
	startPos Position // position of current token start
}

// NewLexer creates a new lexer for the given input.
func NewLexer(filename, input string) *Lexer {
	return &Lexer{
		filename: filename,
		input:    input,
		pos:      0,
		line:     1,
		column:   1,
	}
}

// Next returns the next token from the input.
func (l *Lexer) Next() Token {
	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return Token{Type: TokenEOF, Position: l.currentPos()}
	}

	l.start = l.pos
	l.startPos = l.currentPos()

	ch := l.peek()

	// Handle comments
	if ch == '/' && l.pos+1 < len(l.input) && l.input[l.pos+1] == '/' {
		return l.scanComment()
	}

	// Handle identifiers and keywords
	if isLetter(ch) || ch == '_' {
		return l.scanIdent()
	}

	// Handle numbers
	if isDigit(ch) || (ch == '-' && l.pos+1 < len(l.input) && isDigit(rune(l.input[l.pos+1]))) {
		return l.scanNumber()
	}

	// Handle strings
	if ch == '"' {
		return l.scanString()
	}

	// Handle punctuation
	l.advance()
	switch ch {
	case '{':
		return l.token(TokenLBrace, "{")
	case '}':
		return l.token(TokenRBrace, "}")
	case '[':
		return l.token(TokenLBracket, "[")
	case ']':
		return l.token(TokenRBracket, "]")
	case '(':
		return l.token(TokenLParen, "(")
	case ')':
		return l.token(TokenRParen, ")")
	case ';':
		return l.token(TokenSemicolon, ";")
	case ':':
		return l.token(TokenColon, ":")
	case ',':
		return l.token(TokenComma, ",")
	case '=':
		return l.token(TokenEquals, "=")
	case '.':
		return l.token(TokenDot, ".")
	case '*':
		return l.token(TokenStar, "*")
	case '@':
		return l.token(TokenAt, "@")
	default:
		return l.errorf("unexpected character: %q", ch)
	}
}

// Peek returns the next token without consuming it.
func (l *Lexer) Peek() Token {
	// Save state
	pos := l.pos
	line := l.line
	column := l.column

	tok := l.Next()

	// Restore state
	l.pos = pos
	l.line = line
	l.column = column

	return tok
}

// scanComment scans a comment token.
func (l *Lexer) scanComment() Token {
	// Consume //
	l.advance()
	l.advance()

	// Check for doc comment ///
	isDoc := false
	if l.pos < len(l.input) && l.input[l.pos] == '/' {
		isDoc = true
		l.advance()
	}

	// Scan until end of line
	start := l.pos
	for l.pos < len(l.input) && l.input[l.pos] != '\n' {
		l.advance()
	}

	text := strings.TrimSpace(l.input[start:l.pos])
	if isDoc {
		return l.token(TokenDocComment, text)
	}
	return l.token(TokenComment, text)
}

// scanIdent scans an identifier or keyword.
func (l *Lexer) scanIdent() Token {
	for l.pos < len(l.input) {
		ch := l.peek()
		if !isLetter(ch) && !isDigit(ch) && ch != '_' {
			break
		}
		l.advance()
	}

	ident := l.input[l.start:l.pos]

	// Check for keywords
	if tokType, ok := keywords[ident]; ok {
		return l.token(tokType, ident)
	}

	return l.token(TokenIdent, ident)
}

// scanNumber scans a number literal.
func (l *Lexer) scanNumber() Token {
	// Handle negative numbers
	if l.peek() == '-' {
		l.advance()
	}

	// Scan integer part
	for l.pos < len(l.input) && isDigit(l.peek()) {
		l.advance()
	}

	// Check for float
	isFloat := false
	if l.pos < len(l.input) && l.input[l.pos] == '.' {
		// Look ahead to make sure it's a decimal point, not field access
		if l.pos+1 < len(l.input) && isDigit(rune(l.input[l.pos+1])) {
			isFloat = true
			l.advance() // consume .
			for l.pos < len(l.input) && isDigit(l.peek()) {
				l.advance()
			}
		}
	}

	// Check for exponent
	if l.pos < len(l.input) && (l.input[l.pos] == 'e' || l.input[l.pos] == 'E') {
		isFloat = true
		l.advance()
		if l.pos < len(l.input) && (l.input[l.pos] == '+' || l.input[l.pos] == '-') {
			l.advance()
		}
		for l.pos < len(l.input) && isDigit(l.peek()) {
			l.advance()
		}
	}

	num := l.input[l.start:l.pos]
	if isFloat {
		return l.token(TokenFloat, num)
	}
	return l.token(TokenInt, num)
}

// scanString scans a string literal.
func (l *Lexer) scanString() Token {
	// Consume opening quote
	l.advance()

	var sb strings.Builder
	for {
		if l.pos >= len(l.input) {
			return l.errorf("unterminated string")
		}

		ch := l.input[l.pos]
		if ch == '"' {
			l.advance()
			break
		}

		if ch == '\n' {
			return l.errorf("newline in string literal")
		}

		if ch == '\\' {
			l.advance()
			if l.pos >= len(l.input) {
				return l.errorf("unterminated string escape")
			}
			escaped := l.input[l.pos]
			switch escaped {
			case 'n':
				sb.WriteByte('\n')
			case 't':
				sb.WriteByte('\t')
			case 'r':
				sb.WriteByte('\r')
			case '\\':
				sb.WriteByte('\\')
			case '"':
				sb.WriteByte('"')
			case '0':
				sb.WriteByte('\x00')
			default:
				return l.errorf("unknown escape sequence: \\%c", escaped)
			}
			l.advance()
		} else {
			// Handle multi-byte UTF-8 characters correctly
			r, size := utf8.DecodeRuneInString(l.input[l.pos:])
			if r == utf8.RuneError && size == 1 {
				return l.errorf("invalid UTF-8 sequence in string")
			}
			sb.WriteRune(r)
			l.pos += size
			l.column++
		}
	}

	return l.token(TokenString, sb.String())
}

// Helper methods

func (l *Lexer) currentPos() Position {
	return Position{
		Filename: l.filename,
		Line:     l.line,
		Column:   l.column,
		Offset:   l.pos,
	}
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

func (l *Lexer) advance() {
	if l.pos >= len(l.input) {
		return
	}
	if l.input[l.pos] == '\n' {
		l.line++
		l.column = 1
	} else {
		l.column++
	}
	_, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			l.advance()
		} else {
			break
		}
	}
}

func (l *Lexer) token(typ TokenType, value string) Token {
	return Token{
		Type:     typ,
		Value:    value,
		Position: l.startPos,
	}
}

func (l *Lexer) errorf(format string, args ...any) Token {
	return Token{
		Type:     TokenError,
		Value:    fmt.Sprintf(format, args...),
		Position: l.startPos,
	}
}

func isLetter(ch rune) bool {
	return unicode.IsLetter(ch)
}

func isDigit(ch rune) bool {
	return ch >= '0' && ch <= '9'
}

// Tokenize returns all tokens from the input.
// This is useful for testing and debugging.
func Tokenize(filename, input string) []Token {
	lexer := NewLexer(filename, input)
	var tokens []Token
	for {
		tok := lexer.Next()
		tokens = append(tokens, tok)
		if tok.Type == TokenEOF || tok.Type == TokenError {
			break
		}
	}
	return tokens
}
