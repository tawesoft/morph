package morph_test

import (
    "reflect"
    "testing"

    "github.com/tawesoft/morph"
)


// TestParseFile tests parsing a struct from source code correctly extracts
// the information needed.
func TestParseFile(t *testing.T) {
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

type Foo struct {
    a, b int
    C string
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
    a, b int "foo"
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
            Desc: "full/Foo",
            Source: full,
            Name: "Foo",
            Expected: morph.Struct{
                Name:   "Foo",
                Fields: []morph.Field{
                    {Name: "a", Type: "int"},
                    {Name: "b", Type: "int"},
                    {Name: "C", Type: "string"},
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
                    {Name: "a", Type: "int", Tag: `"foo"`},
                    {Name: "b", Type: "int", Tag: `"foo"`},
                },
            },
        },
        {
            Desc: "full/Empty",
            Source: full,
            Name: "Empty",
            Expected: morph.Struct{
                Name:   "Empty",
                Fields: []morph.Field{},
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
