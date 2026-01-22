package schema

import (
	"fmt"
	"strconv"
)

// Parser parses schema source code into an AST.
type Parser struct {
	lexer     *Lexer
	current   Token
	previous  Token
	errors    []ParseError
	comments  []*Comment // Collected comments
}

// ParseError represents a parsing error.
type ParseError struct {
	Position Position
	Message  string
}

func (e ParseError) Error() string {
	return fmt.Sprintf("%s:%d:%d: %s", e.Position.Filename, e.Position.Line, e.Position.Column, e.Message)
}

// NewParser creates a new parser for the given input.
func NewParser(filename, input string) *Parser {
	p := &Parser{
		lexer: NewLexer(filename, input),
	}
	p.advance() // Load first token
	return p
}

// Parse parses the entire schema file.
func (p *Parser) Parse() (*Schema, []ParseError) {
	schema := &Schema{
		Position: p.current.Position,
	}

	// Collect leading comments
	p.collectComments()

	// Parse package declaration (optional)
	if p.check(TokenPackage) {
		pkg, err := p.parsePackage()
		if err != nil {
			p.errors = append(p.errors, *err)
		} else {
			schema.Package = pkg
		}
	}

	// Parse imports
	for p.check(TokenImport) {
		imp, err := p.parseImport()
		if err != nil {
			p.errors = append(p.errors, *err)
			p.synchronize()
		} else {
			schema.Imports = append(schema.Imports, imp)
		}
	}

	// Parse top-level options
	for p.check(TokenOption) {
		opt, err := p.parseOption()
		if err != nil {
			p.errors = append(p.errors, *err)
			p.synchronize()
		} else {
			schema.Options = append(schema.Options, opt)
		}
	}

	// Parse messages, enums, and interfaces
	for !p.check(TokenEOF) {
		p.collectComments()

		switch {
		case p.check(TokenMessage):
			msg, err := p.parseMessage()
			if err != nil {
				p.errors = append(p.errors, *err)
				p.synchronize()
			} else {
				schema.Messages = append(schema.Messages, msg)
			}
		case p.check(TokenEnum):
			enum, err := p.parseEnum()
			if err != nil {
				p.errors = append(p.errors, *err)
				p.synchronize()
			} else {
				schema.Enums = append(schema.Enums, enum)
			}
		case p.check(TokenInterface):
			iface, err := p.parseInterface()
			if err != nil {
				p.errors = append(p.errors, *err)
				p.synchronize()
			} else {
				schema.Interfaces = append(schema.Interfaces, iface)
			}
		case p.check(TokenComment), p.check(TokenDocComment):
			p.advance()
		case p.check(TokenEOF):
			break
		default:
			p.errors = append(p.errors, ParseError{
				Position: p.current.Position,
				Message:  fmt.Sprintf("unexpected token: %s", p.current.Type),
			})
			p.advance()
		}
	}

	schema.Comments = p.comments
	return schema, p.errors
}

// parsePackage parses: 'package' identifier ';'
func (p *Parser) parsePackage() (*Package, *ParseError) {
	startPos := p.current.Position
	p.advance() // consume 'package'

	if !p.check(TokenIdent) {
		return nil, p.error("expected package name")
	}
	name := p.current.Value
	p.advance()

	endPos := p.current.Position
	if !p.consume(TokenSemicolon, "expected ';' after package name") {
		return nil, p.error("expected ';' after package name")
	}

	return &Package{
		Position: startPos,
		EndPos:   endPos,
		Name:     name,
	}, nil
}

// parseImport parses: 'import' string ('as' identifier)? ';'
func (p *Parser) parseImport() (*Import, *ParseError) {
	startPos := p.current.Position
	p.advance() // consume 'import'

	if !p.check(TokenString) {
		return nil, p.error("expected import path string")
	}
	path := p.current.Value
	p.advance()

	var alias string
	if p.check(TokenAs) {
		p.advance()
		if !p.check(TokenIdent) {
			return nil, p.error("expected alias name after 'as'")
		}
		alias = p.current.Value
		p.advance()
	}

	endPos := p.current.Position
	if !p.consume(TokenSemicolon, "expected ';' after import") {
		return nil, p.error("expected ';' after import")
	}

	return &Import{
		Position: startPos,
		EndPos:   endPos,
		Path:     path,
		Alias:    alias,
	}, nil
}

// parseOption parses: 'option' identifier '=' value ';'
func (p *Parser) parseOption() (*Option, *ParseError) {
	startPos := p.current.Position
	p.advance() // consume 'option'

	if !p.check(TokenIdent) {
		return nil, p.error("expected option name")
	}
	name := p.current.Value
	p.advance()

	if !p.consume(TokenEquals, "expected '=' after option name") {
		return nil, p.error("expected '=' after option name")
	}

	value, err := p.parseValue()
	if err != nil {
		return nil, err
	}

	endPos := p.current.Position
	if !p.consume(TokenSemicolon, "expected ';' after option value") {
		return nil, p.error("expected ';' after option value")
	}

	return &Option{
		Position: startPos,
		EndPos:   endPos,
		Name:     name,
		Value:    value,
	}, nil
}

// parseValue parses a value (string, number, bool, or list).
func (p *Parser) parseValue() (Value, *ParseError) {
	startPos := p.current.Position

	switch p.current.Type {
	case TokenString:
		value := p.current.Value
		endPos := p.current.Position
		endPos.Column += len(p.current.Value) + 2 // Account for quotes
		p.advance()
		return &StringValue{
			Position: startPos,
			EndPos:   endPos,
			Value:    value,
		}, nil

	case TokenInt, TokenFloat:
		value := p.current.Value
		isFloat := p.current.Type == TokenFloat
		endPos := p.current.Position
		endPos.Column += len(value)
		p.advance()
		return &NumberValue{
			Position: startPos,
			EndPos:   endPos,
			Value:    value,
			IsFloat:  isFloat,
		}, nil

	case TokenTrue:
		endPos := p.current.Position
		endPos.Column += 4
		p.advance()
		return &BoolValue{
			Position: startPos,
			EndPos:   endPos,
			Value:    true,
		}, nil

	case TokenFalse:
		endPos := p.current.Position
		endPos.Column += 5
		p.advance()
		return &BoolValue{
			Position: startPos,
			EndPos:   endPos,
			Value:    false,
		}, nil

	case TokenLBracket:
		return p.parseListValue()

	default:
		return nil, p.error("expected value")
	}
}

// parseListValue parses: '[' value (',' value)* ']'
func (p *Parser) parseListValue() (*ListValue, *ParseError) {
	startPos := p.current.Position
	p.advance() // consume '['

	var values []Value
	for !p.check(TokenRBracket) && !p.check(TokenEOF) {
		val, err := p.parseValue()
		if err != nil {
			return nil, err
		}
		values = append(values, val)

		if !p.check(TokenRBracket) {
			if !p.consume(TokenComma, "expected ',' or ']'") {
				return nil, p.error("expected ',' or ']'")
			}
		}
	}

	endPos := p.current.Position
	if !p.consume(TokenRBracket, "expected ']'") {
		return nil, p.error("expected ']'")
	}

	return &ListValue{
		Position: startPos,
		EndPos:   endPos,
		Values:   values,
	}, nil
}

// parseMessage parses: 'message' identifier '{' field* '}'
func (p *Parser) parseMessage() (*Message, *ParseError) {
	docComments := p.getDocComments()
	startPos := p.current.Position
	p.advance() // consume 'message'

	if !p.check(TokenIdent) {
		return nil, p.error("expected message name")
	}
	name := p.current.Value
	p.advance()

	// Check for type ID annotation: @123
	var typeID int
	if p.check(TokenAt) {
		p.advance()
		if !p.check(TokenInt) {
			return nil, p.error("expected type ID after '@'")
		}
		id, err := strconv.Atoi(p.current.Value)
		if err != nil {
			return nil, p.error("invalid type ID")
		}
		typeID = id
		p.advance()
	}

	if !p.consume(TokenLBrace, "expected '{' after message name") {
		return nil, p.error("expected '{' after message name")
	}

	var fields []*Field
	var options []*Option
	for !p.check(TokenRBrace) && !p.check(TokenEOF) {
		p.collectComments()

		if p.check(TokenOption) {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			options = append(options, opt)
		} else if p.check(TokenRBrace) {
			break
		} else {
			field, err := p.parseField()
			if err != nil {
				return nil, err
			}
			fields = append(fields, field)
		}
	}

	endPos := p.current.Position
	if !p.consume(TokenRBrace, "expected '}'") {
		return nil, p.error("expected '}'")
	}

	return &Message{
		Position: startPos,
		EndPos:   endPos,
		Name:     name,
		Fields:   fields,
		Options:  options,
		Comments: docComments,
		TypeID:   typeID,
	}, nil
}

// parseField parses: modifier? type identifier '=' number options? ';'
func (p *Parser) parseField() (*Field, *ParseError) {
	docComments := p.getDocComments()
	startPos := p.current.Position

	// Parse modifiers
	var required, repeated, optional, deprecated bool
	for {
		switch p.current.Type {
		case TokenRequired:
			required = true
			p.advance()
		case TokenRepeated:
			repeated = true
			p.advance()
		case TokenOptional:
			optional = true
			p.advance()
		case TokenDeprecated:
			deprecated = true
			p.advance()
		default:
			goto parseType
		}
	}
parseType:

	// Parse type
	typeRef, err := p.parseTypeRef()
	if err != nil {
		return nil, err
	}

	// Parse field name
	if !p.check(TokenIdent) {
		return nil, p.error("expected field name")
	}
	name := p.current.Value
	p.advance()

	// Parse '='
	if !p.consume(TokenEquals, "expected '=' after field name") {
		return nil, p.error("expected '=' after field name")
	}

	// Parse field number
	if !p.check(TokenInt) {
		return nil, p.error("expected field number")
	}
	num, parseErr := strconv.Atoi(p.current.Value)
	if parseErr != nil {
		return nil, p.error("invalid field number")
	}
	p.advance()

	// Parse optional field options
	var options []*Option
	if p.check(TokenLBracket) {
		opts, err := p.parseFieldOptions()
		if err != nil {
			return nil, err
		}
		options = opts
	}

	endPos := p.current.Position
	if !p.consume(TokenSemicolon, "expected ';' after field") {
		return nil, p.error("expected ';' after field")
	}

	field := &Field{
		Position:   startPos,
		EndPos:     endPos,
		Name:       name,
		Number:     num,
		Type:       typeRef,
		Options:    options,
		Comments:   docComments,
		Required:   required,
		Repeated:   repeated,
		Optional:   optional,
		Deprecated: deprecated,
	}

	// Handle map type specially
	if mt, ok := typeRef.(*MapType); ok {
		field.MapKey = mt.Key
		field.MapValue = mt.Value
	}

	return field, nil
}

// parseFieldOptions parses: '[' (identifier '=' value)* ']'
func (p *Parser) parseFieldOptions() ([]*Option, *ParseError) {
	p.advance() // consume '['

	var options []*Option
	for !p.check(TokenRBracket) && !p.check(TokenEOF) {
		startPos := p.current.Position

		if !p.check(TokenIdent) {
			return nil, p.error("expected option name")
		}
		name := p.current.Value
		p.advance()

		if !p.consume(TokenEquals, "expected '=' after option name") {
			return nil, p.error("expected '=' after option name")
		}

		value, err := p.parseValue()
		if err != nil {
			return nil, err
		}

		options = append(options, &Option{
			Position: startPos,
			EndPos:   p.previous.Position,
			Name:     name,
			Value:    value,
		})

		if !p.check(TokenRBracket) && !p.check(TokenComma) {
			break
		}
		if p.check(TokenComma) {
			p.advance()
		}
	}

	if !p.consume(TokenRBracket, "expected ']'") {
		return nil, p.error("expected ']'")
	}

	return options, nil
}

// parseTypeRef parses a type reference.
func (p *Parser) parseTypeRef() (TypeRef, *ParseError) {
	startPos := p.current.Position

	// Pointer type: '*' type
	if p.check(TokenStar) {
		p.advance()
		elem, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		return &PointerType{
			Position: startPos,
			EndPos:   elem.End(),
			Element:  elem,
		}, nil
	}

	// Array type: '[' size? ']' type
	if p.check(TokenLBracket) {
		p.advance()
		var size int
		if p.check(TokenInt) {
			sz, err := strconv.Atoi(p.current.Value)
			if err != nil {
				return nil, p.error("invalid array size")
			}
			size = sz
			p.advance()
		}
		if !p.consume(TokenRBracket, "expected ']'") {
			return nil, p.error("expected ']'")
		}
		elem, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		return &ArrayType{
			Position: startPos,
			EndPos:   elem.End(),
			Element:  elem,
			Size:     size,
		}, nil
	}

	// Map type: 'map' '[' keyType ']' valueType
	if p.check(TokenMap) {
		p.advance()
		if !p.consume(TokenLBracket, "expected '[' after 'map'") {
			return nil, p.error("expected '[' after 'map'")
		}
		keyType, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		if !p.consume(TokenRBracket, "expected ']' after map key type") {
			return nil, p.error("expected ']' after map key type")
		}
		valueType, err := p.parseTypeRef()
		if err != nil {
			return nil, err
		}
		return &MapType{
			Position: startPos,
			EndPos:   valueType.End(),
			Key:      keyType,
			Value:    valueType,
		}, nil
	}

	// Named type or scalar type: identifier ('.' identifier)?
	if !p.check(TokenIdent) {
		return nil, p.error("expected type name")
	}

	name := p.current.Value
	endPos := p.current.Position
	endPos.Column += len(name)
	p.advance()

	// Check if it's a scalar type
	if IsScalar(name) {
		return &ScalarType{
			Position: startPos,
			EndPos:   endPos,
			Name:     name,
		}, nil
	}

	// Check for qualified name: package.Type
	var pkg string
	if p.check(TokenDot) {
		p.advance()
		if !p.check(TokenIdent) {
			return nil, p.error("expected type name after '.'")
		}
		pkg = name
		name = p.current.Value
		endPos = p.current.Position
		endPos.Column += len(name)
		p.advance()
	}

	return &NamedType{
		Position: startPos,
		EndPos:   endPos,
		Package:  pkg,
		Name:     name,
	}, nil
}

// parseEnum parses: 'enum' identifier '{' enumValue* '}'
func (p *Parser) parseEnum() (*Enum, *ParseError) {
	docComments := p.getDocComments()
	startPos := p.current.Position
	p.advance() // consume 'enum'

	if !p.check(TokenIdent) {
		return nil, p.error("expected enum name")
	}
	name := p.current.Value
	p.advance()

	if !p.consume(TokenLBrace, "expected '{' after enum name") {
		return nil, p.error("expected '{' after enum name")
	}

	var values []*EnumValue
	var options []*Option
	for !p.check(TokenRBrace) && !p.check(TokenEOF) {
		p.collectComments()

		if p.check(TokenOption) {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			options = append(options, opt)
		} else if p.check(TokenRBrace) {
			break
		} else {
			val, err := p.parseEnumValue()
			if err != nil {
				return nil, err
			}
			values = append(values, val)
		}
	}

	endPos := p.current.Position
	if !p.consume(TokenRBrace, "expected '}'") {
		return nil, p.error("expected '}'")
	}

	return &Enum{
		Position: startPos,
		EndPos:   endPos,
		Name:     name,
		Values:   values,
		Options:  options,
		Comments: docComments,
	}, nil
}

// parseEnumValue parses: identifier '=' number ';'
func (p *Parser) parseEnumValue() (*EnumValue, *ParseError) {
	docComments := p.getDocComments()
	startPos := p.current.Position

	if !p.check(TokenIdent) {
		return nil, p.error("expected enum value name")
	}
	name := p.current.Value
	p.advance()

	if !p.consume(TokenEquals, "expected '=' after enum value name") {
		return nil, p.error("expected '=' after enum value name")
	}

	if !p.check(TokenInt) {
		return nil, p.error("expected enum value number")
	}
	num, err := strconv.Atoi(p.current.Value)
	if err != nil {
		return nil, p.error("invalid enum value number")
	}
	p.advance()

	endPos := p.current.Position
	if !p.consume(TokenSemicolon, "expected ';' after enum value") {
		return nil, p.error("expected ';' after enum value")
	}

	return &EnumValue{
		Position: startPos,
		EndPos:   endPos,
		Name:     name,
		Number:   num,
		Comments: docComments,
	}, nil
}

// parseInterface parses: 'interface' identifier '{' implementation* '}'
func (p *Parser) parseInterface() (*Interface, *ParseError) {
	docComments := p.getDocComments()
	startPos := p.current.Position
	p.advance() // consume 'interface'

	if !p.check(TokenIdent) {
		return nil, p.error("expected interface name")
	}
	name := p.current.Value
	p.advance()

	if !p.consume(TokenLBrace, "expected '{' after interface name") {
		return nil, p.error("expected '{' after interface name")
	}

	var implementations []*Implementation
	var options []*Option
	for !p.check(TokenRBrace) && !p.check(TokenEOF) {
		p.collectComments()

		if p.check(TokenOption) {
			opt, err := p.parseOption()
			if err != nil {
				return nil, err
			}
			options = append(options, opt)
		} else if p.check(TokenRBrace) {
			break
		} else {
			impl, err := p.parseImplementation()
			if err != nil {
				return nil, err
			}
			implementations = append(implementations, impl)
		}
	}

	endPos := p.current.Position
	if !p.consume(TokenRBrace, "expected '}'") {
		return nil, p.error("expected '}'")
	}

	return &Interface{
		Position:        startPos,
		EndPos:          endPos,
		Name:            name,
		Implementations: implementations,
		Options:         options,
		Comments:        docComments,
	}, nil
}

// parseImplementation parses: number '=' identifier ';'
func (p *Parser) parseImplementation() (*Implementation, *ParseError) {
	docComments := p.getDocComments()
	startPos := p.current.Position

	if !p.check(TokenInt) {
		return nil, p.error("expected type ID")
	}
	typeID, err := strconv.Atoi(p.current.Value)
	if err != nil {
		return nil, p.error("invalid type ID")
	}
	p.advance()

	if !p.consume(TokenEquals, "expected '=' after type ID") {
		return nil, p.error("expected '=' after type ID")
	}

	// Parse the type name (could be qualified)
	if !p.check(TokenIdent) {
		return nil, p.error("expected type name")
	}

	typeStartPos := p.current.Position
	name := p.current.Value
	p.advance()

	var pkg string
	if p.check(TokenDot) {
		p.advance()
		if !p.check(TokenIdent) {
			return nil, p.error("expected type name after '.'")
		}
		pkg = name
		name = p.current.Value
		p.advance()
	}

	endPos := p.current.Position
	if !p.consume(TokenSemicolon, "expected ';' after implementation") {
		return nil, p.error("expected ';' after implementation")
	}

	return &Implementation{
		Position: startPos,
		EndPos:   endPos,
		TypeID:   typeID,
		Type: &NamedType{
			Position: typeStartPos,
			EndPos:   endPos,
			Package:  pkg,
			Name:     name,
		},
		Comments: docComments,
	}, nil
}

// Helper methods

func (p *Parser) advance() {
	p.previous = p.current
	p.current = p.lexer.Next()

	// Skip regular comments, but remember doc comments
	for p.current.Type == TokenComment {
		p.current = p.lexer.Next()
	}
}

func (p *Parser) check(typ TokenType) bool {
	return p.current.Type == typ
}

func (p *Parser) consume(typ TokenType, msg string) bool {
	if p.check(typ) {
		p.advance()
		return true
	}
	return false
}

func (p *Parser) error(msg string) *ParseError {
	return &ParseError{
		Position: p.current.Position,
		Message:  msg,
	}
}

// synchronize skips tokens until we find a likely sync point.
func (p *Parser) synchronize() {
	for !p.check(TokenEOF) {
		if p.previous.Type == TokenSemicolon || p.previous.Type == TokenRBrace {
			return
		}
		switch p.current.Type {
		case TokenPackage, TokenImport, TokenMessage, TokenEnum, TokenInterface:
			return
		}
		p.advance()
	}
}

// collectComments collects doc comments preceding the current position.
func (p *Parser) collectComments() {
	for p.current.Type == TokenDocComment || p.current.Type == TokenComment {
		if p.current.Type == TokenDocComment {
			p.comments = append(p.comments, &Comment{
				Position: p.current.Position,
				EndPos:   p.current.Position,
				Text:     p.current.Value,
				IsDoc:    true,
			})
		}
		p.current = p.lexer.Next()
	}
}

// getDocComments returns recent doc comments that apply to the next declaration.
func (p *Parser) getDocComments() []*Comment {
	// For now, return all collected doc comments and clear
	// A more sophisticated implementation would track positions
	result := make([]*Comment, len(p.comments))
	copy(result, p.comments)
	p.comments = nil
	return result
}

// ParseFile is a convenience function that parses a schema file.
func ParseFile(filename, input string) (*Schema, []ParseError) {
	parser := NewParser(filename, input)
	return parser.Parse()
}
