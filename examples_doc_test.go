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

func ExampleParseStruct_fromFile() {
    animal := must(morph.ParseStruct("examples_doc_test.go", nil, "Animal"))
    fmt.Println(animal)

    // output:
    // type Animal struct {
    //	Name string
    //	Born time.Time
    // }
}

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
            output.Converter = "$dest.$ = $src.$.Weight()"
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

    functionSignature := "($src.$type.$untitle $src.$type) To$dest.$type() $dest.$type"
    fmt.Println(must(morph.StructConverter(functionSignature, apple, orange)))

    // Output:
    // // ToOrange converts a value of type [Apple] to a value of type [Orange].
    // func (apple Apple) ToOrange() Orange {
    //	_out := Orange{}
    //
    //	// convert time.Time to int64
    //	_out.Picked = apple.Picked.UTC().Unix()
    //
    //	// convert time.Time to int64
    //	_out.LastEaten = apple.LastEaten.UTC().Unix()
    //
    //	// convert weight.Weight to int64
    //	_out.Weight = apple.Weight.Weight()
    //
    //	return _out
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
            output.Converter = "$dest.$ = weight.FromGrams($src.$)"
            output.Comment = ""
            emit(output)
        }

        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Converter = "$dest.$ = $src.$.Grams()" // rewrite the value
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

    const functionSignature = "($src.$type.$untitle $src.$type) $src.$(type)To$dest.$type($dest.$type.$untitle *$dest.$type)"

    fmt.Println(orange)
    fmt.Println(must(morph.StructConverter(functionSignature, orange, apple)))

    appleAgain := orange.Map(structmappers.Reverse)

    fmt.Println(appleAgain)
    fmt.Println(must(morph.StructConverter(functionSignature, apple, orange)))

    // Output:
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64 // weight in grams
    // }
    // // OrangeToApple converts a value of type [Orange] to a value of type [Apple].
    // func (orange Orange) OrangeToApple(apple *Apple) {
    //	_out := Apple{}
    //
    //	// convert int64 to time.Time
    //	_out.Picked = orange.Picked
    //
    //	// convert int64 to time.Time
    //	_out.LastEaten = orange.LastEaten
    //
    //	// convert int64 to weight.Weight
    //	_out.Weight = orange.Weight
    //
    //	*dest = _out
    // }
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    // }
    // // AppleToOrange converts a value of type [Apple] to a value of type [Orange].
    // func (apple Apple) AppleToOrange(orange *Orange) {
    //	_out := Orange{}
    //
    //	// convert time.Time to int64
    //	_out.Picked = apple.Picked.UTC().Unix()
    //
    //	// convert time.Time to int64
    //	_out.LastEaten = apple.LastEaten.UTC().Unix()
    //
    //	// convert weight.Weight to int64
    //	_out.Weight = apple.Weight.Grams()
    //
    //	*dest = _out
    // }
}
