package gen

import (
	"strings"
	"unicode"

	alacris "github.com/bmartel/alacris-go"
)

// attrName is the runtime's own prop-to-attribute conversion, so generated
// code and rendered elements cannot disagree about a name.
func attrName(prop string) string { return alacris.AttrName(prop) }

// initialisms are the words go vet and reviewers expect to see capitalised in
// a Go identifier.
var initialisms = map[string]string{
	"acl": "ACL", "api": "API", "ascii": "ASCII", "cpu": "CPU", "css": "CSS",
	"dns": "DNS", "eof": "EOF", "guid": "GUID", "html": "HTML", "http": "HTTP",
	"https": "HTTPS", "id": "ID", "ip": "IP", "json": "JSON", "lhs": "LHS",
	"qps": "QPS", "ram": "RAM", "rhs": "RHS", "rpc": "RPC", "sla": "SLA",
	"smtp": "SMTP", "sql": "SQL", "ssh": "SSH", "tcp": "TCP", "tls": "TLS",
	"ttl": "TTL", "udp": "UDP", "ui": "UI", "uid": "UID", "uuid": "UUID",
	"uri": "URI", "url": "URL", "utf8": "UTF8", "vm": "VM", "xml": "XML",
	"xsrf": "XSRF", "xss": "XSS",
}

// exportedName turns a tag or prop name into an exported Go identifier:
// "user-card" and "userCard" both become "UserCard", and a recognised
// initialism keeps its conventional casing.
func exportedName(s string) string {
	words := splitWords(s)
	var b strings.Builder
	for _, w := range words {
		if up, ok := initialisms[strings.ToLower(w)]; ok {
			b.WriteString(up)
			continue
		}
		r := []rune(w)
		b.WriteRune(unicode.ToUpper(r[0]))
		if len(r) > 1 {
			b.WriteString(string(r[1:]))
		}
	}
	out := b.String()
	if out == "" {
		return "X"
	}
	if unicode.IsDigit(rune(out[0])) {
		return "X" + out
	}
	return out
}

// splitWords breaks an identifier at separators and at camelCase boundaries.
func splitWords(s string) []string {
	var (
		words []string
		cur   strings.Builder
	)
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	runes := []rune(s)
	for i, r := range runes {
		switch {
		case r == '-' || r == '_' || r == '.' || r == ' ' || r == ':':
			flush()
		case unicode.IsUpper(r):
			// Break before a capital, but keep runs of capitals together so
			// "URLPath" splits as "URL" and "Path" rather than letter by letter.
			prevLower := i > 0 && unicode.IsLower(runes[i-1])
			nextLower := i+1 < len(runes) && unicode.IsLower(runes[i+1])
			if prevLower || (nextLower && cur.Len() > 0) {
				flush()
			}
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	flush()
	return words
}

// fileBase turns a source path into the base name of its generated Go file.
//
// It reduces to a base name itself rather than trusting the caller to have
// done it. Every caller does today, so nothing was ever written outside the
// output directory — but the name of this function is a promise, and a path
// separator surviving into a generated file name would be a bad way to find
// out it was not keeping it.
func fileBase(name string) string {
	name = strings.ReplaceAll(name, `\`, "/")
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		name = name[i+1:]
	}
	if i := strings.LastIndexByte(name, ':'); i >= 0 {
		name = name[i+1:] // a Windows drive letter, or a file:line suffix
	}
	name = strings.TrimSuffix(name, ".js")
	name = strings.TrimSuffix(name, ".mjs")
	name = strings.TrimSuffix(name, ".json")
	var b strings.Builder
	for _, w := range splitWords(name) {
		if b.Len() > 0 {
			b.WriteByte('_')
		}
		b.WriteString(strings.ToLower(w))
	}
	if b.Len() == 0 {
		return "components"
	}
	out := b.String()
	if unicode.IsDigit(rune(out[0])) {
		return "x" + out
	}
	return out
}
