package morph_test

import (
    "fmt"
    "time"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fieldmappers"
    "github.com/tawesoft/morph/structmappers"
)

func must[X any](value X, err error) X {
    if err != nil { panic(err) }
    return value
}

func ExampleParseStruct_fromString() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
    Price     price.Price
}
`

    apple := must(morph.ParseStruct("test.go", source, "Apple"))
    fmt.Println(apple)

    // output:
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
}

type Animal struct {
    Name string
    Born time.Time
}

func ExampleParseStruct_from_file() {
    animal := must(morph.ParseStruct("examples_test.go", nil, "Animal"))
    fmt.Println(animal)

    // output:
    // type Animal struct {
    //	Name string
    //	Born time.Time
    // }
}

/*
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
*/

func ExampleFieldMapper() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
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
    )
    fmt.Println(orange)

    // Output:
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64
    // }
}

func ExampleStruct_Converter() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Value = "$.$.Weight()"
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
    )

    functionSignature := "$FromTo$To($from $From) $To"
    fmt.Println(must(orange.Converter(functionSignature)))

    // Output:
    // // AppleToOrange converts [Apple] to [Orange].
    // func AppleToOrange(apple Apple) Orange {
    //	return Orange{
    //		Picked:    apple.Picked.UTC().Unix(),
    //		LastEaten: apple.LastEaten.UTC().Unix(),
    //		Weight:    apple.Weight.Weight(),
    //	}
    // }
}


func ExampleStruct_Converter_reverse() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        reverse := func(input morph.Field, emit func(output morph.Field)) {
            output := input
            output.Type = "weight.Weight"
            output.Value = "weight.FromGrams($.$)"
            output.Comment = ""
            emit(output)
        }

        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Value = "$.$.Grams()" // rewrite the value
            output.Reverse = fieldmappers.Compose(reverse, output.Reverse)
            output.Comment = "weight in grams"
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
    )

    const functionSignature = "$FromTo$To($from $From) $To"

    fmt.Println(orange)
    fmt.Println(must(orange.Converter(functionSignature)))

    appleAgain := orange.Map(structmappers.Reverse)

    fmt.Println(appleAgain)
    fmt.Println(must(appleAgain.Converter(functionSignature)))

    // Output:
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64 // weight in grams
    // }
    // // AppleToOrange converts [Apple] to [Orange].
    // func AppleToOrange(apple Apple) Orange {
    //	return Orange{
    //		Picked:    apple.Picked.UTC().Unix(),
    //		LastEaten: apple.LastEaten.UTC().Unix(),
    //		Weight:    apple.Weight.Grams(),
    //	}
    // }
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    // }
    // // OrangeToApple converts [Orange] to [Apple].
    // func OrangeToApple(orange Orange) Apple {
    //	return Apple{
    //		Picked:    time.Unix(orange.Picked, 0).UTC(),
    //		LastEaten: time.Unix(orange.LastEaten, 0).UTC(),
    //		Weight:    weight.FromGrams(orange.Weight),
    //	}
    // }
}
