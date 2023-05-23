// Package structs implements some useful transformations from one struct
// to another, in a "functional options" style for the [morph.Struct.Map]
// method.
//
// Each transformer receives a new copy that it is free to mutate.
package structs

import (
    "strings"

    "github.com/tawesoft/morph"
)

// Compose returns a new [morph.StructMapper] that applies each of the given
// mappers, from left to right.
func Compose(mappers ... morph.StructMapper) morph.StructMapper {
    return func(in morph.Struct) morph.Struct {
        for _, mapper := range mappers {
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
func Rename(name string) morph.StructMapper {
    return func(s morph.Struct) morph.Struct {
        s.From = s.Name
        s.Name = strings.ReplaceAll(name, "$", s.Name)
        return s
    }
}

// AppendFields returns a new [morph.StructMapper] that adds the given fields
// to the end of a struct's list of fields.
func AppendFields(fields []morph.Field) morph.StructMapper {
    return func(s morph.Struct) morph.Struct {
        if len(fields) == 0 { return s }
        s.Fields = append(s.Fields, fields...)
        return s
    }
}

// PrependFields returns a new [morph.StructMapper] that inserts the given
// fields at the start of a struct's list of fields.
func PrependFields(fields []morph.Field) func (in morph.Struct) morph.Struct {
    return func(s morph.Struct) morph.Struct {
        if len(fields) == 0 { return s }
        s.Fields = append(fields, s.Fields...)
        return s
    }
}