package internal

import (
    "go/format"
    "os"
    "os/exec"
    "path"
    "strings"
    "testing"
    "unicode"

    "github.com/tawesoft/morph/tag"
)

// TestCompileAndRun compiles and runs a Go program (using "go run") and
// verifies that:
//
//   * it compiles successfully
//   * it generates a normal exit code
//   * it doesn't write to stderr
//   * captures the writes to stdout
//   * asserts that the provided "expected" func returns a nil error when
//     called with the captured input.
//
// WARNING: this function can compile and run arbitrary Go code. This function
// MUST NOT be used on untrusted sources.
func TestCompileAndRun(t *testing.T, source string, expected func(stdout string) error) {
    dir := t.TempDir()
    dest := path.Join(dir, "generated-for-morph-test.go")

    err := os.WriteFile(dest, []byte(source), 0600)
    if err != nil {
        t.Fatalf("could not write temporary file %q", dest)
    }

    cmd := exec.Command("go", "run", dest)
    var stdout, stderr strings.Builder
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    err = cmd.Run()
    t.Logf("source: %s", source)
    if (err != nil) {
        t.Fatalf("generated code failed to compile: %v", err)
    }

    sout, serr := stdout.String(), stderr.String()
    if sout != "" {
        t.Logf("stdout: %s", sout)
    }
    if serr != "" {
        t.Logf("stderr: %s", serr)
    }
    if err := expected(sout); err != nil {
        t.Fatalf("unexpected result: %v", err)
    }
}

// FirstOrDefault returns the first value in the slice, or, if empty, the default value
func FirstOrDefault[X comparable](xs []X, defaultIfMissing X) X {
    if len(xs) >= 1 {
        return xs[0]
    } else {
        return defaultIfMissing
    }
}

func Filter[X any](filter func(x X) bool, xs []X) []X {
    var result []X
    for _, x := range xs {
        if filter(x) {
            result = append(result, x)
        }
    }
    return result
}

// RecursiveCopySlice deeply copies a slice of elements that each, in turn,
// must be copied using their own Copy method.
func RecursiveCopySlice[X interface{Copy() X}](xs []X) []X {
    results := []X(nil)
    for _, x := range xs {
        results = append(results, x.Copy())
    }
    return results
}

// RewriteSignatureString performs the special '$' replacement in a function
// signature specified as a string.
//
// TODO use tokenReplacer instead
// Deprecated.
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

func IsGoIdentStarter(x rune) bool {
    return x == '_' || unicode.IsLetter(x)
}

func IsGoIdent(x rune) bool {
    return x == '_' || unicode.IsLetter(x) || unicode.IsDigit(x)
}

func IsGoIdentIdx(x rune, idx int) bool {
    return IsGoIdentStarter(x) || ((idx > 0) && unicode.IsDigit(x))
}

func First[T any](xs []T) (T, bool) {
    var zero T
    if len(xs) == 0 { return zero, false }
    return xs[0], true
}

func Last[T any](xs []T) (T, bool) {
    var zero T
    if len(xs) == 0 { return zero, false }
    return xs[len(xs)-1], true
}

func Must[T any](result T, err error) T {
    if err == nil { return result } else { panic(err) }
}

func Assert(err error) {
    if err != nil { panic(err) }
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

func FormatSource(source string) (string, error) {
    s, err := format.Source([]byte(source))
    if err != nil { return "", err }
    return strings.TrimSpace(string(s)), nil
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

func runeIsHSpace(c rune) bool {
    return (c == '\t') || (c == ' ')
}
