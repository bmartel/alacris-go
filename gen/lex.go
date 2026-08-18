package gen

import (
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// A small JavaScript lexer.
//
// It exists to find define() calls and read the object literal passed to them.
// It is not a JavaScript parser and does not try to be: everything it cannot
// interpret it skips over as balanced source text, and a props object it
// cannot read as literal data is reported as an error rather than guessed at.
//
// The parts that have to be right are the ones that decide where a token ends
// — strings, template literals with nested substitutions, comments and regex
// literals — because getting any of those wrong makes the lexer read code as
// data.

type tokKind uint8

const (
	tEOF tokKind = iota
	tIdent
	tString   // val holds the decoded contents
	tTemplate // val holds the contents, only when there are no substitutions
	tNumber
	tPunct
	tRegex
	tDoc // a /** ... */ comment, val holds the raw text
)

func (k tokKind) String() string {
	switch k {
	case tEOF:
		return "end of file"
	case tIdent:
		return "identifier"
	case tString:
		return "string"
	case tTemplate:
		return "template literal"
	case tNumber:
		return "number"
	case tPunct:
		return "punctuation"
	case tRegex:
		return "regular expression"
	case tDoc:
		return "doc comment"
	}
	return "token"
}

type token struct {
	kind tokKind
	text string // exactly as it appears in the source
	val  string // decoded value: string contents, identifier name, punctuation
	line int
	off  int

	// hasSubst marks a template literal that interpolates something, which
	// therefore has no literal value.
	hasSubst bool
}

func (t token) is(p string) bool { return t.kind == tPunct && t.val == p }

type lexer struct {
	src  string
	pos  int
	line int

	// prev is the last token that can end an expression, which is what decides
	// whether a '/' opens a regex or divides.
	prev token
}

// bom is a UTF-8 byte order mark, written as an escape so this file has
// none of its own.
const (
	bomRune = '\uFEFF'
	bom     = string(bomRune)
)

func newLexer(src string) *lexer {
	// A leading BOM is not whitespace to the lexer, so drop it up front.
	src = strings.TrimPrefix(src, bom)
	return &lexer{src: src, line: 1}
}

type syntaxError struct {
	line int
	msg  string
}

func (e *syntaxError) Error() string { return fmt.Sprintf("line %d: %s", e.line, e.msg) }

func (l *lexer) errf(line int, format string, args ...any) error {
	return &syntaxError{line: line, msg: fmt.Sprintf(format, args...)}
}

// next returns the next token, skipping whitespace and non-doc comments.
func (l *lexer) next() (token, error) {
	for {
		l.skipSpace()
		if l.pos >= len(l.src) {
			return token{kind: tEOF, line: l.line, off: l.pos}, nil
		}

		start, startLine := l.pos, l.line
		c := l.src[l.pos]

		switch {
		case c == '/' && l.peekAt(1) == '/':
			// A run of // lines is a doc comment when it carries @tags or
			// spans more than one line — the shape Alacris UI uses at the
			// top of each component file. A single // line without tags is
			// an ordinary comment and must stay invisible, or a trailing
			// `const x = 1; // note` would attach to the next define().
			start, startLine := l.pos, l.line
			lines, hasTag := 0, false
			for l.pos < len(l.src) && l.src[l.pos] == '/' && l.peekAt(1) == '/' {
				lines++
				l.pos += 2
				lineStart := l.pos
				for l.pos < len(l.src) && l.src[l.pos] != '\n' {
					l.pos++
				}
				if strings.Contains(l.src[lineStart:l.pos], "@") {
					hasTag = true
				}
				if l.pos < len(l.src) && l.src[l.pos] == '\n' {
					l.line++
					l.pos++
				}
				for l.pos < len(l.src) && (l.src[l.pos] == ' ' || l.src[l.pos] == '\t') {
					l.pos++
				}
			}
			if lines >= 2 || hasTag {
				return token{kind: tDoc, text: l.src[start:l.pos], val: l.src[start:l.pos], line: startLine, off: start}, nil
			}
			continue

		case c == '/' && l.peekAt(1) == '*':
			doc := strings.HasPrefix(l.src[l.pos:], "/**")
			l.pos += 2
			for {
				if l.pos >= len(l.src) {
					return token{}, l.errf(startLine, "unterminated block comment")
				}
				if l.src[l.pos] == '*' && l.peekAt(1) == '/' {
					l.pos += 2
					break
				}
				if l.src[l.pos] == '\n' {
					l.line++
				}
				l.pos++
			}
			if doc {
				// A doc comment does not end an expression, so prev is left
				// alone and the regex heuristic still sees the real token.
				return token{kind: tDoc, text: l.src[start:l.pos], val: l.src[start:l.pos], line: startLine, off: start}, nil
			}
			continue

		case c == '/':
			if l.regexAllowed() {
				tok, err := l.lexRegex()
				if err != nil {
					return token{}, err
				}
				l.prev = tok
				return tok, nil
			}
			l.pos++
			tok := token{kind: tPunct, text: "/", val: "/", line: startLine, off: start}
			l.prev = tok
			return tok, nil

		case c == '\'' || c == '"':
			tok, err := l.lexString(c)
			if err != nil {
				return token{}, err
			}
			l.prev = tok
			return tok, nil

		case c == '`':
			tok, err := l.lexTemplate()
			if err != nil {
				return token{}, err
			}
			l.prev = tok
			return tok, nil

		case isDigit(c) || (c == '.' && isDigit(l.peekAt(1))):
			tok, err := l.lexNumber()
			if err != nil {
				return token{}, err
			}
			l.prev = tok
			return tok, nil

		case isIdentStart(rune(c)) || c >= utf8.RuneSelf:
			tok := l.lexIdent()
			l.prev = tok
			return tok, nil

		default:
			l.pos++
			tok := token{kind: tPunct, text: string(c), val: string(c), line: startLine, off: start}
			l.prev = tok
			return tok, nil
		}
	}
}

func (l *lexer) skipSpace() {
	for l.pos < len(l.src) {
		switch l.src[l.pos] {
		case ' ', '\t', '\r', '\v', '\f':
			l.pos++
		case '\n':
			l.line++
			l.pos++
		default:
			// Unicode whitespace shows up in minified and copy-pasted source.
			r, size := utf8.DecodeRuneInString(l.src[l.pos:])
			if r == bomRune || r == 0x00A0 || r == 0x2028 || r == 0x2029 ||
				(r >= 0x2000 && r <= 0x200A) || r == 0x3000 {
				if r == 0x2028 || r == 0x2029 {
					l.line++
				}
				l.pos += size
				continue
			}
			return
		}
	}
}

func (l *lexer) peekAt(n int) byte {
	if l.pos+n < len(l.src) {
		return l.src[l.pos+n]
	}
	return 0
}

// regexAllowed decides whether a '/' starts a regular expression, using the
// standard heuristic: a regex cannot follow something that ends a value.
func (l *lexer) regexAllowed() bool {
	switch l.prev.kind {
	case tEOF:
		return true
	case tNumber, tString, tTemplate, tRegex:
		return false
	case tIdent:
		// A keyword is not a value, so a regex can follow it.
		switch l.prev.val {
		case "return", "typeof", "instanceof", "in", "of", "new", "delete", "void",
			"throw", "case", "do", "else", "yield", "await":
			return true
		}
		return false
	case tPunct:
		switch l.prev.val {
		case ")", "]", "}", "+", "-":
			// ')' and ']' usually close a value. '}' is ambiguous (block vs
			// object literal) and both '+' and '-' may be postfix; treating
			// them as value-ending costs at most a missed regex, which only
			// matters if that regex contains a quote or a brace.
			return false
		}
		return true
	}
	return true
}

func (l *lexer) lexRegex() (token, error) {
	start, startLine := l.pos, l.line
	l.pos++ // opening /
	inClass := false
	for {
		if l.pos >= len(l.src) {
			return token{}, l.errf(startLine, "unterminated regular expression")
		}
		switch c := l.src[l.pos]; c {
		case '\\':
			l.pos += 2
			continue
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				l.pos++
				for l.pos < len(l.src) && isIdentPart(rune(l.src[l.pos])) {
					l.pos++
				}
				return token{kind: tRegex, text: l.src[start:l.pos], line: startLine, off: start}, nil
			}
		case '\n':
			return token{}, l.errf(startLine, "unterminated regular expression")
		}
		l.pos++
	}
}

func (l *lexer) lexString(quote byte) (token, error) {
	start, startLine := l.pos, l.line
	l.pos++
	var sb strings.Builder
	for {
		if l.pos >= len(l.src) {
			return token{}, l.errf(startLine, "unterminated string")
		}
		c := l.src[l.pos]
		switch c {
		case quote:
			l.pos++
			return token{kind: tString, text: l.src[start:l.pos], val: sb.String(), line: startLine, off: start}, nil
		case '\n':
			return token{}, l.errf(startLine, "unterminated string")
		case '\\':
			l.pos++
			if err := l.readEscape(&sb); err != nil {
				return token{}, err
			}
		default:
			sb.WriteByte(c)
			l.pos++
		}
	}
}

func (l *lexer) lexTemplate() (token, error) {
	start, startLine := l.pos, l.line
	l.pos++ // opening backtick
	var sb strings.Builder
	subst := false
	for {
		if l.pos >= len(l.src) {
			return token{}, l.errf(startLine, "unterminated template literal")
		}
		c := l.src[l.pos]
		switch {
		case c == '`':
			l.pos++
			return token{
				kind: tTemplate, text: l.src[start:l.pos], val: sb.String(),
				line: startLine, off: start, hasSubst: subst,
			}, nil

		case c == '\\':
			l.pos++
			if err := l.readEscape(&sb); err != nil {
				return token{}, err
			}

		case c == '$' && l.peekAt(1) == '{':
			// A substitution can hold anything, nested templates included, so
			// it is skipped with a real balanced scan rather than a search for
			// the closing brace.
			subst = true
			l.pos += 2
			if err := l.skipSubstitution(startLine); err != nil {
				return token{}, err
			}

		default:
			if c == '\n' {
				l.line++
			}
			sb.WriteByte(c)
			l.pos++
		}
	}
}

// skipSubstitution consumes a template substitution up to its closing brace,
// having already consumed "${". The body is tokenised with next() so a regex
// literal (including one that contains a quote) is not read as a string.
func (l *lexer) skipSubstitution(openedAt int) error {
	saved := l.prev
	defer func() { l.prev = saved }()
	// After "${", a regex is allowed the same way it is after "{".
	l.prev = token{kind: tPunct, val: "{"}
	depth := 1
	for {
		tok, err := l.next()
		if err != nil {
			return err
		}
		if tok.kind == tEOF {
			return l.errf(openedAt, "unterminated template substitution")
		}
		if tok.kind != tPunct {
			continue
		}
		switch tok.val {
		case "{":
			depth++
		case "}":
			depth--
			if depth == 0 {
				return nil
			}
		}
	}
}

func (l *lexer) readEscape(sb *strings.Builder) error {
	if l.pos >= len(l.src) {
		return l.errf(l.line, "unterminated escape sequence")
	}
	c := l.src[l.pos]
	l.pos++
	switch c {
	case 'n':
		sb.WriteByte('\n')
	case 't':
		sb.WriteByte('\t')
	case 'r':
		sb.WriteByte('\r')
	case 'b':
		sb.WriteByte('\b')
	case 'f':
		sb.WriteByte('\f')
	case 'v':
		sb.WriteByte('\v')
	case '0':
		if l.pos < len(l.src) && isDigit(l.src[l.pos]) {
			return l.errf(l.line, "legacy octal escape")
		}
		sb.WriteByte(0)
	case '\n':
		l.line++ // a line continuation contributes nothing
	case '\r':
		if l.pos < len(l.src) && l.src[l.pos] == '\n' {
			l.pos++
		}
		l.line++
	case 'x':
		if l.pos+2 > len(l.src) {
			return l.errf(l.line, "truncated \\x escape")
		}
		n, err := parseHex(l.src[l.pos : l.pos+2])
		if err != nil {
			return l.errf(l.line, "invalid \\x escape")
		}
		l.pos += 2
		sb.WriteRune(rune(n))
	case 'u':
		r, err := l.readUnicodeEscape()
		if err != nil {
			return err
		}
		// A lone high surrogate is only meaningful paired with the next one.
		if utf16.IsSurrogate(r) && strings.HasPrefix(l.src[l.pos:], `\u`) {
			save := l.pos
			l.pos += 2
			if r2, err := l.readUnicodeEscape(); err == nil {
				if combined := utf16.DecodeRune(r, r2); combined != utf8.RuneError {
					sb.WriteRune(combined)
					return nil
				}
			}
			l.pos = save
		}
		sb.WriteRune(r)
	default:
		sb.WriteByte(c)
	}
	return nil
}

func (l *lexer) readUnicodeEscape() (rune, error) {
	if l.pos < len(l.src) && l.src[l.pos] == '{' {
		end := strings.IndexByte(l.src[l.pos:], '}')
		if end < 0 {
			return 0, l.errf(l.line, "unterminated \\u{...} escape")
		}
		n, err := parseHex(l.src[l.pos+1 : l.pos+end])
		if err != nil {
			return 0, l.errf(l.line, "invalid \\u{...} escape")
		}
		l.pos += end + 1
		return rune(n), nil
	}
	if l.pos+4 > len(l.src) {
		return 0, l.errf(l.line, "truncated \\u escape")
	}
	n, err := parseHex(l.src[l.pos : l.pos+4])
	if err != nil {
		return 0, l.errf(l.line, "invalid \\u escape")
	}
	l.pos += 4
	return rune(n), nil
}

func parseHex(s string) (int, error) {
	if s == "" {
		return 0, fmt.Errorf("empty")
	}
	n := 0
	for i := 0; i < len(s); i++ {
		var d int
		switch c := s[i]; {
		case c >= '0' && c <= '9':
			d = int(c - '0')
		case c >= 'a' && c <= 'f':
			d = int(c-'a') + 10
		case c >= 'A' && c <= 'F':
			d = int(c-'A') + 10
		default:
			return 0, fmt.Errorf("not hex")
		}
		n = n*16 + d
	}
	return n, nil
}

func (l *lexer) lexNumber() (token, error) {
	start, startLine := l.pos, l.line
	src := l.src

	if src[l.pos] == '0' && l.pos+1 < len(src) {
		switch src[l.pos+1] {
		case 'x', 'X', 'o', 'O', 'b', 'B':
			l.pos += 2
			for l.pos < len(src) && (isHexDigit(src[l.pos]) || src[l.pos] == '_') {
				l.pos++
			}
			return l.finishNumber(start, startLine)
		}
	}

	for l.pos < len(src) && (isDigit(src[l.pos]) || src[l.pos] == '_') {
		l.pos++
	}
	if l.pos < len(src) && src[l.pos] == '.' {
		l.pos++
		for l.pos < len(src) && (isDigit(src[l.pos]) || src[l.pos] == '_') {
			l.pos++
		}
	}
	if l.pos < len(src) && (src[l.pos] == 'e' || src[l.pos] == 'E') {
		next := l.pos + 1
		if next < len(src) && (src[next] == '+' || src[next] == '-') {
			next++
		}
		if next < len(src) && isDigit(src[next]) {
			l.pos = next
			for l.pos < len(src) && isDigit(src[l.pos]) {
				l.pos++
			}
		}
	}
	return l.finishNumber(start, startLine)
}

func (l *lexer) finishNumber(start, startLine int) (token, error) {
	if l.pos < len(l.src) && l.src[l.pos] == 'n' {
		l.pos++ // BigInt; the value parser rejects it
	}
	return token{kind: tNumber, text: l.src[start:l.pos], val: l.src[start:l.pos], line: startLine, off: start}, nil
}

func (l *lexer) lexIdent() token {
	start, startLine := l.pos, l.line
	for l.pos < len(l.src) {
		r, size := utf8.DecodeRuneInString(l.src[l.pos:])
		if !isIdentPart(r) {
			break
		}
		l.pos += size
	}
	text := l.src[start:l.pos]
	return token{kind: tIdent, text: text, val: text, line: startLine, off: start}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func isHexDigit(c byte) bool {
	return isDigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

func isIdentStart(r rune) bool {
	return r == '_' || r == '$' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r >= utf8.RuneSelf
}

func isIdentPart(r rune) bool {
	return isIdentStart(r) || (r >= '0' && r <= '9')
}
