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
    simpleOneInput := morph.Function{
        Signature: morph.FunctionSignature{
            Name: "SimpleOneInput",
            Arguments: []morph.Field{
                {Name: "x", Type: "float64"},
            },
        },
        Body: "x = x",
    }.Wrap()
    foo := morph.Function{
        Signature: morph.FunctionSignature{
            Name:      "Foo",
            Arguments: []morph.Field{
                {Name: "foo", Type: "float64"},
            },
            Returns:   []morph.Field{
                {Type: "float64"},
            },
        },
        Body: `
        return 2 * foo
`,
    }.Wrap()
    divide2 := morph.WrappedFunction{
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
                {Name: "i", Type: "float64", Value: "$a",},
                {Name: "j", Type: "float64", Value: "$b",},
            },
            Formatter: "$i, $j",
        },
        Outputs: morph.ArgRewriter{
            Capture: []morph.Field{
                {Name: "f",  Type: "float64", Value: "$value"},
                {Name: "ok", Type: "bool",    Value: "$err == nil"},
            },
            Formatter: "$f, $ok",
        },
        Wraps: &divide,
    }

    tests := []struct{
        wrapped morph.WrappedFunction
        expected string
        fails bool
    }{
        {
            wrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Name:      "SimpleConstArg",
                },
                Inputs: morph.ArgRewriter{
                    Formatter: "2",
                },
                Wraps: &simpleOneInput,
            },
            expected: `
func SimpleConstArg() {
    SimpleOneInput(2)

    return
}
`,
        },
        {
            wrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Name:      "FailsInputNotReferenced",
                    Arguments: []morph.Field{
                        {Name: "a", Type: "float64"},
                    },
                },
                Inputs: morph.ArgRewriter{
                    Formatter: "2",
                },
                Wraps: &simpleOneInput,
            },
            fails: true,
        },
        {
            wrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Name:      "FailsByIndex",
                    Arguments: []morph.Field{
                        {Name: "a", Type: "float64"},
                    },
                },
                Inputs: morph.ArgRewriter{
                    Capture:   []morph.Field{
                        {Name: "", Type: "float64", Value: "$1",},
                    },
                    Formatter: "$0",
                },
                Wraps: &simpleOneInput,
            },
            fails: true,
        },
        {
            wrapped: divide2,
            expected: `
// Divide_ReturnValueOk returns the result of [Divide] with the result rewritten as (value, err == nil).
func Divide_ReturnValueOk(a float64, b float64) (float64, bool) {
    _in0 := a // accessible as $0 or $i
    _in1 := b // accessible as $1 or $j

    _r0, _r1 := Divide(_in0, _in1) // results accessible as $value, $err

    _out0 := _r0        // accessible as $0 or $f
    _out1 := _r1 == nil // accessible as $1 or $ok

    return _out0, _out1
}
`,
        },
        {
            wrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Comment:   "$ returns the result of Divide(a, 2) as (float64, bool) by wrapping Divide_ReturnValueOk.",
                    Name:      "Divide_Const_ReturnValueOk",
                    Arguments: []morph.Field{
                        {Name: "a", Type: "float64"},
                    },
                    Returns:   []morph.Field{
                        {Type: "float64"},
                        {Type: "bool"},
                    },
                },
                Inputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Type: "float64", Value: "$0"},
                    },
                    Formatter: "$0, 2",
                },
                Outputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Type: "float64", Value: "$0"},
                        {Type: "bool",    Value: "$1"},
                    },
                    Formatter: "$0, $1",
                },
                Wraps: &divide2,
            },
            expected: `
// Divide_Const_ReturnValueOk returns the result of Divide(a, 2) as (float64, bool) by wrapping Divide_ReturnValueOk.
func Divide_Const_ReturnValueOk(a float64) (float64, bool) {
    // from Divide_ReturnValueOk
    _f0 := func(a float64, b float64) (float64, bool) {
    _in0 := a // accessible as $0 or $i
    _in1 := b // accessible as $1 or $j

        _r0, _r1 := Divide(_in0, _in1) // results accessible as $value, $err

        _out0 := _r0        // accessible as $0 or $f
        _out1 := _r1 == nil // accessible as $1 or $ok

        return _out0, _out1
    }

    _in0 := a // accessible as $0

    _r0, _r1 := _f0(_in0, 2) // results accessible as $0, $1

    _out0 := _r0 // accessible as $0
    _out1 := _r1 // accessible as $1

    return _out0, _out1
}
`,
        },
        {
            wrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Comment:   "$ returns the result of math.Modf(Foo(x)).",
                    Name:      "Foo_Modf",
                    Arguments: []morph.Field{
                        {Name: "x", Type: "float64"},
                    },
                    Returns:   []morph.Field{
                        {Type: "float64"},
                        {Type: "float64"},
                    },
                },
                Inputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Type: "float64", Value: "$0"},
                    },
                    Formatter: "$0",
                },
                Outputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Type: "float64, float64", Value: "math.Modf($0)"},
                    },
                    Formatter: "$0.0, $0.1",
                },
                Wraps: &foo,
            },
            expected: `
// Foo_Modf returns the result of math.Modf(Foo(x)).
func Foo_Modf(x float64) (float64, float64) {
    _in0 := x // accessible as $0

    _r0 := Foo(_in0) // results accessible as $0

    _out0_0, _out0_1 := math.Modf(_r0) // accessible as $0.N

    return _out0_0, _out0_1
}
`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.wrapped.Signature.Name, func(t *testing.T) {
            result, err := tt.wrapped.Format()
            if tt.fails != (err != nil) {
                if tt.fails == false {
                    t.Errorf("error formatting wrapped function: %v", err)
                } else {
                    t.Errorf("wrapped function successfully, but expected to fail")
                }
                return
            }

            expected := internal.FormatSource(tt.expected)
            if result != expected {
                t.Logf("got:\n%+v", result)
                t.Logf("expected:\n%+v", expected)
                t.Errorf("wrapped function does not format as expected")
            }
        })
    }
}
