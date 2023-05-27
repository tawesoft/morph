package morph_test

import (
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/internal"
)

func TestStruct_String(t *testing.T) {
    tests := []struct{
        Input morph.Struct
        Expected string
    }{
        { // test 0
            Input: morph.Struct{
                Comment:    "Comment.\nMultiline.",
                Name:       "Name",
                TypeParams: []morph.Field{
                    {Name: "X", Type: "any"},
                },
                Fields:     []morph.Field{
                    {
                        Name:    "One",
                        Type:    "maybe.M[X]",
                        Value:   "maybe.Some(New[X]())",
                        Tag:     `tag:"foo"`,
                        Comment: "Field One",
                    },
                    {
                        Name:    "Two",
                        Type:    "int",
                        Value:   "123",
                        Comment: "Comment.\nMultiline.",
                    },
                },
            },
            Expected: internal.FormatSource(`
// Comment.
// Multiline.
type Name[X any] struct {
    One maybe.M[X] `+"`"+`tag:"foo"`+"`"+` // Field One
    // Comment.
    // Multiline.
    Two int
}`),
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

func TestFunction_String(t *testing.T) {
    tests := []struct{
        Input morph.Function
        Expected string
    }{
        { // test 0
            Input: morph.Function{
                Signature: morph.FunctionSignature{
                    Comment:   "Foo is a function that panics.",
                    Name:      "Foo",
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.FormatSource(`
// Foo is a function that panics.
func Foo() {
    panic("not implemented")
}`),
        },
        { // test 1
            Input: morph.Function{
                Signature: morph.FunctionSignature{
                    Comment:   "Foo is a complicated function.\nMultiline comment.",
                    Name:      "Foo",
                    Type:      []morph.Field{
                        {Name: "X", Type: "any"},
                        {Name: "Y", Type: "interface{}"},
                        {Name: "Z", Type: "interface{~[]Y}"},
                    },
                    Arguments: []morph.Field{
                        {Name: "i", Type: "maybe.M[X]"},
                        {Name: "j", Type: "either.E[Y, Z]"},
                    },
                    Returns:   []morph.Field{
                        {Name: "named1", Type: "int"},
                        {Name: "named2", Type: "bool"},
                    },
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.FormatSource(`
// Foo is a complicated function.
// Multiline comment.
func Foo[X any, Y interface{}, Z interface{~[]Y}](i maybe.M[X], j either.E[Y, Z]) (named1 int, named2 bool) {
    panic("not implemented")
}`),
        },
        { // test 2
            Input: morph.Function{
                Signature: morph.FunctionSignature{
                    Comment:   "Bar is a method on a Foo",
                    Name:      "Bar",
                    Receiver: morph.Field{
                        Name:    "f",
                        Type:    "*Foo[X]",
                    },
                },
                Body: `panic("not implemented")`,
            },
            Expected: internal.FormatSource(`
// Bar is a method on a Foo
func (f *Foo[X]) Bar() {
    panic("not implemented")
}`),
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

func TestWrappedFunction_String(t *testing.T) {
    divide := morph.Function{
        Signature: morph.FunctionSignature{
            Name:      "Divide",
            Comment:   "Divide returns a divided by b. It is an error to divide by zero.",
            Arguments: []morph.Field{
                {Name: "a", Type: "float64"},
                {Name: "b", Type: "float64"},
            },
            Returns:   []morph.Field{
                {Name: "value", Type: "float64"},
                {Name: "err",   Type: "error"},
            },
        },
        Body: `
    if b == 0.0 {
        return 0, DivideByZeroError
    } else {
        return a / b, nil
    }
`,
    }.Wrap()

    tests := []struct{
        wrapped morph.WrappedFunction
        expected string
    }{
        {
            wrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Comment:   "$ returns the result of [Divide] with the result rewritten as (value, err == nil).",
                    Name:      "Divide_ReturnValueOk",
                    Arguments: []morph.Field{
                        {Name: "a", Type: "float64"},
                        {Name: "b", Type: "float64"},
                    },
                    Returns:   []morph.Field{
                        {Type: "float64"},
                        {Type: "bool"},
                    },
                },
                Inputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Name: "a", Type: "float64", Value: "$a",},
                        {Name: "b", Type: "float64", Value: "$b",},
                    },
                    Formatter: "$a, $b",
                },
                Outputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Name: "f",  Type: "float64", Value: "$value"},
                        {Name: "ok", Type: "bool",    Value: "$err == nil"},
                    },
                    Formatter: "$f, $ok",
                },
                Wraps: &divide,
            },
            expected: `
// Divide_ReturnValueOk returns the result of [Divide] with the result rewritten as (value, err == nil).
func Divide_ReturnValueOk(a float64, b float64) (float64, bool) {
    _in0 := a // accessible as $0 or $a
    _in1 := b // accessible as $1 or $b

    _r0, _r1 := Divide(_in0, _in1) // results accessible as $value, $err

    _out0 := value      // accessible as $0 or $f
    _out1 := err == nil // accessible as $1 or $ok

    return _out0, _out1
}
`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.wrapped.Signature.Name, func(t *testing.T) {
            result, err := tt.wrapped.Format()
            if err != nil {
                t.Errorf("error formatting wrapped function: %s", err)
            }

            expected := internal.FormatSource(tt.expected)
            if result != expected {
                t.Logf("got %+v", result)
                t.Logf("expected %+v", expected)
                t.Errorf("wrapped function does not format as expected")
            }
        })
    }
}
