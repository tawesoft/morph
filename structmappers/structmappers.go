// Package structmappers implements some useful transformations from one struct
// to another, in a "functional options" style for the [morph.Struct.Map]
// method.
//
// Each transformer receives a new copy that it is free to mutate.
package structmappers

import (
    "strings"

    "github.com/tawesoft/morph"
)

// Compose returns a new [morph.StructMapper] that applies each of the given
// non-nil mappers, from left to right. Nil mappers are skipped.
func Compose(mappers ... morph.StructMapper) morph.StructMapper {
    mappers = append([]morph.StructMapper{}, mappers...)
    return func(in morph.Struct) morph.Struct {
        for _, mapper := range mappers {
            if mapper == nil { continue }
            in = mapper(in)
        }
        return in
    }
}

// StripComment is a [morph.StructMapper] that sets the struct's comment to the
// empty string.
func StripComment(s morph.Struct) morph.Struct {
    s.Comment = ""
    return s
}

// SetComment returns a new [morph.StructMapper] that sets the struct's comment
// to the provided string.
func SetComment(comment string) morph.StructMapper {
    return func(s morph.Struct) morph.Struct {
        s.Comment = comment
        return s
    }
}

// Rename returns a new [morph.StructMapper] that sets the struct's name
// to the provided string and sets the struct's From field to its original
// name.
//
// In the provided name, the token "$" is rewritten to the existing name. For
// example, Rename("$Xml") on a struct named "Foo" maps to a struct named
// "FooXml" with its From field set to Foo.
//
// Rename is reversible with [Reverse].
func Rename(name string) morph.StructMapper {
    return func(s morph.Struct) morph.Struct {
        oldName := s.Name
        s.From = s.Name
        s.Name = strings.ReplaceAll(name, "$", s.Name)
        s.Reverse = Compose(func (in morph.Struct) morph.Struct {
            out := in
            out.Name = oldName
            out.From = in.Name
            return out
        }, s.Reverse)
        return s
    }
}

// AppendFields returns a new [morph.StructMapper] that adds the given fields
// to the end of a struct's list of fields.
//
// AppendFields is reversible with [Reverse].
func AppendFields(fields []morph.Field) morph.StructMapper {
    return func(s morph.Struct) morph.Struct {
        if len(fields) == 0 { return s }
        start := len(s.Fields)
        end := start + len(fields)
        s.Fields = append(s.Fields, fields...)

        s.Reverse = Compose(func (in morph.Struct) morph.Struct {
            out := in
            out.Fields = out.Fields[start:end]
            return out
        }, s.Reverse)
        return s
    }
}

// PrependFields returns a new [morph.StructMapper] that inserts the given
// fields at the start of a struct's list of fields.
//
// PrependFields is reversible with [Reverse].
func PrependFields(fields []morph.Field) func (in morph.Struct) morph.Struct {
    return func(s morph.Struct) morph.Struct {
        if len(fields) == 0 { return s }
        end := len(s.Fields)
        s.Fields = append(fields, s.Fields...)

        s.Reverse = Compose(func (in morph.Struct) morph.Struct {
            out := in
            out.Fields = out.Fields[0:end]
            return out
        }, s.Reverse)
        return s
    }
}

// Reverse is a [morph.StructMapper] that maps a mapped struct back to its
// original, to the extent that this is possible, by applying the reverse
// StructMapper.
func Reverse(input morph.Struct) morph.Struct {
    if input.Reverse != nil {
        return input.Reverse(input.Copy())
    } else {
        return input
    }
}
