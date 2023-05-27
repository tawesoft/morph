package internal_test

import (
    "fmt"
    "testing"

    "github.com/tawesoft/morph/internal"
)

func TestTokenReplacer_Replace(t *testing.T) {
    tr := internal.TokenReplacer{
        ByIndex: func(i int) (string, bool) {
            return fmt.Sprintf("ByIndex(%d)", i), true
        },
        ByName: func(name string) (string, bool) {
            return fmt.Sprintf("ByName(%q)", name), true
        },
        TupleByIndex: func(i, j int) (string, bool) {
            return fmt.Sprintf("TupleByIndex(%d, %d)", i, j), true
        },
        TupleByName: func(name string, i int) (string, bool) {
            return fmt.Sprintf("TupleByName(%q, %d)", name, i), true
        },
    }

    tests := []struct{
        input string
        expected string
        fails bool
    }{
        {input: "foo bar",      expected: "foo bar"},
        {input: "$",            fails: true},
        {input: "$a",           expected: `ByName("a")`},
        {input: "$a,b",         expected: `ByName("a"),b`},
        {input: "$a.b",         expected: `ByName("a").b`},
        {input: "$a.0",         expected: `TupleByName("a", 0)`},
        {input: "$a.0,b",       expected: `TupleByName("a", 0),b`},
        {input: "$0",           expected: `ByIndex(0)`},
        {input: "$0,b",         expected: `ByIndex(0),b`},
        {input: "$0.b",         expected: `ByIndex(0).b`},
        {input: "$0.1",         expected: `TupleByIndex(0, 1)`},
        {input: `foo "$0"`,     expected: `foo "$0"`},
        {input: `foo "\"$0"`,   expected: `foo "\"$0"`},
        {input: `$0 "`,         fails: true},
    }

    for _, tt := range tests {
        result, err := tr.Replace(tt.input)
        if (err == nil) == tt.fails {
            t.Errorf("test %s had error: %v (but expected fails=%t)",
                tt.input, err, tt.fails)
            continue
        }

        if result != tt.expected {
            t.Logf("got: %s", result)
            t.Logf("expected: %s", tt.expected)
            t.Errorf("test %q failed", tt.input)
        }
    }
}
