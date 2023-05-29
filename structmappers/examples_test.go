package structmappers_test

import (
    "fmt"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/structmappers"
)

func must[X any](value X, err error) X {
    if err != nil { panic(err) }
    return value
}

func ExampleRename() {
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
    orange := apple.Map(structmappers.Rename("Orange"))
    fmt.Println(orange)

    appleAgain := orange.Map(structmappers.Reverse)
    fmt.Println(appleAgain)

    // output:
    // type Orange struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
}
