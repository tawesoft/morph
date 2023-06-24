package morph

import (
    "testing"

    "github.com/tawesoft/morph/internal"
)

/*
func TestFieldExpressionType_RewriteStringSrcDest(t *testing.T) {
    fet := ConverterFieldExpression("").Type()

    src := Struct{
        Name:   "Apple",
        Fields: []Field{
            {
                Name:      "Picked",
                Type:      "time.Time",
            },
        },
    }

    dest := Struct{
        Name: "Orange",
        Fields: []Field{
            {
                Name:      "Picked",
                Type:      "int64",
                Converter: "$dest.$ = $src.$.UTC().Unix()",
            },
        },
    }

    srcArgument := Argument{
        Name: "from",
        Type: "Apple",
    }

    destArgument := Argument{
        Name: "to",
        Type: "Orange",
    }

    tests := []struct{
        sig string
        expected string
    }{
        {
            "$(src).$(type).$(untitle)To$dest.$type",
            "appleToOrange",
        },
        {
            "// as $dest.$.$type from $src.$.$(type)",
            "// as int64 from time.Time",
        },
        {
            "$dest.$ = $src.$.UTC().Unix()",
            "to.Picked = from.Picked.UTC().Unix()",
        },
    }

    for _, tt := range tests {
        s, err := fet.rewriteStringSrcDest(tt.sig, src, srcArgument, src.Fields[0], dest, destArgument)
        if err != nil {
            t.Errorf("error formatting %q: %v", tt.sig, err)
        } else if s != tt.expected {
            t.Logf("got: %s", s)
            t.Logf("expected: %s", tt.expected)
            t.Errorf("unexpected output formatting %q.", tt.sig)
        }
    }
}
*/

func TestFunction_String(t *testing.T) {
    tests := []struct{
        Input Function
        Expected string
    }{
        { // test 0
            Input: Function{
                Signature: FunctionSignature{
                    Comment:   "Foo is a function that panics.",
                    Name:      "Foo",
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.Must(internal.FormatSource(`
// Foo is a function that panics.
func Foo() {
    panic("not implemented")
}`)),
        },
        { // test 1
            Input: Function{
                Signature: FunctionSignature{
                    Comment:   "Foo is a complicated function.\nMultiline comment.",
                    Name:      "Foo",
                    Type:      []Argument{
                        {Name: "X", Type: "any"},
                        {Name: "Y", Type: "interface{}"},
                        {Name: "Z", Type: "interface{~[]Y}"},
                    },
                    Arguments: []Argument{
                        {Name: "i", Type: "maybe.M[X]"},
                        {Name: "j", Type: "either.E[Y, Z]"},
                    },
                    Returns:   []Argument{
                        {Name: "named1", Type: "int"},
                        {Name: "named2", Type: "bool"},
                    },
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.Must(internal.FormatSource(`
// Foo is a complicated function.
// Multiline comment.
func Foo[X any, Y interface{}, Z interface{~[]Y}](i maybe.M[X], j either.E[Y, Z]) (named1 int, named2 bool) {
    panic("not implemented")
}`)),
        },
        { // test 2
            Input: Function{
                Signature: FunctionSignature{
                    Comment:   "Bar is a method on a Foo",
                    Name:      "Bar",
                    Receiver: Argument{
                        Name:    "f",
                        Type:    "*Foo[X]",
                    },
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.Must(internal.FormatSource(`
// Bar is a method on a Foo
func (f *Foo[X]) Bar() {
    panic("not implemented")
}`)),
        },
        { // test 2
            Input: Function{
                Signature: FunctionSignature{
                    Comment:   "Foo is a higher-order function",
                    Name:      "Foo",
                    Returns:   []Argument{
                        {
                            Type: "func(string) (float64, error)",
                        },
                    },
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.Must(internal.FormatSource(`
// Foo is a higher-order function
func Foo() func(string) (float64, error) {
    panic("not implemented")
}`)),
        },
    }

    for i, test := range tests {
        output := test.Input.String()
        if output != test.Expected {
            t.Logf("got %q", output)
            t.Logf("expected %q", test.Expected)
            t.Errorf("test %d failed", i)
        }
    }
}
