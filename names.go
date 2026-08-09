package alacris

import (
	"fmt"
	"strings"
)

// reservedTagNames are the hyphenated SVG and MathML names that
// customElements.define rejects.
var reservedTagNames = map[string]struct{}{
	"annotation-xml":   {},
	"color-profile":    {},
	"font-face":        {},
	"font-face-src":    {},
	"font-face-uri":    {},
	"font-face-format": {},
	"font-face-name":   {},
	"missing-glyph":    {},
}

// ValidTagName reports whether name is a valid custom element name — the same
// rule customElements.define enforces, so a typo fails at render time on the
// server rather than silently rendering an unknown element in the browser.
func ValidTagName(name string) error {
	if name == "" {
		return fmt.Errorf("alacris: empty tag name")
	}
	if _, bad := reservedTagNames[name]; bad {
		return fmt.Errorf("alacris: %q is a reserved element name", name)
	}
	if c := name[0]; c < 'a' || c > 'z' {
		return fmt.Errorf("alacris: tag name %q must start with a lowercase ASCII letter", name)
	}
	if !strings.Contains(name, "-") {
		return fmt.Errorf("alacris: tag name %q must contain a hyphen", name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-', r == '.', r == '_':
		case r >= 0x80:
			// The spec's PCENChar set is a long list of Unicode ranges; every
			// character it permits is non-ASCII, so allow them wholesale.
		case r >= 'A' && r <= 'Z':
			return fmt.Errorf("alacris: tag name %q cannot contain uppercase letters", name)
		default:
			return fmt.Errorf("alacris: tag name %q contains an invalid character %q", name, r)
		}
	}
	return nil
}

// ValidAttrName reports whether name can be written into an HTML start tag.
//
// An attribute name is structure, not content: it goes into the tag verbatim,
// because there is no escaping that would leave it meaning the same thing. So
// this is an allowlist rather than a list of characters to avoid — a name that
// is not recognisably a name is refused, and the caller finds out at render
// rather than the browser finding out at parse.
//
// A leading '-' is permitted: AttrName produces one for a prop whose name
// begins with a capital, matching define.js, and the element really is
// listening for that attribute.
func ValidAttrName(name string) error {
	if name == "" {
		return fmt.Errorf("alacris: empty attribute name")
	}
	for i, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r == ':':
		case r == '-':
			// Allowed anywhere, including first — see above.
		case i > 0 && (r >= '0' && r <= '9' || r == '.'):
		default:
			if i == 0 {
				return fmt.Errorf("alacris: attribute name %q must start with a letter, '_', ':' or '-'", name)
			}
			return fmt.Errorf("alacris: attribute name %q contains an invalid character %q", name, r)
		}
	}
	return nil
}

// rawTextElements do not escape their content the way every other element
// does, so markup written inside one is not text — it is script, or style, or
// something the parser treats specially.
var rawTextElements = map[string]struct{}{
	"script": {}, "style": {}, "textarea": {}, "title": {},
	"iframe": {}, "noscript": {}, "noembed": {}, "noframes": {}, "xmp": {},
	"plaintext": {},
}

// ValidElementName reports whether name can be written as a tag.
//
// Used for the wrapper a slot's content is placed in. Raw-text elements are
// refused outright: a slot wrapper has no business being one, and content that
// is escaped everywhere else would not be escaped inside it.
func ValidElementName(name string) error {
	if name == "" {
		return fmt.Errorf("alacris: empty element name")
	}
	if c := name[0]; !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') {
		return fmt.Errorf("alacris: element name %q must start with a letter", name)
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.':
		default:
			return fmt.Errorf("alacris: element name %q contains an invalid character %q", name, r)
		}
	}
	if _, raw := rawTextElements[strings.ToLower(name)]; raw {
		return fmt.Errorf("alacris: <%s> does not escape its content, so it cannot wrap slot content", name)
	}
	return nil
}

// sanitizeDeclaration checks one inline CSS declaration.
//
// The rules are deliberately narrower than CSS: everything that could end the
// declaration, end the attribute, or open a comment is refused. Values keep
// parentheses and commas, because colour and length functions are the whole
// point of setting a custom property from the server.
func sanitizeDeclaration(property, value string) (string, string, error) {
	if err := validCSSProperty(property); err != nil {
		return "", "", err
	}
	if err := validCSSValue(value); err != nil {
		return "", "", fmt.Errorf("value for %q: %w", property, err)
	}
	return property, strings.TrimSpace(value), nil
}

func validCSSProperty(p string) error {
	name := p
	custom := strings.HasPrefix(p, "--")
	if custom {
		name = p[2:]
	} else {
		name = strings.TrimPrefix(name, "-") // vendor prefix
	}
	if name == "" {
		return fmt.Errorf("alacris: empty CSS property name")
	}
	if !custom {
		if c := name[0]; !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') {
			return fmt.Errorf("alacris: CSS property %q must start with a letter", p)
		}
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return fmt.Errorf("alacris: CSS property %q contains an invalid character %q", p, r)
		}
	}
	return nil
}

func validCSSValue(v string) error {
	for _, r := range v {
		switch r {
		case ';', '{', '}', '<', '>', '"', '\'', '\\', '@':
			return fmt.Errorf("alacris: character %q is not allowed in an inline CSS value", r)
		}
		if r < 0x20 && r != '\t' || r == 0x7f {
			return fmt.Errorf("alacris: control character in CSS value")
		}
	}
	if strings.Contains(v, "/*") || strings.Contains(v, "*/") {
		return fmt.Errorf("alacris: comment markers are not allowed in an inline CSS value")
	}
	lower := strings.ToLower(v)
	// url() survives because it is legitimately useful; the schemes that turn
	// a style into script do not.
	for _, bad := range []string{"javascript:", "expression(", "-moz-binding"} {
		if strings.Contains(lower, bad) {
			return fmt.Errorf("alacris: %q is not allowed in an inline CSS value", bad)
		}
	}
	return nil
}
