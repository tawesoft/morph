package morph

import (
    "fmt"
    "go/parser"
    "reflect"
    "strings"
    "testing"

    "github.com/tawesoft/morph/internal"
)

func Test_simpleTypeExpr(t *testing.T) {
    rows := []struct {
        input string
        cmp string
    }{
        {"int",          "int"},
        {"T",            "T"},
        {"a",            "a"},
        {"*a",           "*a"},
        {"a[T]",         "a"},
        {"a[A, B]",      "a"},
        {"[]a",          ""},
        {"[2]a",         ""},
        {"map[a]b",      ""},
        {"func() bool",  ""},
        {"func()",       ""},
    }
    for _, test := range rows {
        x, err := parser.ParseExpr(test.input)
        if (err != nil) {
            t.Errorf("failed to parse %q", test.input)
            continue
        }
        out, ok := simpleTypeExpr(x)
        if (test.cmp != out) {
            t.Errorf("expected simpleTypeExpr(%q) == %q but got %q, %t",
                test.input, test.cmp, out, ok)
        }
    }
}

func Test_parseTypeList(t *testing.T) {
    tests := []struct {
        input string
        expected []string
        fails bool
    }{
        {
            input:    "int",
            expected: []string{`"int":false`},
        },
        {
            input: "int, string",
            expected: []string{`"int":false`, `"string":false`},
        },
        {
            input: "a, (b, c), d",
            expected: []string{`"a":false`, `"b, c":true`, `"d":false`},
        },
        {
            input: " a , ( b , c ) , d ",
            expected: []string{`"a":false`, `"b , c":true`, `"d":false`},
        },
        {
            input: "a, func(a, b) (c, d), e",
            expected: []string{`"a":false`, `"func(a, b) (c, d)":false`, `"e":false`},
        },
        {
            input: "a, (b, (c, d)), e",
            expected: []string{`"a":false`, `"b, (c, d)":true`, `"e":false`},
        },
    }

    captured := make([]string, 0)
    for i, tt := range tests {
        captured = captured[0:0]
        ok := internal.ParseTypeList(tt.input, func(x string, more bool) bool {
            captured = append(captured, fmt.Sprintf("%q:%t", x, more))
            return true
        })
        if (!ok) != tt.fails {
            t.Errorf("test %d was ok=%t but expected fails=%t", i, ok, tt.fails)
        } else if !reflect.DeepEqual(captured, tt.expected) {
            t.Logf("got: {%v}", strings.Join(captured, ", "))
            t.Logf("expected: {%s}", strings.Join(tt.expected, ", "))
            t.Errorf("compare failed on test %d", i)
        }
    }
}

func Test_parseTypeListRecursive(t *testing.T) {
    tests := []struct {
        input string
        expected []string
        fails bool
    }{
        {
            input:    "a",
            expected: []string{`0:"a"`},
        },
        {
            input: "a, b",
            expected: []string{`0:"a"`, `0:"b"`},
        },
        {
            input: "a, (b, c), d",
            expected: []string{`0:"a"`, `1:"b"`, `1:"c"`, `0:"d"`},
        },
        {
            input: " a , ( b , c ) , d ",
            expected: []string{`0:"a"`, `1:"b"`, `1:"c"`, `0:"d"`},
        },
        {
            input: "a, func(a, b) (c, d), e",
            expected: []string{`0:"a"`, `0:"func(a, b) (c, d)"`, `0:"e"`},
        },
        {
            input: "a, (b, (c, d)), e",
            expected: []string{`0:"a"`, `1:"b"`, `2:"c"`, `2:"d"`, `0:"e"`},
        },
    }

    captured := make([]string, 0)
    for i, tt := range tests {
        captured = captured[0:0]
        ok := internal.ParseTypeListRecursive(tt.input, func(depth int, x string) bool {
            captured = append(captured, fmt.Sprintf("%d:%q", depth, x))
            return true
        })
        if (!ok) != tt.fails {
            t.Errorf("test %d was ok=%t but expected fails=%t", i, ok, tt.fails)
        } else if !reflect.DeepEqual(captured, tt.expected) {
            t.Logf("got: {%v}", strings.Join(captured, ", "))
            t.Logf("expected: {%s}", strings.Join(tt.expected, ", "))
            t.Errorf("compare failed on test %d", i)
        }
    }
}
