package jscode

import (
	"bytes"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// unquote spells out a quoted string, or a template that substitutes nothing, with its escapes read.
func unquote(quoted []byte) (string, bool) {
	if len(quoted) < 2 || quoted[len(quoted)-1] != quoted[0] {
		return "", false
	}
	body := quoted[1 : len(quoted)-1]
	if bytes.IndexByte(body, '\\') < 0 {
		return string(body), true
	}
	var spelled strings.Builder
	for at := 0; at < len(body); at++ {
		if body[at] != '\\' {
			spelled.WriteByte(body[at])
			continue
		}
		at++
		if at >= len(body) {
			return "", false
		}
		read, ok := escape(body, at)
		if !ok {
			return "", false
		}
		spelled.WriteString(read.text)
		at += read.length - 1
	}
	return spelled.String(), true
}

type escaped struct {
	text   string
	length int
}

var simpleEscapes = map[byte]string{'n': "\n", 't': "\t", 'r': "\r", 'b': "\b", 'f': "\f", 'v': "\v", '0': "\x00"}

// escape reads the escape whose first byte after the backslash is at, and how many bytes it spans.
func escape(body []byte, at int) (escaped, bool) {
	c := body[at]
	if text, ok := simpleEscapes[c]; ok {
		return escaped{text: text, length: 1}, true
	}
	switch c {
	case '\n':
		return escaped{length: 1}, true
	case '\r':
		if at+1 < len(body) && body[at+1] == '\n' {
			return escaped{length: 2}, true
		}
		return escaped{length: 1}, true
	case 'x':
		code, ok := hexCode(body, at+1, at+3)
		return escaped{text: string(rune(code)), length: 3}, ok
	case 'u':
		return unicodeEscape(body, at)
	default:
		return escaped{text: string(c), length: 1}, true
	}
}

// unicodeEscape reads a unicode escape of four hex digits or of digits in braces, joining a surrogate pair written as two.
func unicodeEscape(body []byte, at int) (escaped, bool) {
	code, length, ok := unicodeCode(body, at)
	if !ok {
		return escaped{}, false
	}
	if utf16.IsSurrogate(rune(code)) && at+length+1 < len(body) && body[at+length] == '\\' && body[at+length+1] == 'u' {
		low, lowLength, lowOK := unicodeCode(body, at+length+1)
		if joined := utf16.DecodeRune(rune(code), rune(low)); lowOK && joined != utf8.RuneError {
			return escaped{text: string(joined), length: length + 1 + lowLength}, true
		}
	}
	if code > utf8.MaxRune {
		return escaped{}, false
	}
	return escaped{text: string(rune(code)), length: length}, true
}

// unicodeCode reads the code point of the unicode escape whose u is at, and how many bytes it spans from the u.
func unicodeCode(body []byte, at int) (uint64, int, bool) {
	if at+1 < len(body) && body[at+1] == '{' {
		end := bytes.IndexByte(body[at+2:], '}')
		if end < 1 {
			return 0, 0, false
		}
		code, ok := hexCode(body, at+2, at+2+end)
		return code, end + 3, ok
	}
	code, ok := hexCode(body, at+1, at+5)
	return code, 5, ok
}

func hexCode(body []byte, from, to int) (uint64, bool) {
	if to > len(body) {
		return 0, false
	}
	code, err := strconv.ParseUint(string(body[from:to]), 16, 32)
	return code, err == nil
}
