![Morph](morph.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/tawesoft/morph#section-documentation.svg)](https://pkg.go.dev/github.com/tawesoft/morph#section-documentation)

Morph
=====

Morph is a small Go code generator that generates code to map between
structs and call functions in different ways.

 - without runtime reflection.

 - without stuffing a new domain-specific language into struct field tags.

 - with a simple, fully programmable mapping described in native Go code.

 - where you can map to existing struct types, or use Morph to automatically
   generate new struct types.

 - with full support for generics.

Example: Mapping between struct types
-------------------------------------

Take the source code for an example struct, `Apple`:

```go
type Apple struct {
    Colour    color.RGBA
    Picked    time.Time
    LastEaten time.Time
}
```

In this example, we are going to create a new `Orange` struct. It is going 
to have the same fields as the `Apple` struct, except it will store time as 
integer epoch seconds instead.

We define two functions. The first derives the `Orange` struct from an `Apple`
struct. The second describes how to turn an `Apple` value into an `Orange`
value.

```go
func(name, Type, tag string, emit func(name, Type, tag string)) {
    if Type == "time.Time" {
        emit(name, "int64", tag) // epoch seconds
    } else {
        emit(name, Type, tag) // unchanged
    }
}

func(name, Type, tag string, emit func(name, value string))
    if Type == "time.Time" {
        emit(name, "$.UTC().Unix()")
    } else {
        emit(name, "$")
    }
}
```

These happen to be general purpose enough that we could use them on other
types, too. Regardless, these functions can be used to generate 
the following Go source code from our `Apple`:

```go
type Orange struct {
    Colour    color.RGBA
    Picked    int64
    LastEaten int64
}

func appleToOrange(a Apple) Orange {
    return Orange{
        Colour:    a.Colour,
        Picked:    a.Picked.UTC().Unix(),
        LastEaten: a.LastEaten.UTC().Unix(),
        Contains:  a.Contains,
    }
}
```

The [examples](https://pkg.go.dev/github.com/tawesoft/morph#pkg-examples)
demonstrate how to achieve this in more detail.


Example: Manipulating a function's form
---------------------------------------

Take the source code for an example function, `Divide`:

```go
// Divide returns a divided by b. If b is zero, returns an error.
func Divide(a, b float64) (float64, error) {
    if b == 0.0 {
        return 0, fmt.Errorf("error: can't divide %f by zero", a)
    } else {
        return (a / b), nil
    }
}
```

Morph can parse this and derive other forms of this function that partially
apply arguments, wrap inputs or results, or operate as a method, by wrapping
the original `Divide` function. It can generate Go source code like so:

```go
// Divider partially applies [Divide] with a constant divisor.
func Divider(b float64) func (a float64) (float64, error) { ... }

// DividePromise returns a promise to call [Divide] on the provided arguments. 
func DividePromise(a, b float64) func () (float64, error) { ... }

// DivideResult returns the result of [Divide] on the provided arguments,
// collected into a Result type.
func DivideResult(a, b float64) Result.R[float64] { ... }

// Decimal converts a fraction such as
// Fraction{Numerator: 1, Denominator: 3}` into a result such as 2.3333.
func (f Fraction) Decimal() (float64, error) { ... }
```

The function internals are elided for the sake of demonstration, but are
generally thin wrappers around a call to the original function.

The [examples](https://pkg.go.dev/github.com/tawesoft/morph#pkg-examples)
demonstrate how to achieve this in more detail.


Usage
--------------

You use morph as a library:

```go
import "github.com/tawesoft/morph"
```

Then simply `go run` something that uses morph to write to a file.


Security Model
--------------

WARNING: It is assumed that all inputs are trusted. DO NOT accept arbitrary
input from untrusted sources under any circumstances, as this will parse
and generate arbitrary code.
