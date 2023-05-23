// Package fieldmappers provides helpful composable functions that implement
// [morph.FieldMapper] for mapping the fields between two structs using morph.
package fieldmappers

import (
    "github.com/tawesoft/morph"
)

// Compose returns a new [morph.FieldMapper] that applies each of the given
// mappers, from left to right.
func Compose(mappers ... morph.FieldMapper) morph.FieldMapper {
    return func(input morph.Field, emit func(output morph.Field)) {
        outputs := []morph.Field{input}
        catch := func(out morph.Field) {
            outputs = append(outputs, out)
        }
        for _, mapper := range mappers {
            fields := outputs
            outputs = nil
            for _, in := range fields {
                emit2 := func(output morph.Field) {
                    catch(output.Rewrite(input))
                }
                mapper(in, emit2)
            }
        }
        for _, output := range outputs {
            emit(output)
        }
    }
}

// All is a [morph.FieldMapper] that emits every input unchanged.
func All(input morph.Field, emit func(output morph.Field)) {
    emit(input)
}

// DeleteNamed returns a new [morph.FieldMapper] that removes the named fields
// from a struct.
func DeleteNamed(names ... string) morph.FieldMapper {
    // O(1)ish lookup
    nameMap := make(map[string]struct{})
    for _, name := range names {
        nameMap[name] = struct{}{}
    }
    return func(input morph.Field, emit func(output morph.Field)) {
        if _, exists := nameMap[input.Name]; !exists {
            emit(input)
        }
    }
}

// Filter returns a new [morph.FieldMapper] that only emits fields where
// the provided filter function returns true.
func Filter(filter func(input morph.Field) bool) morph.FieldMapper {
    return func(input morph.Field, emit func(output morph.Field)) {
        if filter(input) {
            emit(input)
        }
    }
}

// StripComments is a [morph.FieldMapper] that strips all comments from each
// input field.
func StripComments(input morph.Field, emit func(output morph.Field)) {
    output := input
    output.Comment = ""
    emit(output)
}

// StripTags is a [morph.FieldMapper] that strips all struct tags from each
// input field.
func StripTags(input morph.Field, emit func(output morph.Field)) {
    output := input
    output.Tag = ""
    emit(output)
}

// TimeToInt64 is a [morph.FieldMapper] that converts any `time.Time` field
// to an `int64` field containing the time in seconds since the Unix epoch.
//
// As it is difficult to distinguish between an int64 that's just an integer,
// and an int64 that used to be a time, this adds "morph-reverse" field
// tags to the output field. This allows [Reverse] to automatically perform the
// reverse mapping.
func TimeToInt64(input morph.Field, emit func(output morph.Field)) {
    if input.Type == "time.Time" {
        f := morph.Field{
            Name:    input.Name,
            Type:    "int64",
            Value:   "$.$.UTC().Unix()",
            Comment: "time in seconds since Unix epoch",
            Reverse: func(input2 morph.Field, emit func(output morph.Field)) {
                input2.Type = "time.Time"
                input2.Value = "time.Unix($.$, 0).UTC()"
                input2.Comment = input.Comment
                emit(input2)
            },
        }
        emit(f)
    } else {
        emit(input)
    }
}

// Reverse is a [morph.FieldMapper] that maps a mapped struct back to its
// original, to the extent that this is possible, by examining the generated
// "morph-reverse" tags on a generated field.
func Reverse(input morph.Field, emit func(output morph.Field)) {
    if input.Reverse != nil {
        input.Reverse(input, emit)
    } else {
        emit(input)
    }
}
