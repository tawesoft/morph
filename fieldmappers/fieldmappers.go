// Package fieldmappers provides helpful composable functions that implement
// [morph.FieldMapper] for mapping the fields between two structs using morph.
//
// A subpackage, [morph/fieldmappers/fieldops], provides additional
// field mappers that set the Comparer, Copier, and Orderer expressions on
// struct fields.
package fieldmappers

import (
    "strings"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fieldmappers/fieldops"
)

// Compose returns a new [morph.FieldMapper] that applies each of the given
// non-nil mappers, from left to right. Nil mappers are skipped.
func Compose(mappers ... morph.FieldMapper) morph.FieldMapper {
    return func(input morph.Field, emit func(output morph.Field)) {
        outputs := []morph.Field{input}
        catch := func(out morph.Field) {
            outputs = append(outputs, out)
        }
        for _, mapper := range mappers {
            if mapper == nil { continue }
            fields := outputs
            outputs = nil
            for _, in := range fields {
                emit2 := func(output morph.Field) {
                    output = output.Copy()
                    output.Rewrite(input)
                    catch(output)
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

// None is a [morph.FieldMapper] that deletes every input.
func None(input morph.Field, emit func(output morph.Field)) {
    // intentionally left empty
}

// DeleteNamed returns a new [morph.FieldMapper] that removes the named fields
// from a struct.
func DeleteNamed(names ... string) morph.FieldMapper {
    return Conditionally(FilterNamed(names...), None)
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

// FilterInv returns a filter that implements the inverse of the provided
// filter. Wherever the input filter would return true, the output filter
// instead returns false, and vice versa.
func FilterInv(filter func(input morph.Field) bool) func(input morph.Field) bool {
    return func(input morph.Field) bool {
        return !filter(input)
    }
}

// FilterNamed returns a filter that returns true for any field with a name
// matching any provided name argument.
func FilterNamed(names ... string) func(morph.Field) bool {
    // O(1)ish lookup
    nameMap := make(map[string]struct{})
    for _, name := range names {
        nameMap[name] = struct{}{}
    }
    return func(input morph.Field) bool {
        _, exists := nameMap[input.Name]
        return exists
    }
}

// FilterTypes returns a filter that returns true for any field with a type
// name matching any provided type name argument.
func FilterTypes(types ... string) func(morph.Field) bool {
    // O(1)ish lookup
    nameMap := make(map[string]struct{})
    for _, name := range types {
        nameMap[name] = struct{}{}
    }
    return func(input morph.Field) bool {
        _, exists := nameMap[input.Type]
        return exists
    }
}

// FilterSlices is a filter that returns true for any field with a type
// that is a slice.
func FilterSlices(input morph.Field) bool {
    return strings.HasPrefix(input.Type, "[]")
}

// Conditionally returns a new [morph.FieldMapper] that applies mapper
// to any field where the filter func returns true, or emits the field
// unchanged if the filter func returns false.
func Conditionally(filter func(morph.Field) bool, mapper morph.FieldMapper) morph.FieldMapper {
    return func(input morph.Field, emit func(output morph.Field)) {
        if filter(input) {
            mapper(input, emit)
        } else {
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
// and an int64 that used to be a time, this sets a Reverse method on output
// field. This allows [Reverse] to automatically perform the reverse mapping.
//
// The function sets appropriate Comparer, Copier, and Orderer expressions on
// the output field and on the reverse output field.
func TimeToInt64(input morph.Field, emit func(output morph.Field)) {
    if input.Type == "time.Time" {
        f := morph.Field{
            Name:    input.Name,
            Type:    "int64",
            Converter: "$dest.$ = $src.$.UTC().Unix()",
            Comment: "time in seconds since Unix epoch",
            Reverse: Compose(func(input2 morph.Field, emit2 func(output morph.Field)) {
                output := input2
                output.Type = "time.Time"
                output.Converter = "$dest.$ = time.Unix($src.$, 0).UTC()"
                output.Comment = input.Comment
                fieldops.Time(output, emit2)
            }, input.Reverse),
        }
        fieldops.Time(f, emit)
    } else {
        emit(input)
    }
}

// Reverse is a [morph.FieldMapper] that maps a mapped struct back to its
// original, to the extent that this is possible, by applying the reverse
// FieldMapper on each field.
func Reverse(input morph.Field, emit func(output morph.Field)) {
    if input.Reverse != nil {
        input.Reverse(input, emit)
    } else {
        emit(input)
    }
}
