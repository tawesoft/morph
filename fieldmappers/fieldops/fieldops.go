// Package fieldops implements morph FieldMappers that set appropriate
// Comparer, Copier, and Orderer expressions on morph Fields.
//
// These expressions are used to implement custom equality, copies, and sorting
// orders in code generated by [morph.Comparer], [morph.Copier], and
// [morph.Orderer].
package fieldops

import (
    "github.com/tawesoft/morph"
)

// Time sets appropriate expressions on fields of type [time.Time].
func Time(in morph.Field, emit func(out morph.Field)) {
    if in.Type == "time.Time" {
        out := in
        out.Comparer = "$a.$.Equals($b.$)"
        out.Copier = ""
        out.Orderer = "$b.$.After($a.$)"
        emit(out)
    } else {
        emit(in)
    }
}
