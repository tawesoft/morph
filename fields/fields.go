// Package fields provides helpful composable functions for mapping the fields
// between two structs using morph.
package fields

import (
    "strings"

    "github.com/tawesoft/morph"
)

// AppendTags returns a new struct tag string with the given tags appended
// with a space separator, as is the convention.
//
// Each tag in the tags list to be appended should be a single key:value pair,
// but the tag to be appended to can be a full list of pairs.
//
// If a tag in the tags list to be appended is already present in the original
// struct tag string, it is not appended.
//
// If any tags do not have the conventional format, the value returned
// is unspecified.
//
// Note that unlike the Go parser, struct tag strings in morph do not include
// the literal enclosing quote marks around the list of tags.
func AppendTags(tag string, tags ... string) string {
    elements := []string{tag}
    for _, t := range tags {
        key, _, _, ok := morph.NextTagPair(t)
        if !ok { continue }
        _, exists := morph.LookupTag(tag, key)
        if exists { continue }
        elements = append(elements, t)
    }
    return strings.Join(elements, " ")
}

// AppendComments returns a new comment string with the given comments appended
// with a newline separator.
//
// Note that unlike the Go parser, comment strings in morph do not have a
// trailing new line. Comments also do not have their leading "//", or "/*"
// "*/" marks.
func AppendComments(comment string, comments ... string) string {
    elements := append([]string{comment}, comments...)
    return strings.Join(elements, "\n")
}

// Compose returns a new [morph.Mapper] that applies each of the given
// mappers, from left to right.
func Compose(mappers ... morph.Mapper) morph.Mapper {
    return func(source morph.Struct, input morph.Field, emit func(output morph.Field)) {
        outputs := []morph.Field{input}
        catch := func(out morph.Field) {
            outputs = append(outputs, out)
        }
        for _, mapper := range mappers {
            s := source.Copy()
            s.Fields = append([]morph.Field{}, outputs...)
            outputs = nil
            for _, in := range s.Fields {
                mapper(s, in, catch)
            }
        }
        for _, output := range outputs {
            emit(output)
        }
    }
}

// Append returns a new [morph.Mapper] that adds the provided fields to
// the end of a struct.
func Append(fields ... morph.Field) morph.Mapper {
    return func(source morph.Struct, input morph.Field, emit func(output morph.Field)) {
        last := len(source.Fields) - 1
        if input.Name == source.Fields[last].Name {
            for _, field := range fields {
                field.Tag = "`from:\"nil\"`"
                emit(field)
            }
        }
        emit(input)
    }
}

// Prepend returns a new [morph.Mapper] that adds the provided fields to
// the start of a struct.
func Prepend(fields ... morph.Field) morph.Mapper {
    return func(source morph.Struct, input morph.Field, emit func(output morph.Field)) {
        if input.Name == source.Fields[0].Name {
            for _, field := range fields {
                field.Tag = "`from:\"nil\"`"
                emit(field)
            }
        }
        emit(input)
    }
}

// DeleteNamed returns a new [morph.Mapper] that removes the named fields
// from a struct.
func DeleteNamed(names ... string) morph.Mapper {
    // O(1)ish lookup
    nameMap := make(map[string]struct{})
    for _, name := range names {
        nameMap[name] = struct{}{}
    }
    return func(_ morph.Struct, input morph.Field, emit func(output morph.Field)) {
        if _, exists := nameMap[input.Name]; !exists {
            emit(input)
        }
    }
}

// Filter returns a new [morph.Mapper] that only emits fields where
// the provided filter function returns true.
func Filter(filter func(input morph.Field) bool) morph.Mapper {
    return func(_ morph.Struct, input morph.Field, emit func(output morph.Field)) {
        if filter(input) {
            emit(input)
        }
    }
}

// StripComments is a [morph.Mapper] that strips all comments from each input
// field.
func StripComments(_ morph.Struct, input morph.Field, emit func(output morph.Field)) {
    output := input
    output.Comment = ""
    emit(output)
}

// StripTags is a [morph.Mapper] that strips all struct tags from each input
// field.
func StripTags(_ morph.Struct, input morph.Field, emit func(output morph.Field)) {
    output := input
    output.Tag = ""
    emit(output)
}

// TimeToInt64 is a [morph.Mapper] that converts any `time.Time` field
// to an `int64` field containing the time in seconds since the Unix epoch.
//
// As it is difficult to distinguish between an int64 that's just an integer,
// and an int64 that used to be a time, this adds "from" and "reverse" field
// tags to the output field allowing [Reverse] to
// automatically perform the reverse mapping.
func TimeToInt64(_ morph.Struct, input morph.Field, emit func(output morph.Field)) {
    if input.Type == "time.Time" {
        emit(morph.Field{
            Name:    input.Name,
            Type:    "int64",
            Value:   "$.UTC().Unix()",
            Tag:     AppendTags(input.Tag, `from:"time.Time"`, `reverse:"time.Unix($, 0).UTC()"`),
            Comment: AppendComments(input.Comment, "time in seconds since Unix epoch"),
        })
    } else {
        emit(input)
    }
}

// Reverse is a [morph.Mapper] that maps a mapped struct back to its original,
// to the extent that this is possible, by using the "from" and "reverse" tags
// on a generated field.
//
// In the "from" tag the string "nil" represents a field that has been created
// "from nothing" and the reverse operation is to delete it.
//
// TODO not implemented
func Reverse(_ morph.Struct, input morph.Field, emit func(output morph.Field)) {
}

// Only constructs a new [morph.Mapper] that applies the provided mapper only
// to fields matching the given filter. Fields not matching the filter are instead
// emitted unchanged.
//
// TODO not implemented
func Only(filter func(morph.Field)) morph.Mapper {
    return nil
}
