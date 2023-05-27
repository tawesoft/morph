package morph_test

import (
    "strings"
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/internal"
    "github.com/tawesoft/morph/structmappers"
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

func morphAllFields(input morph.Field, emit func(output morph.Field)) {
    input = input.AppendComments("comment added by morph")
    if len(input.Tag) > 0 {
        input = input.AppendTags(`test:"morph"`)
    }
    emit(morph.Field{
        Name:    "$2",
        Type:    "maybe.M[$]",
        Value:   "maybe.Some($.$)",
        Tag:     input.Tag,
        Comment: input.Comment,
    })
}

func TestStruct_Map(t *testing.T) {
    // This is a complete end-to-end test.
    tests := []struct{
        input morph.Struct
        name string
        mapper morph.FieldMapper
        expected string
    }{
        { // test 0
            input:     internal.Must(morph.ParseStruct("test.go", testSource, "Apple")),
            name:     "Orange",
            mapper:    morphAllFields,
            expected:  internal.FormatSource(`
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
            input:     internal.Must(morph.ParseStruct("test.go", testSource, "GenericApple")),
            name:      "Orange",
            mapper:    morphAllFields,
            expected:  internal.FormatSource(`
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
        s := test.input.
            Map(structmappers.StripComment).
            MapFields(test.mapper).
            Map(structmappers.Rename("Orange"))
        result := strings.TrimSpace(s.String())
        if result != test.expected {
            t.Logf("got:\n%s", result)
            t.Logf("expected:\n%s", test.expected)
            t.Errorf("test %d failed: unexpected output", i)
        }
    }
}

func TestStruct_Converter(t *testing.T) {
    // This is a complete end-to-end test.
    tests := []struct{
        input morph.Struct
        name string
        signature string
        mapper morph.FieldMapper
        expected string
    }{
        { // test 0
            input:     internal.Must(morph.ParseStruct("test.go", testSource, "Apple")),
            name:      "Orange",
            signature: "$FromTo$To($from $From) $To",
            mapper:    morphAllFields,
            expected:  internal.FormatSource(`
// AppleToOrange converts [Apple] to [Orange].
func AppleToOrange(apple Apple) Orange {
    return Orange{
        Picked2:    maybe.Some(apple.Picked),
        LastEaten2: maybe.Some(apple.LastEaten),
        Weight2:    maybe.Some(apple.Weight),
        Price2:     maybe.Some(apple.Price),
    }
}`),
        },
    }

    for i, test := range tests {
        s, err := test.input.
            Map(structmappers.Rename(test.name)).
            MapFields(test.mapper).
            Converter(test.signature)
        if err != nil {
            t.Errorf("test %d failed: error: %v", i, err)
            continue
        }
        result := s.String()
        if result != test.expected {
            t.Logf("got:\n%s", result)
            t.Logf("expected:\n%s", test.expected)
            t.Errorf("test %d failed: unexpected output", i)
        }
    }
}
