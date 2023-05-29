package morph_test

import (
    "fmt"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fieldmappers"
    "github.com/tawesoft/morph/structmappers"
)

func must[X any](value X, err error) X {
    if err != nil { panic(err) }
    return value
}

func Example_applesToOranges() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
    Price     price.Price
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        int64ToWeight := func(input morph.Field, emit func(output morph.Field)) {
            output := input
            output.Type = "weight.Weight"
            output.Value = "weight.FromGrams($.$)"
            emit(output)
        }

        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Value = "$.$.Grams()" // rewrite the value
            output.Comment = "grams from weight.Weight"
            output.Reverse = fieldmappers.Compose(int64ToWeight, output.Reverse)
            emit(output)
        } else {
            emit(input)
        }
    }

    orange := apple.Map(
        structmappers.Rename("Orange"),
    ).MapFields(
        fieldmappers.TimeToInt64,
        WeightToInt64,
        // PriceToInt64 -- left as exercise for the reader
    )

    fmt.Println(apple)
    fmt.Println(orange)

    const converterFormat = "($from $From) To$To() $To"
    appleToOrange := must(orange.Converter(converterFormat))

    appleFromOrange := orange.Map(structmappers.Rename("Apple")).MapFields(fieldmappers.Reverse)
    orangeToApple := must(appleFromOrange.Converter(converterFormat))

    fmt.Println(appleToOrange)
    fmt.Println(orangeToApple)

    // output:
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64 // grams from weight.Weight
    //	Price     price.Price
    // }
    // // ToOrange converts [Apple] to [Orange].
    // func (apple Apple) ToOrange() Orange {
    //	return Orange{
    //		Picked:    apple.Picked.UTC().Unix(),
    //		LastEaten: apple.LastEaten.UTC().Unix(),
    //		Weight:    apple.Weight.Grams(),
    //		// Price is the zero value.
    //	}
    // }
    // // ToApple converts [Orange] to [Apple].
    // func (orange Orange) ToApple() Apple {
    //	return Apple{
    //		Picked:    time.Unix(orange.Picked, 0).UTC(),
    //		LastEaten: time.Unix(orange.LastEaten, 0).UTC(),
    //		Weight:    weight.FromGrams(orange.Weight),
    //		// Price is the zero value.
    //	}
    // }
}
