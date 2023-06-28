![Morph](morph.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/tawesoft/morph#section-documentation.svg)](https://pkg.go.dev/github.com/tawesoft/morph#section-documentation)

Morph
=====

Morph is a Go code generator that makes it easy to map between
Go structs, automatically derive new Go structs, compare structs, and transform 
functions in different ways.

**Use cases for Morph**

* "Compile-time reflection" for Go.
* Serialisation and interoperability with other systems e.g. JSON, XML, SQL, 
  and binary.
* Separation between data modeling layers.
* Functional programming.
* Graphics programming.
* Statistics e.g. histograms and buckets.

**Why use Morph?**

* Reduce time spent maintaining boilerplate.
* Fewer opportunities for bugs caused by typos in copy & paste code.

**Morph features**

 - Fully programmable (in native Go).
 - Full support for generics.
 - No runtime reflection.
 - No struct field tags to learn or cause clutter.
 - Elegant and composable building blocks.
 - Big library of done-for-you mappings and helpers:
   * [fieldmappers] for struct fields.
   * [structmappers] for structs.
   * [funcwrappers] for functions.
- Zero external dependencies!

[fieldmappers]: https://pkg.go.dev/github.com/tawesoft/morph/fieldmappers
[structmappers]: https://pkg.go.dev/github.com/tawesoft/morph/structmappers
[funcwrappers]: https://pkg.go.dev/github.com/tawesoft/morph/funcwrappers

**Status**

* Morph core is feature complete ✓
* **broken** refactor for more general purpose custom extensibility ⬅
* Documentation is in progress ⬅
* Needs more features in subpackages
* Morph core needs tidying and better error handling


Quick Examples
--------------

These are all covered in more detail in the tutorials which follow this
section.

### Mapping between structs

Take the source code for an example struct, Apple, which uses custom-made
weight and price packages.

```go
type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
    Price     price.Price
}
```

Let's say we want a new struct, similar to this, but where every field is an
integer, to make it easier to serialise and interoperate with other systems.

We're going to call it an Orange.

Morph can quickly let us generate:

```go
// Orange is like an [Apple], but represented with ints.
type Orange struct {
    Picked    int64 // time in seconds since Unix epoch
    LastEaten int64 // time in seconds since Unix epoch
    Weight    int64 // weight in grams
    Price     int64 // price in pence
}

// AppleToOrange converts an [Apple] to an [Orange].
func AppleToOrange(from Apple) Orange {
    return Orange{
        Picked:    from.Picked.UTC().Unix(),
        LastEaten: from.LastEaten.UTC().Unix(),
        Weight:    from.Weight.Grams(),
        Price:     from.Price.Pence(),
    }
}

// OrangeToApple converts an [Orange] to an [Apple].
func OrangeToApple(from Orange) Apple {
    return Apple{
        Picked:    time.Unix(from.Picked, 0).UTC(),
        LastEaten: time.Unix(from.LastEaten, 0).UTC(),
        Weight:    weight.FromGrams(from.Weight),
        Price:     price.FromPence(from.Price),
    }
}
```

In fact, we can quickly generate any additional assignment, comparison, or 
inspection functions we desire.

From sorting:

```go
// Fresher returns true if Orange "o" was picked more recently than Orange
// "target". This can be used as an ordering to sort oranges by freshness or 
// in order of those most in danger of going bad.
func (o Orange) FresherThan(target Orange) bool { /* ... */ }
```

To serialization:

```go
// MarshalJSON converts an Orange to a JSON representation. It omits empty 
// fields, converts all field names to lowercase, and means we don't 
// have to pollute our struct type definition with tags.
func (o Orange) MarshalJSON() ([]byte, error) { /* ... */ }
```

### Deep copy and deep equals without reflection

Let's take the code for a recursive tree type:

```go
type Tree struct {
    Value string
    Time time.Time
    Children []Tree
}
```

Morph can generate custom copy, equals, ordering, deep copy, deep equals, 
and deep ordering functions -- all without using runtime reflection:

```go
// TreesEqual returns true if two [Tree] values are equal, with the Value
// string compared in a case-insensitive manner.
func TreesEqual(a Tree, b Tree) bool {
    return strings.EqualFold(a.Value, b.Value) &&
        a.Time.Equals(b.Time) &&
        slices.EqualFunc(a.Children, b.Children, TreesEqual)
}

// Copy returns a copy of the [Tree] t.
func (t *Tree) Copy() Tree {
    // for a supplied slice application operator, `Map`.
    return Tree{
        Value: t.Value,
        Time: t.Time,
        Children: append([]Tree(nil), Map(Tree.Copy, t.Children))
    }
}

// TreesLessThan returns true if the first [Tree] is less than the second.
func TreesLessThan(a Tree, b Tree) {
    // for a supplied slice comparison operator, `LessThanFunc`.
    return (a.Value < b.Value) ||
        b.Time.After(a) ||
        LessThanFunc(a.Children, b.Children, TreesLessThan)
}
```

This can also be extended to support collections that contain cycles.


### Wrapping a function in different ways

Go supports higher-order functions and generic types. This means we can
take an ordinary function, like `func (a A, b B) (A, error)` for any type A 
and B, and write a generic higher-order function that takes this function as 
an input and returns a different function as a result.

However, Go is limited in that it can't do this for arbitrary functions of
any number of input arguments and any number of return values. A developer
has to create a separate function for every combination of input and output 
counts.

For example, there's no way to write a higher order function in native Go that
automatically applies [context.TODO] to every possible function that takes a
[context.Context] for its first argument.

We can use Morph to create and compose various automatic transformations of
functions, thereby increasing our expressive power for functional programming
in Go.

[context.Context]: https://pkg.go.dev/context#Context
[context.TODO]: https://pkg.go.dev/context#TODO

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
// when called. The division is not performed until the returned promise is
// called.
func DividePromise(x, b float64) func () (float64, error) { /* ... */ }

// DivideResult returns the result of Divide(a,b) collected into a Result 
// sum type.
func DivideResult(a, b float64) result.R[float64] { /* ... */ }
```

Tutorials
---------

### Structs

*This section is to be read in order, as it incrementally introduces more
advanced Morph features for manipulating structs.*

1. [Apples To Oranges: mapping between Go structs with Morph.]
2. [Deep copy and deep equals without runtime reflection.]
3. [Automatically generate XML or JSON struct tags.]
4. [Mapping to column-orientated data types.]

### Functions

*This section is to be read in order, as it incrementally introduces more
advanced Morph features for manipulating functions.*

* Wrapping and transforming Go functions with Morph.
* Creating custom function wrappers.
* Wrapping and transforming higher-order functions.

### General

* Repeatable Morph code generation with `go generate`.

[Apples To Oranges: mapping between Go structs with Morph.]: doc/mapping-go-structs-with-morph.md
[Deep copy and deep equals without runtime reflection.]: doc/deep-copy-equals-without-reflection.md
[Automatically generate XML or JSON struct tags.]: doc/auto-xml-json-struct-tags.md
[Mapping to column-orientated data types.]: ./doc/column-orientated-structs.md

Security Model
--------------

WARNING: It is assumed that all inputs are trusted. DO NOT accept arbitrary
input from untrusted sources under any circumstances, as this will parse
and generate arbitrary code.
