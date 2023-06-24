package morph_test

import (
    "testing"

    "github.com/tawesoft/morph"
)

func TestStruct_Map(t *testing.T) {
    apple := morph.Struct{
        Name:   "Apple",
        Fields: []morph.Field{
            {
                Name:      "Picked",
                Type:      "time.Time",
            },
        },
    }
    // TODO
    apple=apple
}

/*
func morphAllFields(input morph.Field, emit func(output morph.Field)) {
    input.AppendComments("comment added by morph")
    if len(input.Tag) > 0 {
        input.AppendTags(`test:"morph"`)
    }
    emit(morph.Field{
        Name:    "$2",
        Type:    "maybe.M[$]",
        Converter:   "maybe.Some($.$)",
        Tag:     input.Tag,
        Comment: input.Comment,
    })
}
*/
