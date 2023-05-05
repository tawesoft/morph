package morph

import (
    "go/parser"
    "testing"
)

func Test_simpleType(t *testing.T) {
    type row struct {
        input string
        cmp string
    }
    rows := []row{
        {"int",      "int"},
        {"T",        "T"},
        {"a",        "a"},
        {"*a",       "*a"},
        {"a[T]",     "a[T]"},
        {"a[A, B]",  "a[A, B]"},
        {"[]a",      ""},
        {"map[a]b",  ""},
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
