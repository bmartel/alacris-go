package gen

import (
	"fmt"
	"strconv"
	"strings"
)

// jsKind is the shape of a JavaScript literal.
type jsKind uint8

const (
	jsOpaque jsKind = iota // valid JavaScript that is not literal data
	jsString
	jsNumber
	jsBool
	jsNull
	jsArray
	jsObject
)

func (k jsKind) String() string {
	switch k {
	case jsString:
		return "string"
	case jsNumber:
		return "number"
	case jsBool:
		return "boolean"
	case jsNull:
		return "null"
	case jsArray:
		return "array"
	case jsObject:
		return "object"
	}
	return "expression"
}

type jsValue struct {
	kind jsKind
	str  string
	num  float64
	// integral records that the literal was written without a fraction or an
	// exponent, which is what decides between int and float64 on the Go side.
	integral bool
	boolean  bool
	arr      []jsValue
	obj      []jsField
	raw      string
	line     int

	// partial marks an object that held something the parser could not read as
	// data — a spread, a computed key, a method. The members it did read are
	// still there, but the object is no longer a complete description of
	// itself.
	partial bool

	// skipped names the members that were given up on but did have a readable
	// key, which is enough to recognise an options object by its shape.
	skipped []string
}

// hasKey reports whether the object has the named member, read or skipped.
func (v jsValue) hasKey(key string) bool {
	if _, ok := v.get(key); ok {
		return true
	}
	for _, k := range v.skipped {
		if k == key {
			return true
		}
	}
	return false
}

type jsField struct {
	key  string
	val  jsValue
	doc  string
	line int
}

// get returns the named field of an object value.
func (v jsValue) get(key string) (jsValue, bool) {
	for _, f := range v.obj {
		if f.key == key {
			return f.val, true
		}
	}
	return jsValue{}, false
}

// parser walks a pre-lexed token stream.
type parser struct {
	toks []token
	pos  int
	file string
}

func newParser(file, src string) (*parser, error) {
	l := newLexer(src)
	var toks []token
	for {
		tok, err := l.next()
		if err != nil {
			return nil, fmt.Errorf("%s:%w", file, err)
		}
		toks = append(toks, tok)
		if tok.kind == tEOF {
			break
		}
	}
	return &parser{toks: toks, file: file}, nil
}

func (p *parser) at(i int) token {
	if i < 0 || i >= len(p.toks) {
		return token{kind: tEOF}
	}
	return p.toks[i]
}

func (p *parser) cur() token  { return p.at(p.pos) }
func (p *parser) peek() token { return p.at(p.pos + 1) }

func (p *parser) advance() token {
	t := p.cur()
	if t.kind != tEOF {
		p.pos++
	}
	return t
}

func (p *parser) errf(t token, format string, args ...any) error {
	return fmt.Errorf("%s:%d: %s", p.file, t.line, fmt.Sprintf(format, args...))
}

// expect consumes a specific punctuation token.
func (p *parser) expect(punct string) error {
	t := p.cur()
	if !t.is(punct) {
		return p.errf(t, "expected %q, found %s %q", punct, t.kind, t.text)
	}
	p.advance()
	return nil
}

// parseValue reads a literal. Anything that is not literal data is consumed
// and returned as jsOpaque, so a props object can sit beside a setup function
// without the parser having to understand the function.
func (p *parser) parseValue() (jsValue, error) {
	t := p.cur()
	switch {
	case t.kind == tString:
		p.advance()
		return jsValue{kind: jsString, str: t.val, raw: t.text, line: t.line}, nil

	case t.kind == tTemplate && !t.hasSubst:
		p.advance()
		return jsValue{kind: jsString, str: t.val, raw: t.text, line: t.line}, nil

	case t.kind == tNumber:
		p.advance()
		return numberValue(t)

	case t.is("-") && p.peek().kind == tNumber:
		p.advance()
		num := p.advance()
		v, err := numberValue(num)
		if err != nil {
			return jsValue{}, err
		}
		v.num = -v.num
		v.raw = "-" + v.raw
		v.line = t.line
		return v, nil

	case t.kind == tIdent && (t.val == "true" || t.val == "false"):
		p.advance()
		return jsValue{kind: jsBool, boolean: t.val == "true", raw: t.text, line: t.line}, nil

	case t.kind == tIdent && (t.val == "null" || t.val == "undefined"):
		p.advance()
		return jsValue{kind: jsNull, raw: t.text, line: t.line}, nil

	case t.is("["):
		return p.parseArray()

	case t.is("{"):
		return p.parseObject()
	}
	return p.skipOpaque()
}

func numberValue(t token) (jsValue, error) {
	text := strings.ReplaceAll(t.val, "_", "")
	if strings.HasSuffix(text, "n") {
		return jsValue{}, fmt.Errorf("line %d: BigInt literal %s has no JavaScript number equivalent", t.line, t.text)
	}

	var (
		n   float64
		err error
	)
	switch {
	case strings.HasPrefix(text, "0x"), strings.HasPrefix(text, "0X"),
		strings.HasPrefix(text, "0o"), strings.HasPrefix(text, "0O"),
		strings.HasPrefix(text, "0b"), strings.HasPrefix(text, "0B"):
		var i int64
		i, err = strconv.ParseInt(text[2:], baseOf(text[1]), 64)
		n = float64(i)
	default:
		n, err = strconv.ParseFloat(text, 64)
	}
	if err != nil {
		return jsValue{}, fmt.Errorf("line %d: cannot read number %s", t.line, t.text)
	}

	integral := !strings.ContainsAny(text, ".eE") || strings.HasPrefix(strings.ToLower(text), "0x")
	if integral && n != float64(int64(n)) {
		integral = false
	}
	return jsValue{kind: jsNumber, num: n, integral: integral, raw: t.text, line: t.line}, nil
}

func baseOf(c byte) int {
	switch c {
	case 'x', 'X':
		return 16
	case 'o', 'O':
		return 8
	}
	return 2
}

func (p *parser) parseArray() (jsValue, error) {
	open := p.cur()
	p.advance()
	out := jsValue{kind: jsArray, line: open.line}
	for {
		if p.cur().kind == tEOF {
			return jsValue{}, p.errf(open, "unterminated array literal")
		}
		if p.cur().is("]") {
			p.advance()
			return out, nil
		}
		if p.cur().is(",") {
			p.advance()
			continue
		}
		if p.cur().is(".") { // a spread element makes the array non-literal
			return p.skipBalancedFrom(open, "]", jsOpaque)
		}
		v, err := p.parseValue()
		if err != nil {
			return jsValue{}, err
		}
		if v.kind == jsOpaque {
			return p.skipBalancedFrom(open, "]", jsOpaque)
		}
		out.arr = append(out.arr, v)
	}
}

func (p *parser) parseObject() (jsValue, error) {
	open := p.cur()
	p.advance()
	out := jsValue{kind: jsObject, line: open.line}
	var pendingDoc string

	for {
		t := p.cur()
		switch {
		case t.kind == tEOF:
			return jsValue{}, p.errf(open, "unterminated object literal")

		case t.is("}"):
			p.advance()
			return out, nil

		case t.is(","):
			p.advance()
			continue

		case t.kind == tDoc:
			pendingDoc = t.val
			p.advance()
			continue
		}

		// A key: an identifier, a string, or a number. A computed key or a
		// spread is something else, and only that member is given up on —
		// alacris' own options object mixes data (props) with code (setup), so
		// abandoning the whole object at the first function would lose the
		// props sitting next to it.
		var key string
		switch {
		case t.kind == tIdent, t.kind == tString, t.kind == tNumber:
			key = t.val
			p.advance()
		default:
			out.partial = true
			if err := p.skipMember(); err != nil {
				return jsValue{}, err
			}
			pendingDoc = ""
			continue
		}

		if !p.cur().is(":") {
			// Shorthand ({ setup }) or a method ({ setup() {} }): both refer to
			// something defined elsewhere. The key still counts as present.
			out.partial = true
			out.skipped = append(out.skipped, key)
			if err := p.skipMember(); err != nil {
				return jsValue{}, err
			}
			pendingDoc = ""
			continue
		}
		p.advance()

		v, err := p.parseValue()
		if err != nil {
			return jsValue{}, err
		}
		out.obj = append(out.obj, jsField{key: key, val: v, doc: pendingDoc, line: t.line})
		pendingDoc = ""
	}
}

// skipMember consumes the rest of one object member, leaving the parser on the
// object's closing brace or just past the comma that ended the member.
func (p *parser) skipMember() error {
	depth := 0
	for {
		t := p.cur()
		if t.kind == tEOF {
			return p.errf(t, "unterminated object literal")
		}
		if t.kind == tPunct {
			switch t.val {
			case "(", "[", "{":
				depth++
			case ")", "]":
				if depth == 0 {
					return nil
				}
				depth--
			case "}":
				if depth == 0 {
					return nil // the caller's loop consumes it
				}
				depth--
			case ",":
				if depth == 0 {
					p.advance()
					return nil
				}
			}
		}
		p.advance()
	}
}

// skipOpaque consumes one expression that is not literal data, stopping at the
// comma or closing bracket that ends it.
func (p *parser) skipOpaque() (jsValue, error) {
	start := p.cur()
	if start.kind == tEOF {
		return jsValue{}, p.errf(start, "unexpected end of file")
	}
	depth := 0
	for {
		t := p.cur()
		if t.kind == tEOF {
			break
		}
		if t.kind == tPunct {
			switch t.val {
			case "(", "[", "{":
				depth++
			case ")", "]", "}":
				if depth == 0 {
					return jsValue{kind: jsOpaque, raw: start.text, line: start.line}, nil
				}
				depth--
			case ",":
				if depth == 0 {
					return jsValue{kind: jsOpaque, raw: start.text, line: start.line}, nil
				}
			}
		}
		p.advance()
	}
	return jsValue{kind: jsOpaque, raw: start.text, line: start.line}, nil
}

// skipBalancedFrom abandons a partially-read bracketed construct, leaving the
// parser just past its closing bracket.
func (p *parser) skipBalancedFrom(open token, closing string, kind jsKind) (jsValue, error) {
	depth := 1
	for {
		t := p.cur()
		if t.kind == tEOF {
			return jsValue{}, p.errf(open, "unterminated %s", map[string]string{"}": "object literal", "]": "array literal"}[closing])
		}
		p.advance()
		if t.kind != tPunct {
			continue
		}
		switch t.val {
		case "(", "[", "{":
			depth++
		case ")", "]", "}":
			depth--
			if depth == 0 {
				return jsValue{kind: kind, raw: open.text, line: open.line}, nil
			}
		}
	}
}
