package fieldmappers_test

import (
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fieldmappers"
    "github.com/tawesoft/morph/internal"
    "github.com/tawesoft/morph/structmappers"
)

func Test(t *testing.T) {
    fsig := morph.FunctionSignature{
        Name:      "InputToOutput",
        Comment:   "InputToOutput converts [Input] to [Output].",
        Arguments: []morph.Field{{Name: "from", Type: "Input"}},
        Returns:   []morph.Field{{Type: "Output"}},
    }
    fsigReverse := morph.FunctionSignature{
        Name:      "OutputToInput",
        Comment:   "OutputToInput converts [Output] to [Input].",
        Arguments: []morph.Field{{Name: "from", Type: "Output"}},
        Returns:   []morph.Field{{Type: "Input"}},
    }
    tests := []struct {
        desc string
        input morph.Struct
        mapper morph.FieldMapper
        expectedStruct morph.Struct
        expectedFunc morph.Function
        expectedReverseStruct morph.Struct
        expectedReverseFunc morph.Function
    }{
        {
            desc: "fields.Compose",
            input: morph.Struct{
                Name:   "Input",
                Fields: []morph.Field{
                    {Name: "A", Type: "int"},
                    {Name: "B", Type: "int"},
                    {Name: "C", Type: "int"},
                },
            },
            mapper: fieldmappers.Compose(
                fieldmappers.DeleteNamed("A"),
                func(in morph.Field, emit func(morph.Field)) {
                    emit(in)
                    emit(morph.Field{Name: "$2", Type: "$", Value: "$.$"})
                },
                fieldmappers.DeleteNamed("B"),
            ),
            expectedStruct: morph.Struct{
                Name:   "Output",
                Fields: []morph.Field{
                    {Name: "B2", Type: "int", Value: "$.B",},
                    {Name: "C",  Type: "int"},
                    {Name: "C2", Type: "int", Value: "$.C",},
                },
            },
            expectedFunc: morph.Function{
                Signature: fsig,
                Body: `    return Output{
        B2: from.B,
        // C is the zero value.
        C2: from.C,
    }`,
            },
        },
        {
            desc: "fields.TimeToInt64",
            input: morph.Struct{
                Name:   "Input",
                Fields: []morph.Field{
                    {Name: "A", Type: "int"},
                    {Name: "B", Type: "time.Time"},
                    {Name: "C", Type: "int"},
                },
            },
            mapper: fieldmappers.TimeToInt64,
            expectedStruct: morph.Struct{
                Name:   "Output",
                Fields: []morph.Field{
                    {Name: "A", Type: "int"},
                    {
                        Name:    "B",
                        Type:    "int64",
                        Comment: "time in seconds since Unix epoch",
                    },
                    {Name: "C",  Type: "int"},
                },
            },
            expectedFunc: morph.Function{
                Signature: fsig,
                Body: `    return Output{
        // A is the zero value.
        B: from.B.UTC().Unix(),
        // C is the zero value.
    }`,
            },
            expectedReverseStruct: morph.Struct{
                Name:   "Input",
                Fields: []morph.Field{
                    {Name: "A", Type: "int"},
                    {Name: "B",  Type: "time.Time"},
                    {Name: "C",  Type: "int"},
                },
            },
            expectedReverseFunc: morph.Function{
                Signature: fsigReverse,
                Body: `    return Input{
        // A is the zero value.
        B: time.Unix(from.B, 0).UTC(),
        // C is the zero value.
    }`,
            },
        },
    }

    for _, test := range tests {
        t.Run(test.desc, func(t *testing.T) {
            resultStruct := test.input.
                MapFields(test.mapper).
                Map(structmappers.Rename(test.expectedStruct.Name))

            if resultStruct.String() != test.expectedStruct.String() {
                t.Logf("got struct:\n%s", resultStruct)
                t.Logf("expected struct:\n%s", test.expectedStruct)
                t.Errorf("structs did not compare equal")
            }

            resultFunc, err := resultStruct.Converter(fsig.String())
            if err != nil {
                t.Errorf("error: %s", err)
            } else {
                if resultFunc.String() != test.expectedFunc.String() {
                    t.Logf("got func:\n%s", resultFunc)
                    t.Logf("expected func:\n%s", test.expectedFunc)
                    t.Errorf("funcs did not compare equal")
                }
            }

            if test.expectedReverseStruct.Name != "" {
                resultReverseStruct := resultStruct.
                    MapFields(fieldmappers.Reverse).
                    Map(structmappers.Rename(test.expectedReverseStruct.Name))
                if resultReverseStruct.String() != test.expectedReverseStruct.String() {
                    t.Logf("got reverse struct:\n%s", resultReverseStruct)
                    t.Logf("expected reverse struct:\n%s", test.expectedReverseStruct)
                    t.Errorf("reverse structs did not compare equal")
                }

                resultReverseFunc, err := resultReverseStruct.Converter(fsigReverse.String())
                if err != nil {
                    t.Errorf("error: %s", err)
                } else {
                    if resultReverseFunc.String() != test.expectedReverseFunc.String() {
                        t.Logf("got reverse func:\n%s", resultReverseFunc)
                        t.Logf("expected reverse func:\n%s", test.expectedReverseFunc)
                        t.Errorf("reverse funcs did not compare equal")
                    }
                }
            }
        })
    }
}

// FuzzCompose tries composing multiple mappers in random orders, and ensures
// that they give the same result as applying the mappers in that order
// one-by-one.
//
// TODO there are few enough combinations we could just enumerate them all...
func FuzzCompose(f *testing.F) {
    // Note this array is append only as otherwise the seed corpus will become
    // invalid.
    mappers := []struct{
        Name string
        Mapper morph.FieldMapper
    }{
        {"All", fieldmappers.All},                             // 0
        {"DeleteNamed", fieldmappers.DeleteNamed("FieldTwo")}, // 1
        {"Filter", fieldmappers.Filter(func (field morph.Field) bool { // 2
            return field.Type != "time.Time"
        })},
        {"StripTags", fieldmappers.StripTags},         // 3
        {"StripComments", fieldmappers.StripComments}, // 4
        {"Duplicate", func(input morph.Field, emit func(output morph.Field)) { // 5
            emit(input)
            input.Name = "$"
            emit(input)
        }},
        {"TimeToInt64", fieldmappers.TimeToInt64}, // 6
        {"Reverse", fieldmappers.Reverse},         // 7
    }

    f.Add(-1, -2, -3)
    f.Add(-1, 5, -2)
    f.Add(0, 1, 2)
    f.Add(0, 1, 2)
    f.Add(3, 4, 0)
    f.Add(4, 1, 0)
    f.Add(5, 2, 1)
    f.Add(0, 6, 6)
    f.Add(2, 6, 0)
    f.Add(2, 6, 2)
    f.Add(6, 7, 6)
    f.Add(7, 7, 7)

    f.Fuzz(func (t *testing.T, a, b, c int) {
        if (a >= len(mappers)) { return }
        if (b >= len(mappers)) { return }
        if (c >= len(mappers)) { return }
        if (a == b) || (b == c) || (c == a) { return }
        if a < 0 { a = 0 }
        if b < 0 { b = 0 }
        if c < 0 { c = 0 }

        input := morph.Struct{
            Name:       "Input",
            TypeParams: nil,
            Fields:     []morph.Field{
                {
                    Name:    "FieldOne",
                    Type:    "int",
                    Value:   "111",
                    Tag:     `tag:"field1"`,
                    Comment: "this is field one",
                },
                {Name: "FieldTwo", Type: "int"},
                {Name: "FieldThree", Type: "time.Time"},
            },
        }

        composedStructResult := input.MapFields(fieldmappers.Compose(
            mappers[a].Mapper, mappers[b].Mapper, mappers[c].Mapper,
        )).Map(structmappers.Rename("Output"))

        sequentialStructResult := func(input morph.Struct) morph.Struct {
            x := input.MapFields(mappers[a].Mapper)
            y := x.MapFields(mappers[b].Mapper)
            z := y.MapFields(mappers[c].Mapper)
            return z.Map(structmappers.Rename("Output"))
        }(input)

        if composedStructResult.String() != sequentialStructResult.String() {
            t.Logf("composed struct: %v", composedStructResult)
            t.Logf("sequential struct: %v", sequentialStructResult)
            t.Errorf("structs do not match when composing mappers %s, %s, %s",
                mappers[a].Name, mappers[b].Name, mappers[c].Name,
            )
        }

        fsig := "InputToOutput(from Input) Output"
        composedFuncResult := internal.Must(input.
            MapFields(fieldmappers.Compose(
                mappers[a].Mapper, mappers[b].Mapper, mappers[c].Mapper,
            )).Map(structmappers.Rename("Output")).
            Converter(fsig))

        sequentialFuncResult := func(input morph.Struct) morph.Function {
            x := input.MapFields(mappers[a].Mapper)
            y := x.MapFields(mappers[b].Mapper)
            z := y.MapFields(mappers[c].Mapper)
            w := z.Map(structmappers.Rename("Output"))
            return internal.Must(w.Converter(fsig))
        }(input)

        if composedFuncResult.String() != sequentialFuncResult.String() {
            t.Logf("composed func: %v", composedFuncResult)
            t.Logf("sequential func: %v", sequentialFuncResult)
            t.Errorf("funcs do not match when composing mappers %s, %s, %s",
                mappers[a].Name, mappers[b].Name, mappers[c].Name,
            )
        }
    })
}
