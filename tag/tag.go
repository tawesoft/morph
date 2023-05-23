package tag

// This file contains code Copyright 2009 The Go Authors. All rights reserved.
//
// This includes some internals from strconv.
//
// This also includes some implementation from reflect, to avoid a heavy
// and sometimes unportable import on the reflect package for what is merely
// some string processing.

import (
    "strconv"
    "unicode/utf8"
)

// Quote is like [strconv.Quote], but uses backticks instead of double
// quotes.
func Quote(s string) string {
	return quoteWith(s, '`', false, false)
}

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/quote.go;l=23
func quoteWith(s string, quote byte, ASCIIonly, graphicOnly bool) string {
	return string(appendQuotedWith(make([]byte, 0, 3*len(s)/2), s, quote, ASCIIonly, graphicOnly))
}

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/quote.go;l=13
const (
	lowerhex = "0123456789abcdef"
	upperhex = "0123456789ABCDEF"
)

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/isprint.go;l=701
var isGraphic = []uint16{
	0x00a0,
	0x1680,
	0x2000,
	0x2001,
	0x2002,
	0x2003,
	0x2004,
	0x2005,
	0x2006,
	0x2007,
	0x2008,
	0x2009,
	0x200a,
	0x202f,
	0x205f,
	0x3000,
}

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/quote.go;l=31
func appendQuotedWith(buf []byte, s string, quote byte, ASCIIonly, graphicOnly bool) []byte {
	// Often called with big strings, so preallocate. If there's quoting,
	// this is conservative but still helps a lot.
	if cap(buf)-len(buf) < len(s) {
		nBuf := make([]byte, len(buf), len(buf)+1+len(s)+1)
		copy(nBuf, buf)
		buf = nBuf
	}
	buf = append(buf, quote)
	for width := 0; len(s) > 0; s = s[width:] {
		r := rune(s[0])
		width = 1
		if r >= utf8.RuneSelf {
			r, width = utf8.DecodeRuneInString(s)
		}
		if width == 1 && r == utf8.RuneError {
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[s[0]>>4])
			buf = append(buf, lowerhex[s[0]&0xF])
			continue
		}
		buf = appendEscapedRune(buf, r, quote, ASCIIonly, graphicOnly)
	}
	buf = append(buf, quote)
	return buf
}

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/quote.go;l=68
func appendEscapedRune(buf []byte, r rune, quote byte, ASCIIonly, graphicOnly bool) []byte {
	var runeTmp [utf8.UTFMax]byte
	if r == rune(quote) || r == '\\' { // always backslashed
		buf = append(buf, '\\')
		buf = append(buf, byte(r))
		return buf
	}
	if ASCIIonly {
		if r < utf8.RuneSelf && strconv.IsPrint(r) {
			buf = append(buf, byte(r))
			return buf
		}
	} else if strconv.IsPrint(r) || graphicOnly && isInGraphicList(r) {
		n := utf8.EncodeRune(runeTmp[:], r)
		buf = append(buf, runeTmp[:n]...)
		return buf
	}
	switch r {
	case '\a':
		buf = append(buf, `\a`...)
	case '\b':
		buf = append(buf, `\b`...)
	case '\f':
		buf = append(buf, `\f`...)
	case '\n':
		buf = append(buf, `\n`...)
	case '\r':
		buf = append(buf, `\r`...)
	case '\t':
		buf = append(buf, `\t`...)
	case '\v':
		buf = append(buf, `\v`...)
	default:
		switch {
		case r < ' ' || r == 0x7f:
			buf = append(buf, `\x`...)
			buf = append(buf, lowerhex[byte(r)>>4])
			buf = append(buf, lowerhex[byte(r)&0xF])
		case !utf8.ValidRune(r):
			r = 0xFFFD
			fallthrough
		case r < 0x10000:
			buf = append(buf, `\u`...)
			for s := 12; s >= 0; s -= 4 {
				buf = append(buf, lowerhex[r>>uint(s)&0xF])
			}
		default:
			buf = append(buf, `\U`...)
			for s := 28; s >= 0; s -= 4 {
				buf = append(buf, lowerhex[r>>uint(s)&0xF])
			}
		}
	}
	return buf
}

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/quote.go;l=593
func isInGraphicList(r rune) bool {
	// We know r must fit in 16 bits - see makeisprint.go.
	if r > 0xFFFF {
		return false
	}
	rr := uint16(r)
	i := bsearch16(isGraphic, rr)
	return i < len(isGraphic) && rr == isGraphic[i]
}

// from https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/strconv/quote.go;l=13
func bsearch16(a []uint16, x uint16) int {
	i, j := 0, len(a)
	for i < j {
		h := i + (j-i)>>1
		if a[h] < x {
			i = h + 1
		} else {
			j = h
		}
	}
	return i
}

// Lookup looks up values in a struct tag, which by convention is
// a sequence of key:"value" pairs, optionally separated by whitespace.
// See [reflect.StructTag].
//
// Note that, unlike the Go parser and reflect package, struct tag strings in
// morph are not enclosed with a quote pair like a Go string literal. The
// input to this function is unquoted.
//
// It returns the value associated with key in the tag string. If the key
// is present in the tag the value (which may be empty) is returned. Otherwise,
// the returned value will be the empty string. The ok return value reports
// whether the value was explicitly set in the tag string. If the tag does not
// have the conventional format, the value returned by Lookup is unspecified.
func Lookup(tag, key string) (value string, ok bool) {
	// This function is mostly a copy of the [reflect.StructTag.Lookup] method in
	// [reflect/type.go]

	if len(tag) == 0 {
		return "", false
	}
	// tag = strconv.Quote(tag)

	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		if key == name {
			value, err := strconv.Unquote(qvalue)
			if err != nil {
				break
			}
			return value, true
		}
	}
	return "", false
}

// NextPair returns the next key, value, and remaining tag in a struct tag
// string.
//
// Note that, unlike the Go parser and reflect package, struct tag strings in
// morph are not enclosed with a quote pair like a Go string literal. The
// input to this function is unquoted.
//
// The ok return value reports whether the value was explicitly set in the tag
// string. If the tag does not have the conventional format, the value returned
// by is unspecified.
func NextPair(tag string) (key, value, rest string, ok bool) {
	// This function is based on the [reflect.StructTag.Lookup] method in
	// [reflect/type.go](https://cs.opensource.google/go/go/+/refs/tags/go1.20.4:src/reflect/type.go;l=1201).

	if len(tag) == 0 {
		return "", "", "", false
	}

	for tag != "" {
		// Skip leading space.
		i := 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		if tag == "" {
			break
		}

		// Scan to colon. A space, a quote or a control character is a syntax error.
		// Strictly speaking, control chars include the range [0x7f, 0x9f], not just
		// [0x00, 0x1f], but in practice, we ignore the multi-byte control characters
		// as it is simpler to inspect the tag's bytes than the tag's runes.
		i = 0
		for i < len(tag) && tag[i] > ' ' && tag[i] != ':' && tag[i] != '"' && tag[i] != 0x7f {
			i++
		}
		if i == 0 || i+1 >= len(tag) || tag[i] != ':' || tag[i+1] != '"' {
			break
		}
		name := string(tag[:i])
		tag = tag[i+1:]

		// Scan quoted string to find value.
		i = 1
		for i < len(tag) && tag[i] != '"' {
			if tag[i] == '\\' {
				i++
			}
			i++
		}
		if i >= len(tag) {
			break
		}
		qvalue := string(tag[:i+1])
		tag = tag[i+1:]

		// Skip leading space
		i = 0
		for i < len(tag) && tag[i] == ' ' {
			i++
		}
		tag = tag[i:]
		// Skip trailing space.
		i = len(tag) - 1
		for i > 0 && tag[i] == ' ' {
			i--
		}
		if i < len(tag)-1 {
			tag = tag[:i]
		}

		value, err := strconv.Unquote(qvalue)
		if err != nil {
			break
		}
		return name, value, tag, true
	}
	return "", "", "", false
}
