package internal

import (
    "go/format"
    "strings"
    "unicode"

    "github.com/tawesoft/morph/tag"
)

// RewriteSignatureString performs the special '$' replacement in a function
// signature specified as a string.
//
// TODO ignore "$" inside string literal and use tokenReplacer instead
//   note -- pass through other occurences of "$" unchanged
func RewriteSignatureString(sig string, from string, to string) string {
    if strings.HasPrefix(from, "*") { from = from[1:] }
    if strings.HasPrefix(to, "*") { to = to[1:] }

    lower := func(x string) string {
        if len(x) == 0 { return x }
        if len(x) == 1 { strings.ToLower(x) }
        return strings.ToLower(string(x[0])) + x[1:]
    }
    upper := func(x string) string {
        if len(x) == 0 { return x }
        if len(x) == 1 { strings.ToUpper(x) }
        return strings.ToUpper(string(x[0])) + x[1:]
    }

    sig = strings.ReplaceAll(sig, "$FROM", upper(from))
    sig = strings.ReplaceAll(sig, "$From", from)
    sig = strings.ReplaceAll(sig, "$from", lower(from))
    sig = strings.ReplaceAll(sig, "$TO",   upper(to))
    sig = strings.ReplaceAll(sig, "$To",   to)
    sig = strings.ReplaceAll(sig, "$to",   lower(to))
    return sig
}

type Set[X comparable] interface {
    Add(x X)
    Contains(x X) bool
}

type set[X comparable] struct {
    s map[X]struct{}
}

func NewSet[X comparable]() Set[X] {
    return set[X]{
        s: make(map[X]struct{}),
    }
}

func (s set[X]) Add(x X) {
    s.s[x] = struct{}{}
}

func (s set[X]) Contains(x X) bool {
    _, ok := s.s[x]
    return ok
}

func IsAsciiNumber(x rune) bool {
    return (x >= '0') && (x <= '9')
}

func IsGoIdentLetter(x rune) bool {
    return x == '_' || unicode.IsLetter(x)
}

func Must[T any](result T, err error) T {
    if err == nil { return result } else { panic(err) }
}

func Map[X, Y any](fn func(x X) Y, xs []X) []Y {
    if xs == nil { return nil }
    result := make([]Y, len(xs))
    for i := 0; i < len(xs); i++ {
        result[i] = fn(xs[i])
    }
    return result
}

func AppendComments(comment string, comments ... string) string {
    var elements []string
    if len(comment) == 0 {
        elements = append([]string{}, comments...)
    } else {
        elements = append([]string{comment}, comments...)
    }
    return strings.Join(elements, "\n")
}

func AppendTags(t string, tags ... string) string {
    var elements []string
    if len(t) == 0 {
        elements = []string{}
    } else {
        elements = []string{t}
    }
    for _, tt := range tags {
        key, _, _, ok := tag.NextPair(tt)
        if !ok { continue }
        _, exists := tag.Lookup(t, key)
        if exists { continue }
        elements = append(elements, tt)
    }
    return strings.Join(elements, " ")
}

func FormatSource(s string) string {
    return strings.TrimSpace(string(Must(format.Source([]byte(s)))))
}

func RemoveElementByIndex[X any](idx int, xs []X) []X {
    if len(xs) == 0 { return xs }
    result := make([]X, len(xs) - 1)
    for i := 0; i < len(xs); i++ {
        if i == idx { continue }
        result[i] = xs[i]
    }
    return result
}

// ParseTypeList parses a comma-separated list of types, including bracketed
// tuples of types.
//
// Bracketed tuples are not recursively passed by this function but are simply
// indicated by calling visit on the entire tuple with "more" as true when
// calling the visit function.
//
// For example:
//
//     ParseTypeList(0, "a, (b, (c, d)), func (e, f)", visit)
//
// Calls visit with these arguments:
//
//     visit("a", false)
//     visit("b, (c, d)", true)
//     visit("func (e, f)", false)
//
// Returns false on parse error such as unpaired brackets.
func ParseTypeList(types string, visit func(x string, more bool) bool) bool {
    types += "," // simplify end of string handling
    bracketDepth := 0
    token := make([]rune, 0)
    ok := true

    for _, c := range types {
        // skip leading space
        if (len(token) == 0) && runeIsHSpace(c) {
            continue
        }
        token = append(token, c)

        if c == '(' {
            bracketDepth++
        } else if c == ')' {
            bracketDepth--
            if bracketDepth < 0 {
                return false
            }
        } else if (c == ',') && (bracketDepth == 0) {
            if len(token) == 0 { return false }
            x := strings.TrimSpace(string(token[0:len(token)-1]))
            if len(x) == 0 { return false }

            if (x[0] == '(') && (x[len(x)-1] == ')') {
                ok = ok && visit(strings.TrimSpace(x[1:len(x)-1]), true)
            } else {
                ok = ok && visit(x, false)
            }
            token = token[0:0]
        }
    }
    return ok && (bracketDepth == 0)
}

func runeIsHSpace(c rune) bool {
    return (c == '\t') || (c == ' ')
}

// ParseTypeListRecursive parses a comma-separated list of types, including
// bracketed tuples of types.
//
// Bracketed tuples are recursively passed by this function.
//
// For example:
//
//     ParseTypeList(0, "a, (b, (c, d)), func (e, f)", visit)
//
// Calls visit with these arguments:
//
//     visit(0, "a")
//     visit(1, "b")
//     visit(2, "c")
//     visit(2, "d")
//     visit(0, "func (e, f)")
//
// Returns false on parse error such as unpaired brackets.
func ParseTypeListRecursive(types string, visit func(depth int, x string) bool) bool {
    ok := true

    visit_flat := func(x string, more bool) bool {
        if more {
            visit2 := func(depth int, x string) bool {
                ok = ok && visit(depth + 1, x)
                return ok
            }
            ok = ok && ParseTypeListRecursive(x, visit2)
        } else {
            ok = ok && visit(0, x)
        }
        return ok
    }

    return ok && ParseTypeList(types, visit_flat)
}

// SplitTypeTuple parses flat comma-separated tuple of types, such as
// "string, error", and returns each token as a string.
func SplitTypeTuple(types string) ([]string, bool) {
    var results []string
    ok := ParseTypeList(types, func(x string, more bool) bool {
        results = append(results, x)
        return more == false
    })
    if len(results) == 0 { return nil, false }
    return results, ok
}
