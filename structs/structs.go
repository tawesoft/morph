// Package structs implements some useful transformations from one struct
// to another, in a "functional options" style for the [morph.Struct.Copy]
// method.
//
// Each transformer receives a new copy that it is free to mutate.
package structs

import (
    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/internal"
)

// StripComment is a struct transformer that sets the struct's comment to the
// empty string.
func StripComment(s morph.Struct) morph.Struct {
    s.Comment = ""
    return s
}

// SetComment returns a new struct transformer that sets the struct's comment
// to the provided string.
func SetComment(comment string) func (in morph.Struct) morph.Struct {
    return func(s morph.Struct) morph.Struct {
        s.Comment = comment
        return s
    }
}

// AppendFields returns a new struct transformer that adds the given fields
// to the end of a struct's list of fields. Each [morph.Field.Value] field is
// ignored.
func AppendFields(fields []morph.Field) func (in morph.Struct) morph.Struct {
    fields = internal.Map(func (f morph.Field) morph.Field {
        f.Value = ""
        return f
    }, fields)
    return func(s morph.Struct) morph.Struct {
        if len(fields) == 0 { return s }
        s.Fields = append(s.Fields, fields...)
        return s
    }
}

// PrependFields returns a new struct transformer that adds the given fields
// to the start of a struct's list of fields. Each [morph.Field.Value] field is
// ignored.
func PrependFields(fields []morph.Field) func (in morph.Struct) morph.Struct {
    fields = internal.Map(func (f morph.Field) morph.Field {
        f.Value = ""
        return f
    }, fields)
    return func(s morph.Struct) morph.Struct {
        if len(fields) == 0 { return s }
        s.Fields = append(fields, s.Fields...)
        return s
    }
}
