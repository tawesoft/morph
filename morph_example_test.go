package morph_test

import (
    "fmt"

    "github.com/tawesoft/morph"
)

func Example() {
    // In this example, we have an "Apple" struct that we'd like to
    // automatically create an "Orange" struct from, with functions to
    // map between them.

    //  First we take our existing source code, and parse it for the Apple
    //  struct.
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
    fmt.Println(apple.String())

    // Then we describe how to create an Orange struct based on the Apple
    // struct. For this example, we're going to copy every field across
    // unchanged, except we're going to use integers instead of the Go time
    // type, perhaps because the Orange is being shared with other interfaces
    // that only understand Unix Epoch time.
    appleStructToOrangeStructMorpher := func(name, Type, tag string, emit func(name, Type, tag string)) {
        if Type == "time.Time" {
            emit(name, "int64", tag) // epoch seconds
        } else {
            emit(name, Type, tag)
        }
    }
    orange, err := apple.StructFunc("Orange[X any]", appleStructToOrangeStructMorpher)
    if err != nil { panic(err) }
    fmt.Println(orange.String())

    // This generates:
    // type Orange[X any] struct {
    //	Colour    color.RGBA
    //	Picked    int64
    //	LastEaten int64
    //	Contains  []X
    // }

    // Now, we want to generate a function that can map any Apple value into an
    // Orange value. The function signature can be almost any form, but it must
    // have at least one receiver or input argument of the source struct type
    // (here, Apple) and must return exactly one value of the target type (here,
    // Orange). Here, "$" is shorthand for the input field.
    sigA2O := "appleToOrange[I Insect](a Apple[I]) Orange[I]"
    appleToOrangeMorpher := func(name, Type, tag string, emit func(name, value string)) {
        if Type == "time.Time" {
            emit(name, "$.UTC().Unix()")
        } else {
            emit(name, "$")
        }
    }
    appleToOrange, err := apple.FunctionFunc(sigA2O, appleToOrangeMorpher)
    if err != nil { panic(err) }
    fmt.Println(appleToOrange)

    // This generates:
    // func appleToOrange[I Insect](a Apple[I]) Orange[I] {
    //	return Orange[I]{
    //		Colour:    a.Colour,
    //		Picked:    a.Picked.UTC().Unix(),
    //		LastEaten: a.LastEaten.UTC().Unix(),
    //		Contains:  a.Contains,
    //	}
    // }

    // Here's the reverse, which maps any Orange value back into an Apple value,
    // with the function signature defining a method on an Orange and using pointers.
    sigO2A := "(o *Orange[X]) ToApple() *Apple[X]"
    orangeToAppleMorpher := func(name, Type, tag string, emit func(name, value string)) {
        switch name {
            case "Picked": fallthrough
            case "LastEaten":
                emit(name, "time.Unix($, 0).UTC()")
            default:
                emit(name, "$")
        }
    }
    orangeToApple, err := orange.FunctionFunc(sigO2A, orangeToAppleMorpher)
    if err != nil { panic(err) }
    fmt.Println(orangeToApple)

    // This generates:
    // func (o *Orange[X]) ToApple() *Apple[X] {
    //	return &Apple[X]{
    //		Colour:    o.Colour,
    //		Picked:    time.Unix(o.Picked, 0).UTC(),
    //		LastEaten: time.Unix(o.LastEaten, 0).UTC(),
    //		Contains:  o.Contains,
    //	}
    // }

    // Here's the full output:

    // Output:
    // type Apple[X any] struct {
    //	Colour    color.RGBA
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Contains  []X
    // }
    //
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
    //
    // func (o *Orange[X]) ToApple() *Apple[X] {
    //	return &Apple[X]{
    //		Colour:    o.Colour,
    //		Picked:    time.Unix(o.Picked, 0).UTC(),
    //		LastEaten: time.Unix(o.LastEaten, 0).UTC(),
    //		Contains:  o.Contains,
    //	}
    // }
}
