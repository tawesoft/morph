![Morph](morph.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/tawesoft/morph#section-documentation.svg)](https://pkg.go.dev/github.com/tawesoft/morph#section-documentation)

Morph
=====

Morph is a small Go code generator that makes it easy to map between
Go structs, automatically derive new Go structs, and wrap functions in 
different ways.

Features:

 - fully programmable (in native Go).

 - full support for generics.

 - no runtime reflection.

 - no ugly new struct field tags to learn.

 - elegant and extremely composable mappings.

 - big library of done-for-you mappings and helpers:
   * [fields] for morphing structs
   * ***TODO*** funcs for wrapping functions


Quick Examples
--------------

### Mapping between structs

Take the source code for an example struct, `Apple`:

```go
type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    custom.Grams
    Price     custom.Price
}
```

Let's say we want a version where every field is an integer, for example to 
serialise it to a binary format on disk, or to store it in a database.

Morph can quickly let us generate:

```go
type Orange struct {
    Picked    int64 // time in seconds since Unix epoch
    LastEaten int64 // time in seconds since Unix epoch
    Weight    int32 // weight in grams
    Price     int32 // price in pence
}

func appleToOrange(a Apple) Orange {
    return Orange{
        Picked:    a.Picked.UTC().Unix(),
        LastEaten: a.LastEaten.UTC().Unix(),
        Weight:    int32(a.Weight),
        Price:     int32(a.Price),
    }
}
```

### Wrapping a function in different ways

***TODO*** *This bit isn't fully implemented yet, but it will be soon!*

Let's take the source code for an example function, Divide:

```go
// Divide returns a divided by b. If b is zero, returns an error.
func Divide(a, b float64) (float64, error) { /* ... */ }
```

Morph can quickly let us generate functions that wrap Divide but take a 
bunch of different forms (implementations elided for the sake of readability).

```go
// Halver divides x by two.
func Halver(x float64) float64 { /* ... */ }

// Divider constructs a function that partially applies [Divide] with a
// constant divisor, returning a new function that divides by the given
// divisor. It panics if the divisor is zero.
func Divider(divisor float64) func (a float64) float64 { /* ... */ }

// DividePromise returns a callback function (a promise) to call Divide(a, b) 
// when called.
func DividePromise(x, b float64) func () (float64, error) { /* ... */ }

// DivideResult returns the result of Divide(a,b) collected into a Result 
// sum type.
func DivideResult(a, b float64) Result.R[float64] { /* ... */ }

// Decimal is a method on a Fraction that converts a value such as
// Fraction{Numerator: 1, Denominator: 3}` into a result such as 0.3333.
func (f Fraction) Decimal() (float64, error) { /* ... */ }
```

Tutorials
---------

* [Mapping between Go structs.]
* [Wrapping Go functions in different ways.]
* [Using morph with `go generate`.]
* [Morph recipes for any occasion.]


Security Model
--------------

WARNING: It is assumed that all inputs are trusted. DO NOT accept arbitrary
input from untrusted sources under any circumstances, as this will parse
and generate arbitrary code.


[fields]: https://pkg.go.dev/github.com/tawesoft/morph/fields
[Mapping between Go structs.]: doc/mapping-between-go-structs.md
[Wrapping Go functions in different ways.]: doc/wrapping-go-functions.md
[Using morph with `go generate`.]: doc/morph-go-generate.md
[Morph recipes for any occasion.]: doc/morph-recipies.md
