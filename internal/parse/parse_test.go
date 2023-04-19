package parse_test

import (
    "reflect"
    "testing"

    "github.com/tawesoft/morph/internal/parse"
)

func TestParseFile(t *testing.T) {
    type Test struct {
        Desc string
        Source string
        Expected []parse.Struct
        Filter func(string) bool
    }

    tests := []Test{
        {
            Desc: "full",
            Source: `
package foo

type Foo struct {
    a, b int
    C string
    D struct{a, b int}
    f func(a, b string) (error, string)
    t time.Time
}

type Empty struct {}

type Ignored struct {
    Foo
}

type Embeds struct {
    Foo
    bar int
}

type Generic[T any] struct {
    a, b T
    arr [][4]T
}

type GenericEmbed struct {
    g Generic[int]
}
`,
            Expected: []parse.Struct{
                {
                    Name:   "Foo",
                    Fields: []parse.Field{
                        {
                            Name: "a",
                            Type: "int",
                        },
                        {
                            Name: "b",
                            Type: "int",
                        },
                        {
                            Name: "C",
                            Type: "string",
                        },
                        {
                            Name: "D",
                            Type: "struct{a, b int}",
                        },
                        {
                            Name: "f",
                            Type: "func(a, b string) (error, string)",
                        },
                        {
                            Name: "t",
                            Type: "time.Time",
                        },
                    },
                },
                {
                    Name: "Empty",
                    Fields: []parse.Field{},
                },
                {
                    Name:   "Embeds",
                    Fields: []parse.Field{
                        {
                            Name: "Foo",
                            Type: "Foo",
                        },
                        {
                            Name: "bar",
                            Type: "int",
                        },
                    },
                },
                {
                    Name:   "Generic",
                    Fields: []parse.Field{
                        {
                            Name: "a",
                            Type: "T",
                        },
                        {
                            Name: "b",
                            Type: "T",
                        },
                        {
                            Name: "arr",
                            Type: "[][4]T",
                        },
                    },
                },
                {
                    Name:   "GenericEmbed",
                    Fields: []parse.Field{
                        {
                            Name: "g",
                            Type: "Generic[int]",
                        },
                    },
                },
            },
            Filter: func(x string) bool { return x != "Ignored" },
        },
        {
            Desc: "only global scope",
            Source: `
package foo

func Foo() {
    type Foo struct {
        x int
    }
}
`,
            Expected: nil,
        },
    }

    for _, row := range tests {
        var structs []parse.Struct
        err := parse.File("test.go", row.Source, row.Filter, func(s parse.Struct) {
            structs = append(structs, s)
        })
        if err != nil {
            t.Errorf("Parse failed for test %q: %v", row.Desc, err)
            continue
        }
        if !reflect.DeepEqual(structs, row.Expected) {
            t.Errorf("unexpected result for test %q: got %+v", row.Desc, structs)
        }

    }
}
