package internal

import (
    "fmt"
    "strings"
)

// TokenReplacer can replace $-tokens with values.
//
// In cases where a $-token is ambiguous, use parentheses e.g. "$(foo)".
type TokenReplacer struct {
    // Single converts the $-token "$" when not preceded by or following a dot.
    Single func() (string, bool)

    // ByIndex converts the $-token "$N" for some decimal "N".
    ByIndex func(index int) (string, bool)

    // ByName converts the $-token "$name" for some string "name".
    ByName func(name string) (string, bool)

    // TupleByIndex converts the $-token "$N.M" for some decimals "N" and "M".
    TupleByIndex func(index int, subidx int) (string, bool)

    // TupleByName converts the $-token "$name.M" for some string "name" and
    // some decimal "M".
    TupleByName func(name string, subidx int) (string, bool)

    // FieldByName converts the $-token "$name.field" for some struct value
    // "name" and some field "field".
    FieldByName func(structValue string, field string) (string, bool)

    // Modifier converts any $-token followed by ".$keyword", passing the
    // resolved token up to that point as an argument.
    Modifier func(keyword string, target string) (string, bool)
}

func (t *TokenReplacer) SetDefaults() {
    if t.Single == nil  {
        t.Single = func() (string, bool) { return "", false }
    }
    if t.ByIndex == nil {
        t.ByIndex = func(int) (string, bool) { return "", false }
    }
    if t.ByName == nil {
        t.ByName = func(string) (string, bool) { return "", false }
    }
    if t.TupleByIndex == nil {
        t.TupleByIndex = func(int, int) (string, bool) { return "", false }
    }
    if t.TupleByName == nil {
        t.TupleByName = func(string, int) (string, bool) { return "", false }
    }
    if t.FieldByName == nil {
        t.FieldByName = func(string, string) (string, bool) { return "", false }
    }
    if t.Modifier == nil {
        t.Modifier = func(string, string) (string, bool) { return "", false }
    }
}

func (t TokenReplacer) Replace(in string) (string, error) {
    esc := func(err error) (string, error) {
        return "", fmt.Errorf("token replacement failure in string %q: %w", in, err)
    }
    var out strings.Builder
    for i := 0; i < len(in); i++ { // byte-wise is fine
        c := in[i]
        if (c == '\'') || (c == '"') || (c == '`') {
            l, err := t.consumeStringLiteral(c, in[i:])
            if err != nil { return esc(err) }
            out.WriteString(in[i:i+l])
            i += l
        } else if c == '$' {
            value, l, err := t.parenthesisedErr(in[i:], t.consumeIdent)
            if err != nil { return esc(err) }
            i += l

            for {
                if t.peekDot(in[i+1:]) {
                    if kw, l2, ok := t.parenthesisedOk(in[i+2:], t.consumeKeyword); ok {
                        if value2, ok := t.Modifier(kw, value); ok {
                            i += l2 + 1
                            value = value2
                            continue
                        }
                    }
                }
                break
            }

            out.WriteString(value)
        } else {
            out.WriteByte(c)
        }
    }
    return out.String(), nil
}

// parenthesisedErr wraps a consuming function to transparently handle
// parentheses e.g. "$(foo).$(bar)" is equivalent to $foo.$bar.
func (t TokenReplacer) parenthesisedErr(
    in string,
    consume func(in string) (string, int, error),
) (string, int, error) {
    if !strings.HasPrefix(in, "$(") { return consume(in) }
    idx := strings.IndexRune(in, ')')
    if idx < 0 {
        return "", 0, fmt.Errorf("unmatched parenthesis")
    }
    s, l, err := consume("$"+in[2:idx])
    return s, l + 2, err
}

// parenthesisedOk wraps a consuming function to transparently handle
// parentheses e.g. "$(foo).$(bar)" is equivalent to $foo.$bar.
func (t TokenReplacer) parenthesisedOk(
    in string,
    consume func(in string) (string, int, bool),
) (string, int, bool) {
    if !strings.HasPrefix(in, "$(") { return consume(in) }
    idx := strings.IndexRune(in, ')')
    if idx < 0 {
        return "", 0, false
    }
    s, l, ok := consume("$"+in[2:idx])
    return s, l + 2, ok
}

// consumeIdent returns replacement, advance, error, by calling the
// appropriate TokenReplacer functions on the identifier.
func (t TokenReplacer) consumeIdent(in string) (string, int, error) {
    var err error
    if (len(in) < 0) || (in[0] != '$') {
        return "", 0, fmt.Errorf("expected identifier start")
    }
    in = in[1:]

    if idx, advance, ok := t.consumeNumber(in); ok {
        // either $N or $N.N
        in = in[advance:]
        if t.peekDot(in) {
            in = in[1:]
            if idx2, advance2, ok := t.consumeNumber(in); ok {
                // it's $N.N
                s, ok := t.TupleByIndex(idx, idx2)
                if !ok {
                    err = fmt.Errorf("couldn't find $%d.%d", idx, idx2)
                }
                return s, advance + advance2 + 1, err
            }
        }

        // just $N
        s, ok := t.ByIndex(idx)
        if !ok {
            err = fmt.Errorf("couldn't find indexed $%d", idx)
        }
        return s, advance, err
    } else if name, advance, ok := t.consumeName(in); ok {
        // either $name or $name.N or $name.field
        in = in[advance:]
        if t.peekDot(in) {
            in = in[1:]
            if field, advance2, ok := t.consumeName(in); ok {
                // it's $name.field
                s, ok := t.FieldByName(name, field)
                if !ok {
                    err = fmt.Errorf("couldn't find named field $%s.$%s", name, field)
                }
                return s, advance + advance2 + 1, err
            }

            if idx, advance2, ok := t.consumeNumber(in); ok {
                // it's $name.N
                s, ok := t.TupleByName(name, idx)
                if !ok {
                    err = fmt.Errorf("couldn't find index $%s.%d", name, idx)
                }
                return s, advance + advance2 + 1, err
            }
        }

        // just $name
        s, ok := t.ByName(name)
        if !ok {
            err = fmt.Errorf("couldn't find named $%s", name)
        }
        return s, advance, err
    } else {
        // Just "$" alone
        if s, ok := t.Single(); ok {
            return s, 0, nil
        }
    }

    return "", 0, fmt.Errorf("invalid identifier %q", in)
}

// peekDot returns true if next input is a "."
func (t TokenReplacer) peekDot(in string) bool {
    return (len(in) >= 1) && (in[0] == '.')
}

// peekEmptyParens returns true if next input is "()"
/*
func (t TokenReplacer) peekEmptyParens(in string) bool {
    return (len(in) >= 2) && (in[0] == '(') && (in[1] == ')')
}
*/

// consumeNumber returns value, advance, ok.
func (t TokenReplacer) consumeNumber(in string) (int, int, bool) {
    if len(in) < 1 { return 0, 0, false }
    if !IsAsciiNumber(rune(in[0])) { return 0, 0, false }
    // TODO support arg indexes greater than 9 (would anybody ever need one!?)
    return int(in[0] - '0'), 1, true
}

// consumeName returns name, advance, ok
func (t TokenReplacer) consumeName(in string) (string, int, bool) {
    var i int
    for i = 0; i < len(in); i++ { // byte-wise is fine
        c := in[i]
        valid := IsGoIdentIdx(rune(c), i)
        if !valid { break }
    }
    if i < 1 { return "", 0, false }
    return in[0:i], i, true
}

// consumeKeyword returns name, advance, found
func (t TokenReplacer) consumeKeyword(in string) (string, int, bool) {
    var i int

    if len(in) == 0 {
        return "", 0, false
    }

    if (len(in) >= 1) && (in[0] != '$') {
        return "", 0, false
    }

    for i = 1; i < len(in); i++ { // byte-wise is fine
        c := in[i]
        valid := IsGoIdentIdx(rune(c), i)
        if !valid { break }
    }

    return in[1:i], i, true
}

var errStringNotTerminated = fmt.Errorf("expected identifier start")

// consumeStringLiteral consumes single-quoted, double-quoted, and
// backtick-quoted Go strings, handling escape sequences in single- and
// double-quoted strings.
func (t TokenReplacer) consumeStringLiteral(delim byte, in string) (int, error) {
    if len(in) < 1 {
        return 0, errStringNotTerminated
    }

    if delim == '`' {
        for i := 1; i < len(in); i++ {
            c := in[i]
            if c == '`' {
                return i + 1, nil
            }
        }
    } else {
        for i := 1; i < len(in); i++ {
            c := in[i]
            if c == '\\' {
                i++
                continue
            } else if c == '\n' {
                break
            } else if c == delim {
                return i + 1, nil
            }
        }
    }

    return 0, errStringNotTerminated
}
