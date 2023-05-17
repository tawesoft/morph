package morph_test

import (
    "fmt"

    "github.com/tawesoft/morph"
)

func Example_Struct() {
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


func Example_Function() {
    // In this example, we take a function and use it to automatically
    // construct new related functions of different forms.

    // Here's a simple function to return a divide by b, returning an error
    // it the caller attempts to divide by zero.
    source := `
    // Divide returns a divided by b. If b is zero, returns an error.
    func Divide(a float64, b float64) (float64, error) {
        if b == 0.0 {
            return 0, fmt.Errorf("error: can't divide %f by zero", a)
        } else {
            return (a / b), nil
        }
    }
    `

    // First we take our existing source code, and parse it for the divide
    // function:
    divide := morph.ParseFunc("example.go", source, "Divide")

    // For our first example, let's support partial application by generating
    // the code for a function that returns a version of divide bound with a
    // constant divisor.
    dividePartial := divide.New("Divider").BindRight(1)
    fmt.Println(dividePartial)

    // This generates:
    // func Divider(b float64) func(a float64) (float64, error) {
    // 	return func(a float64) (float64, error) {
    // 		return Divide(a, b)
    // }
    //
    // This can be used like:
    // half := Divider(2)
    // fmt.Println(half(10)) // prints 5

    // Similarly, we can take this further create a function that implements
    // a "Promise" to perform a divide operation at a later date.
    dividePromise := divide.New("DividePromise").BindAll()
    fmt.Println(dividePromise)

    // This generates:
    // func DividePromise(a float64, b float64) func() (float64, error) {
    // 	return func() (float64, error) {
    // 		return Divide(a, b)
    // }
    //
    // This can be used like:
    // scorePromise := DividePromise(10, 2)
    // // some time later...
    // score := scorePromise()

    // As another example, let's manipulate the return type of divide, for
    // example to use a Result type that combines a (result, error) tuple into
    // a single return value:
    divideResult := divide.New("DivideResult").WrapResult("result.R[float64]", "result.New($)")
    fmt.Println(divideResult)

    // This generates:
    // func DivideResult(a float64, b float64) result.R[float64] {
    // 	return result.New(Divide(a, b))
    // }
}
