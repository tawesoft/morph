package morph_test

import (
    "fmt"

    "github.com/tawesoft/morph"
)

type Foo[X any] struct {
    Contains []X
}

type Insect int

func foo[X Insect](f Foo[X]) Foo[X] {
    return Foo[X]{Contains: []X{}}
}

func Example() {
    source := `
        package example

        import (
            "image/color"
            "time"
        )

        type Apple[X any] struct {
            Colour color.RGBA
            Picked time.Time
            LastEaten time.Time
            Contains []X
        }
    `

    apple, err := morph.ParseStruct("example.go", source, "Apple")
    if err != nil { panic(err) }

    definitionMorpher := func(name, Type, tag string, emit func(name, Type, tag string)) {
        if Type == "time.Time" {
            emit(name, "int64", tag) // epoc seconds
        } else {
            emit(name, Type, tag)
        }
    }

    appleToOrangeMorpher := func(name, Type, tag string, emit func(name, value string)) {
        if Type == "time.Time" {
            emit(name, "$.UTC().Unix()")
        } else {
            emit(name, "$")
        }
    }

    orange, err := morph.StructDefinition(apple, "Orange[X any]", definitionMorpher)
    if err != nil { panic(err) }

    sig := "appleToOrange[I Insect](a Apple[I]) Orange[I]"
    appleToOrange, err := morph.StructValue(apple, sig, appleToOrangeMorpher)
    if err != nil { panic(err) }

    fmt.Println(orange.String())
    fmt.Println(appleToOrange)

    // Output:
    // type Orange[X any] struct {
    //	Colour    color.RGBA
    //	Picked    int64
    //	LastEaten int64
    //	Contains  []X
    // }
    //
    // func appleToOrange[I Insect](a Apple[I]) Orange[I] {
    //	return Orange[I]{
    //		Colour:    a.Colour,
    //		Picked:    a.Picked.UTC().Unix(),
    //		LastEaten: a.LastEaten.UTC().Unix(),
    //		Contains:  a.Contains,
    //	}
    // }
}
