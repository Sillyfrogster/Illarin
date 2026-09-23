package jscode

import "bytes"

type tokenType uint8

const (
	punct tokenType = iota + 1
	name
	number
	regex
	// text is a quoted string, or a template that substitutes nothing.
	text
	// template is a piece of a template that substitutes code.
	template
	// unclosed is a string or template that never closes.
	unclosed
)

type token struct {
	tokenType  tokenType
	start, end int
}

// lexer walks the tokens of JavaScript or TypeScript source from any point where code starts.
type lexer struct {
	source []byte
	at     int
	last   token
	// braces counts the open braces of code, and substitutions holds the count at which each open ${ closes.
	braces        int
	substitutions []int
}

// beforeExpression holds the words after which a slash opens a regular expression rather than dividing.
var beforeExpression = map[string]bool{
	"return": true, "typeof": true, "instanceof": true, "in": true, "of": true, "new": true, "delete": true,
	"void": true, "throw": true, "case": true, "do": true, "else": true, "yield": true, "await": true,
}

func (l *lexer) next() (token, bool) {
	l.skipSpace()
	if l.at >= len(l.source) {
		return token{}, false
	}
	l.last = l.read()
	return l.last, true
}

func (l *lexer) read() token {
	start := l.at
	c := l.source[start]
	switch {
	case c == '"' || c == '\'':
		end, closed := quotedEnd(l.source, start)
		l.at = end
		if !closed {
			return token{tokenType: unclosed, start: start, end: end}
		}
		return token{tokenType: text, start: start, end: end}
	case c == '`':
		return l.template(start, true)
	case c == '}' && l.closesSubstitution():
		l.substitutions = l.substitutions[:len(l.substitutions)-1]
		return l.template(start, false)
	case isNameByte(c):
		l.at = wordEnd(l.source, start)
		return token{tokenType: name, start: start, end: l.at}
	case isDigit(c) || (c == '.' && start+1 < len(l.source) && isDigit(l.source[start+1])):
		l.at = numberEnd(l.source, start)
		return token{tokenType: number, start: start, end: l.at}
	case c == '/' && l.expectsExpression():
		if end, ok := regexEnd(l.source, start); ok {
			l.at = end
			return token{tokenType: regex, start: start, end: end}
		}
	case l.startsWith("...") || (l.startsWith("?.") && !(start+2 < len(l.source) && isDigit(l.source[start+2]))):
		l.at = start + 2
		if c == '.' {
			l.at++
		}
		return token{tokenType: punct, start: start, end: l.at}
	}
	l.at = start + 1
	switch c {
	case '{':
		l.braces++
	case '}':
		l.braces--
	}
	return token{tokenType: punct, start: start, end: l.at}
}

// expectsExpression says whether the last token leaves room for a value, where a slash starts a regular expression.
func (l *lexer) expectsExpression() bool {
	last := l.last
	switch last.tokenType {
	case 0:
		return true
	case name:
		return beforeExpression[last.spelled(l.source)]
	case template:
		return l.source[last.end-1] == '{'
	case punct:
		c := l.source[last.start]
		return c != ')' && c != ']'
	default:
		return false
	}
}

func (l *lexer) closesSubstitution() bool {
	return len(l.substitutions) > 0 && l.braces == l.substitutions[len(l.substitutions)-1]
}

// template reads template text from start to its closing backtick or its next substitution.
func (l *lexer) template(start int, whole bool) token {
	for l.at = start + 1; l.at < len(l.source); l.at++ {
		switch l.source[l.at] {
		case '\\':
			l.at++
		case '`':
			l.at++
			if whole {
				return token{tokenType: text, start: start, end: l.at}
			}
			return token{tokenType: template, start: start, end: l.at}
		case '$':
			if l.at+1 < len(l.source) && l.source[l.at+1] == '{' {
				l.at += 2
				l.substitutions = append(l.substitutions, l.braces)
				return token{tokenType: template, start: start, end: l.at}
			}
		}
	}
	l.at = len(l.source)
	return token{tokenType: unclosed, start: start, end: l.at}
}

// skipSpace passes over white space and comments.
func (l *lexer) skipSpace() {
	for l.at < len(l.source) {
		switch {
		case isSpace(l.source[l.at]):
			l.at++
		case l.startsWith("//"):
			for l.at < len(l.source) && l.source[l.at] != '\n' {
				l.at++
			}
		case l.startsWith("/*"):
			end := bytes.Index(l.source[l.at+2:], []byte("*/"))
			if end < 0 {
				l.at = len(l.source)
				return
			}
			l.at += 2 + end + 2
		default:
			return
		}
	}
}

func (l *lexer) startsWith(prefix string) bool {
	return bytes.HasPrefix(l.source[l.at:], []byte(prefix))
}

// quotedEnd finds the end of the string opened at start, stopping at a line break when the string never closes.
func quotedEnd(source []byte, start int) (int, bool) {
	quote := source[start]
	for at := start + 1; at < len(source); at++ {
		switch source[at] {
		case '\\':
			at++
		case quote:
			return at + 1, true
		case '\n', '\r':
			return at, false
		}
	}
	return len(source), false
}

// regexEnd finds the end of the regular expression opened at start, which must close on its own line.
func regexEnd(source []byte, start int) (int, bool) {
	inClass := false
	for at := start + 1; at < len(source); at++ {
		switch source[at] {
		case '\\':
			at++
		case '[':
			inClass = true
		case ']':
			inClass = false
		case '/':
			if !inClass {
				return wordEnd(source, at+1), true
			}
		case '\n', '\r':
			return 0, false
		}
	}
	return 0, false
}

// numberEnd reads past a number in any of its spellings, exponent signs included.
func numberEnd(source []byte, start int) int {
	at := start
	for at < len(source) {
		c := source[at]
		signed := (c == '+' || c == '-') && at > start && (source[at-1] == 'e' || source[at-1] == 'E')
		if !isNameByte(c) && !isDigit(c) && c != '.' && !signed {
			return at
		}
		at++
	}
	return at
}

func wordEnd(source []byte, at int) int {
	for at < len(source) && (isNameByte(source[at]) || isDigit(source[at])) {
		at++
	}
	return at
}

func (t token) is(source []byte, spelled string) bool {
	return t.tokenType == punct && string(source[t.start:t.end]) == spelled
}

// isDot says whether the token reaches into a member, optionally or not.
func (t token) isDot(source []byte) bool { return t.is(source, ".") || t.is(source, "?.") }

func (t token) spelled(source []byte) string { return string(source[t.start:t.end]) }

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r' || c == '\v' || c == '\f'
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

// isNameByte accepts every byte past ASCII, so names written in any script read whole.
func isNameByte(c byte) bool {
	return c == '_' || c == '$' || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c >= 0x80
}
