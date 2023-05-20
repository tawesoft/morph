package morph_test

import (
    "strings"
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fields"
)

const testSource = `
package test

// Comment on Apple
type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    custom.Grams
    Price     custom.Price `+"`"+`tag:"foo"`+"`"+` // in pence
}

// Comment on GenericApple
type GenericApple[T any, W any, P any] struct {
    Picked    T
    LastEaten T
    Weight    W
    Price     P `+"`"+`tag:"foo"`+"`"+` // in pence
}
`

func must[X any](x X, err error) X {
    if err != nil { panic(err) }
    return x
}

func morphAllFields(source morph.Struct, input morph.Field, emit func(output morph.Field)) {
    comment := input.Comment
    if len(comment) > 0 {
        comment = fields.AppendComments(comment, "comment added by morph")
    } else {
        comment = "comment added by morph"
    }
    tag := input.Tag
    if len(input.Tag) > 0 {
        tag = fields.AppendTags(input.Tag, `test:"morph"`)
    }
    emit(morph.Field{
        Name:    "$2",
        Type:    "maybe.M[$]",
        Value:   "maybe.Some($)",
        Tag:     tag,
        Comment: comment,
    })
}

func TestStruct_Struct(t *testing.T) {
    // This is a complete end-to-end test.
    tests := []struct{
        input morph.Struct
        signature string
        mapper morph.Mapper
        expected string
    }{
        { // test 0
            input:     must(morph.ParseStruct("test.go", testSource, "Apple")),
            signature: "Orange",
            mapper:    morphAllFields,
            expected:  formatSource(`
type Orange struct {
    Picked2    maybe.M[time.Time] // comment added by morph
    LastEaten2 maybe.M[time.Time] // comment added by morph
    Weight2    maybe.M[custom.Grams] // comment added by morph
    // in pence
    // comment added by morph
    Price2     maybe.M[custom.Price] `+"`"+`tag:"foo" test:"morph"`+"`"+`
}`),
        },
        { // test 1
            input:     must(morph.ParseStruct("test.go", testSource, "GenericApple")),
            signature: "Orange[T any, W any, P any]",
            mapper:    morphAllFields,
            expected:  formatSource(`
type Orange[T any, W any, P any] struct {
    Picked2    maybe.M[T] // comment added by morph
    LastEaten2 maybe.M[T] // comment added by morph
    Weight2    maybe.M[W] // comment added by morph
    // in pence
    // comment added by morph
    Price2     maybe.M[P] `+"`"+`tag:"foo" test:"morph"`+"`"+`
}`),
        },
    }

    for i, test := range tests {
        s, err := test.input.Struct(test.signature, test.mapper)
        if err != nil {
            t.Errorf("test %d failed: error: %v", i, err)
            continue
        }
        result := strings.TrimSpace(s.String())
        if result != test.expected {
            t.Logf("got:\n%q", result)
            t.Logf("expected:\n%q", test.expected)
            t.Errorf("test %d failed: unexpected output", i)
        }
    }
}

func TestStruct_Function(t *testing.T) {
    // This is a complete end-to-end test.
    tests := []struct{
        input morph.Struct
        signature string
        mapper morph.Mapper
        expected string
    }{
        { // test 0
            input:     must(morph.ParseStruct("test.go", testSource, "Apple")),
            signature: "AppleToOrange(from Apple) Orange",
            mapper:    morphAllFields,
            expected:  formatSource(`
func AppleToOrange(from Apple) Orange {
    return Orange{
        Picked2:    maybe.Some(from.Picked),
        LastEaten2: maybe.Some(from.LastEaten),
        Weight2:    maybe.Some(from.Weight),
        Price2:     maybe.Some(from.Price),
    }
}`),
        },
    }

    for i, test := range tests {
        s, err := test.input.Function(test.signature, test.mapper)
        if err != nil {
            t.Errorf("test %d failed: error: %v", i, err)
            continue
        }
        result := s.String()
        if result != test.expected {
            t.Logf("got:\n%q", result)
            t.Logf("expected:\n%q", test.expected)
            t.Errorf("test %d failed: unexpected output", i)
        }
    }
}
