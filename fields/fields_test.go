package fields_test

import (
    "testing"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fields"
)

func must[X any](x X, err error) X {
    if err != nil { panic(err) }
    return x
}

func Test(t *testing.T) {
    fsig := morph.FunctionSignature{
        Name:      "InputToOutput",
        Arguments: []morph.Field{{Name: "from", Type: "Input"}},
        Returns:   []morph.Field{{Type: "Output"}},
    }
    fsigReverse := morph.FunctionSignature{
        Name:      "OutputToInput",
        Arguments: []morph.Field{{Name: "from", Type: "Output"}},
        Returns:   []morph.Field{{Type: "Input"}},
    }
    tests := []struct {
        desc string
        input morph.Struct
        mapper morph.StructMapper
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
            mapper: fields.Compose(
                fields.DeleteNamed("A"),
                func(in morph.Field, emit func(morph.Field)) {
                    emit(in)
                    emit(morph.Field{Name: "$2", Type: "$", Value: "$.$"})
                },
                fields.DeleteNamed("B"),
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
            mapper: fields.TimeToInt64,
            expectedStruct: morph.Struct{
                Name:   "Output",
                Fields: []morph.Field{
                    {Name: "A", Type: "int"},
                    {
                        Name:    "B",
                        Type:    "int64",
                        Comment: "time in seconds since Unix epoch",
                        Tag:     `morph-reverse-type:"time.Time" morph-reverse-value:"time.Unix($.B, 0).UTC()"`,
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
            resultStruct, err := test.input.Struct("Output", test.mapper)
            if err != nil {
                t.Errorf("error: %s", err)
            } else {
                if resultStruct.String() != test.expectedStruct.String() {
                    t.Logf("got struct:\n%s", resultStruct)
                    t.Logf("expected struct:\n%s", test.expectedStruct)
                    t.Errorf("structs did not compare equal")
                }
            }

            resultFunc, err := test.input.Function(fsig.String(), test.mapper)
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
                resultReverseStruct, err := resultStruct.Struct("Input", fields.Reverse)
                if err != nil {
                    t.Errorf("error: %s", err)
                } else {
                    if resultReverseStruct.String() != test.expectedReverseStruct.String() {
                        t.Logf("got reverse struct:\n%s", resultReverseStruct)
                        t.Logf("expected reverse struct:\n%s", test.expectedReverseStruct)
                        t.Errorf("reverse structs did not compare equal")
                    }
                }
            }

            if test.expectedReverseFunc.Signature.Name != "" {
                resultReverseFunc, err := resultStruct.Function(fsigReverse.String(), fields.Reverse)
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
        Mapper morph.StructMapper
    }{
        {"All", fields.All}, // 0
        {"DeleteNamed", fields.DeleteNamed("FieldTwo")}, // 1
        {"Filter", fields.Filter(func (field morph.Field) bool { // 2
            return field.Type != "time.Time"
        })},
        {"StripTags", fields.StripTags}, // 3
        {"StripComments", fields.StripComments}, // 4
        {"Duplicate", func(input morph.Field, emit func(output morph.Field)) { // 5
            emit(input)
            input.Name = "$"
            emit(input)
        }},
    }

    f.Add(-1, -2, -3)
    f.Add(-1, 5, -2)
    f.Add(0, 1, 2)
    f.Add(0, 1, 2)
    f.Add(3, 4, 0)
    f.Add(4, 1, 0)
    f.Add(5, 2, 1)

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
                {Name: "FieldTwo", Type: "int", Value: "$.FieldOne"},
                {Name: "FieldThree", Type: "time.Time", Value: "time.Now().UTC()"},
            },
        }

        composedStructResult := must(input.Struct("output", fields.Compose(
            mappers[a].Mapper, mappers[b].Mapper, mappers[c].Mapper,
        )))

        sequentialStructResult := func(input morph.Struct) morph.Struct {
            x := must(input.Struct("output", mappers[a].Mapper))
            y := must(x.Struct("output", mappers[b].Mapper))
            z := must(y.Struct("output", mappers[c].Mapper))
            return z
        }(input)

        if composedStructResult.String() != sequentialStructResult.String() {
            t.Logf("composed struct: %v", composedStructResult)
            t.Logf("sequential struct: %v", sequentialStructResult)
            t.Errorf("structs do not match when composing mappers %s, %s, %s",
                mappers[a].Name, mappers[b].Name, mappers[c].Name,
            )
        }

        fsig := "InputToOutput(from Input) output"
        composedFuncResult := must(input.Function(fsig, fields.Compose(
            mappers[a].Mapper, mappers[b].Mapper, mappers[c].Mapper,
        )))

        sequentialFuncResult := func(input morph.Struct) morph.Function {
            x := must(input.Struct(input.Name, mappers[a].Mapper))
            y := must(x.Struct(input.Name, mappers[b].Mapper))
            z := must(y.Struct(input.Name, mappers[c].Mapper))
            return must(z.Function(fsig, fields.All))
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
