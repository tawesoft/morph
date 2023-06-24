package morph_test

import (
    "fmt"
    "go/parser"
    "reflect"
    "strings"
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/internal"
)

func Test_parseSimpleTypeExpr(t *testing.T) {
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
        out, ok := internal.SimpleTypeExpr(x)
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

func TestParseStruct(t *testing.T) {
    type Test struct {
        Desc string
        Source string
        Name string
        Expected morph.Struct
        IsError bool
    }

    notInGlobalScope :=`
package foo

func Foo() {
    type Foo struct {
        x int
    }
}
`

    full := `
package foo

type First struct {
}

// Foo is a foo that can foo things.
type Foo struct {
    // a and b are lorem ipsum
    a, b int
    C string // C is lorem ipsum
    D struct{a, b int}
    f func(a, b string) (error, string)
    t time.Time
}

type Empty struct {}

type Embeds struct {
    Foo
    bar int
}

type Tags struct {
    a, b int `+"`"+`tag:"foo"`+"`"+`
}

type Generic[A any, B any] struct {
    a A
    b B
    arr [][4]A
}

type GenericEmbed struct {
    g Generic[AnotherPackage.Constraint]
}
`

    tests := []Test{
        {
            Desc: "full/First",
            Source: full,
            Name: "",
            Expected: morph.Struct{
                Name:   "First",
            },
        },
        {
            Desc: "full/Foo",
            Source: full,
            Name: "Foo",
            Expected: morph.Struct{
                Comment: "Foo is a foo that can foo things.",
                Name:   "Foo",
                Fields: []morph.Field{
                    {Name: "a", Type: "int", Comment: "a and b are lorem ipsum"},
                    {Name: "b", Type: "int", Comment: "a and b are lorem ipsum"},
                    {Name: "C", Type: "string", Comment: "C is lorem ipsum"},
                    {Name: "D", Type: "struct{a, b int}"},
                    {Name: "f", Type: "func(a, b string) (error, string)"},
                    {Name: "t", Type: "time.Time",},
                },
            },
        },
        {
            Desc: "full/Embeds",
            Source: full,
            Name: "Embeds",
            Expected: morph.Struct{
                Name:   "Embeds",
                Fields: []morph.Field{
                    {Name: "Foo", Type: "Foo"},
                    {Name: "bar", Type: "int"},
                },
            },
        },
        {
            Desc: "full/Tags",
            Source: full,
            Name: "Tags",
            Expected: morph.Struct{
                Name:   "Tags",
                Fields: []morph.Field{
                    {Name: "a", Type: "int", Tag: `tag:"foo"`},
                    {Name: "b", Type: "int", Tag: `tag:"foo"`},
                },
            },
        },
        {
            Desc: "full/Empty",
            Source: full,
            Name: "Empty",
            Expected: morph.Struct{
                Name:   "Empty",
            },
        },
        {
            Desc: "full/Generic",
            Source: full,
            Name: "Generic",
            Expected: morph.Struct{
                Name:   "Generic",
                TypeParams: []morph.Field{
                    {Name: "A", Type: "any"},
                    {Name: "B", Type: "any"},
                },
                Fields: []morph.Field{
                    {Name: "a", Type: "A"},
                    {Name: "b", Type: "B",},
                    {Name: "arr",Type: "[][4]A",},
                },
            },
        },
        {
            Desc: "full/GenericEmbed",
            Source: full,
            Name: "GenericEmbed",
            Expected: morph.Struct{
                Name:   "GenericEmbed",
                Fields: []morph.Field{
                    {Name: "g", Type: "Generic[AnotherPackage.Constraint]",},
                },
            },
        },
        {
            Desc: "notInGlobalScope/Foo",
            Source: notInGlobalScope,
            IsError: true,
        },
    }

    for _, row := range tests {
        s, err := morph.ParseStruct("test.go", row.Source, row.Name)
        if err != nil {
            if row.IsError {
                continue
            } else {
                t.Errorf("Parse failed for test %q: %v", row.Desc, err)
                continue
            }
        }
        if !reflect.DeepEqual(s, row.Expected) {
            t.Errorf("unexpected result for test %q: got %+v, expected %+v", row.Desc, s, row.Expected)
        }
    }
}

func TestParseFunctionSignature(t *testing.T) {
    type Test struct {
        Desc string
        Source string
        Getter func(source string) (morph.FunctionSignature, error)
        Expected morph.FunctionSignature
        IsError bool
    }

    GetNamed := func (name string) func(source string) (morph.FunctionSignature, error) {
        return func(source string) (morph.FunctionSignature, error) {
            return morph.ParseFunctionSignature("test.go", source, name)
        }
    }
    GetMethod := func (Type, name string) func(source string) (morph.FunctionSignature, error) {
        return func(source string) (morph.FunctionSignature, error) {
            return morph.ParseMethodSignature("test.go", source, Type, name)
        }
    }
    GetFirst := func(source string) (morph.FunctionSignature, error) {
        return morph.ParseFirstFunctionSignature("test.go", source)
    }

    source := `
package foo

func (foo T) First() {}

func Foo() {
    NotInGlobalScope := func() int {
        return 0
    }
}

// Function comment
func Function[T, X any](a int, b T[X]) (T[X], bool) {
    return nil
}

// Method comment
func (foo T) Method(a int) (namedReturn string) {
    return "foo"
}

// Method comment
// over two lines
func (foo *T) MethodWithPointerReciever(a *int) *string {
    return "bar"
}
`

    tests := []Test{
        {
            Desc: "source/First",
            Source: source,
            Getter: GetFirst,
            Expected: morph.FunctionSignature{
                Name:      "First",
                Receiver:  morph.Argument{
                    Name: "foo",
                    Type: "T",
                },
            },
        },
        {
            Desc: "source/Named",
            Source: source,
            Getter: GetNamed("Function"),
            Expected: morph.FunctionSignature{
                Comment:   "Function comment",
                Name:      "Function",
                Type:      []morph.Argument{
                    {Name: "T", Type: "any"},
                    {Name: "X", Type: "any"},
                },
                Arguments: []morph.Argument{
                    {Name: "a", Type: "int"},
                    {Name: "b", Type: "T[X]"},
                },
                Returns:   []morph.Argument{
                    {Type: "T[X]"},
                    {Type: "bool"},
                },
            },
        },
        {
            Desc: "source/Method",
            Source: source,
            Getter: GetMethod("T", "Method"),
            Expected: morph.FunctionSignature{
                Comment:   "Method comment",
                Name:      "Method",
                Arguments: []morph.Argument{
                    {Name: "a", Type: "int"},
                },
                Returns:   []morph.Argument{
                    {Name: "namedReturn", Type: "string"},
                },
                Receiver:  morph.Argument{Name: "foo", Type: "T"},
            },
        },
    }

    for _, row := range tests {
        s, err := row.Getter(row.Source)
        if err != nil {
            if row.IsError {
                continue
            } else {
                t.Errorf("Parse failed for test %q: %v", row.Desc, err)
                continue
            }
        }
        if !reflect.DeepEqual(s, row.Expected) {
            t.Logf("got %+v", s)
            t.Logf("expected %+v", row.Expected)
            t.Errorf("unexpected result for test %q", row.Desc)
        }

    }
}
