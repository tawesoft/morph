package internal

import (
    "fmt"
    "strings"
    "unicode"
)

// TokenReplacer can Replace "$N", "$name", "$N.M", "$name.M" tokens with
// values for a decimal digit N and M and a named value "name".
type TokenReplacer struct {
    ByIndex      func(index int) (string, bool)
    ByName       func(name string) (string, bool)
    TupleByIndex func(index int, subidx int) (string, bool)
    TupleByName  func(name string, subidx int) (string, bool)
}

func (t TokenReplacer) Replace(in string) (string, error) {
    var out strings.Builder
    for i := 0; i < len(in); i++ { // bytewise is fine
        c := in[i]
        if (c == '\'') || (c == '"') || (c == '`') {
            l, err := t.consumeStringLiteral(c, in[i:])
            if err != nil { return "", err }
            out.WriteString(in[i:i+l])
            i += l
        } else if c == '$' {
            value, l, err := t.consumeIdent(in[i:])
            if err != nil { return "", err }
            out.WriteString(value)
            i += l
        } else {
            out.WriteByte(c)
        }
    }
    return out.String(), nil
}

// consumeIdent returns advance, error, and calls the appropriate
// TokenReplacer functions on the identifier.
func (t TokenReplacer) consumeIdent(in string) (string, int, error) {
    var err error
    if len(in) < 0 {
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
            err = fmt.Errorf("couldn't find $%d", idx)
        }
        return s, advance, err
    } else if name, advance, ok := t.consumeName(in); ok {
        // either $name or $name.N
        in = in[advance:]
        if t.peekDot(in) {
            in = in[1:]
            if idx, advance2, ok := t.consumeNumber(in); ok {
                // it's $N.N
                s, ok := t.TupleByName(name, idx)
                if !ok {
                    err = fmt.Errorf("couldn't find $%s.%d", name, idx)
                }
                return s, advance + advance2 + 1, err
            }
        }

        // just $name
        s, ok := t.ByName(name)
        if !ok {
            err = fmt.Errorf("couldn't find $%s", name)
        }
        return s, advance, err
    }

    return "", 0, fmt.Errorf("invalid identifier")
}

// peekDot returns true if next input is a "."
func (t TokenReplacer) peekDot(in string) bool {
    // TODO support arg indexes greater than 9 (would anybody ever need one!?)
    return (len(in) >= 1) && (in[0] == '.')
}

// consumeNumber returns value, advance, ok.
func (t TokenReplacer) consumeNumber(in string) (int, int, bool) {
    if len(in) < 1 { return 0, 0, false }
    if !IsAsciiNumber(rune(in[0])) { return 0, 0, false }
    // TODO support arg indexes greater than 9 (would anybody ever need one!?)
    return int(in[0] - '0'), 1, true
}

// consumeNumber returns name, advance, ok
func (t TokenReplacer) consumeName(in string) (string, int, bool) {
    var i int
    for i = 0; i < len(in); i++ { // bytewise is fine
        c := in[i]
        ok := IsGoIdentLetter(rune(c)) ||
            ((i > 0) && unicode.IsNumber(rune(c)))
        if !ok { break }
    }
    if i < 1 { return "", 0, false }
    return in[0:i], i, true
}

func (t TokenReplacer) consumeStringLiteral(delim byte, in string) (int, error) {
    if len(in) < 1 {
        return 0, fmt.Errorf("string not terminated")
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

    return 0, fmt.Errorf("string %q not terminated", in)
}
