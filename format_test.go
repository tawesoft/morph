package morph_test

import (
    "go/format"
    "strings"
    "testing"

    "github.com/tawesoft/morph"
)

func formatSource(s string) string {
    return strings.TrimSpace(string(must(format.Source([]byte(s)))))
}

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
            Expected: formatSource(`
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

func TestFunction_String_String(t *testing.T) {
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
            Expected: formatSource(`
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
            Expected: formatSource(`
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
            Expected: formatSource(`
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
