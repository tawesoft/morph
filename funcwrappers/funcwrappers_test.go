package funcwrappers_test

import (
    "reflect"
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/funcwrappers"
)

func Test(t *testing.T) {
    return // TODO

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

    tests := []struct {
        Name string
        Input morph.WrappedFunction
        Wrapper morph.FunctionWrapper
        ExpectedWrapped morph.WrappedFunction
        ExpectedResult string
    }{
        {
            Name: "SetArg_Divide",
            Input: divide,
            Wrapper: funcwrappers.SetArg("b", "2"), // "2", "float64"
            ExpectedWrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Comment:   "$ returns the result of [Divide] called with the arguments (a, 2).",
                    Name:      "__SetArg__Divide",
                    Arguments: []morph.Field{
                        {Name: "a", Type: "float64"},
                    },
                    Returns:   []morph.Field{
                        {Type: "float64"},
                        {Type: "error"},
                    },
                },
                Inputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Name: "a", Type: "float64", Value: "$0"},
                    },
                    Formatter: "$a, 2",
                },
                Outputs: morph.ArgRewriter{
                    Capture: []morph.Field{
                        {Type: "float64", Value: "$0"},
                        {Type: "error", Value: "$1"},
                    },
                },
                Wraps: &divide,
            },
            ExpectedResult: `
// __SetArg__Divide returns the result of [Divide] called with the arguments (a, 2).
func __SetArg__Divide(a float64) float64 {
}
`,
        },
        {
            Name: "RewriteResults_Divide",
            Input: divide,
            Wrapper: funcwrappers.SimpleRewriteResults("$0, $1 == nil", "float64, bool"),
            ExpectedWrapped: morph.WrappedFunction{
                Signature: morph.FunctionSignature{
                    Comment:   "$ returns the result of [Divide] with the result rewritten as (value, err == nil).",
                    Name:      "__RewriteResults__Divide",
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
            ExpectedResult: `
// __RewriteResults__Divide returns the result of [Divide] with the result rewritten as ($0, $1 == nil).
func __RewriteResults__Divide(a float64, b float64) (float64, bool) {
    TODO
}
`,
        },
    }

    for _, tt := range tests {
        t.Run(tt.Name, func(t *testing.T) {
            wrapped, err := tt.Wrapper(tt.Input)
            if err != nil {
                t.Errorf("error applying wrapper: %s", err)
            }

            if !reflect.DeepEqual(wrapped, tt.ExpectedWrapped) {
                t.Logf("got %+v", wrapped)
                t.Logf("expected %+v", tt.ExpectedWrapped)
                t.Errorf("wrapped functions do not compare equal")
            }

            result := tt.ExpectedWrapped.String()
            if result != tt.ExpectedResult {
                t.Logf("got %s", result)
                t.Logf("expected %s", tt.ExpectedResult)
                t.Errorf("wrapper strings do not compare equal")
            }
        })
    }
}
