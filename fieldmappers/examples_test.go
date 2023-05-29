package fieldmappers_test

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

func ExampleTimeToInt64() {
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
    orange := apple.Map(
        structmappers.Rename("Orange"),
    ).MapFields(
        fieldmappers.TimeToInt64,
    )
    fmt.Println(orange)

    // output:
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
}
