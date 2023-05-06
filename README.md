![Morph](morph.png)

[![Go Reference](https://pkg.go.dev/badge/github.com/tawesoft/morph#section-documentation.svg)](https://pkg.go.dev/github.com/tawesoft/morph#section-documentation)

Morph
=====

Morph is a small library that generates Go code to map between structs...

- without runtime reflection.

- without stuffing a new domain-specific language into struct field tags.

- with a simple, fully programmable mapping described in native Go code.

- where you can map to existing types, or use Morph to automatically generate 
  new types. 

Example
-------

Take an existing struct, `Apple`, that we want to automatically generate a new
`Orange` struct from.

The new `Orange` struct is automatically going to have the same fields as the
`Apple` struct, except it will store time as integer epoch seconds instead.

```go
type Apple[X any] struct {
    Colour    color.RGBA
    Picked    time.Time
    LastEaten time.Time
    Contains  []X
}
```

First we parse it:

```go
apple, err := morph.ParseStruct("example.go", source, "Apple")
```

Then we describe how to create a new `Orange` struct definition from it by 
defining a function that is called on every field in `Apple`:

```go
orange, err := apple.StructFunc("Orange[X any]",
    func(name, Type, tag string, emit func(name, Type, tag string)) {
        
    if Type == "time.Time" {
        emit(name, "int64", tag) // epoch seconds
    } else {
        emit(name, Type, tag) // unchanged
    }
})
```

This generates the Go source code:

```go
type Orange[X any] struct {
    Colour    color.RGBA
    Picked    int64
    LastEaten int64
    Contains  []X
}
```

We then describe how to generate a function that can map any `Apple` value 
into an `Orange` value by defining a function that is again called on every
field in `Apple`:

```go
signature := "appleToOrange[I Insect](a Apple[I]) Orange[I]"
appleToOrange, err := apple.FunctionFunc(signature,
    func(name, Type, tag string, emit func(name, value string)) {
        
    if Type == "time.Time" {
        emit(name, "$.UTC().Unix()")
    } else {
        emit(name, "$")
    }
})
```

Here, `$` is used as a short-hand to refer to the matching input field.

We could have instead written, with `a` corresponding to the input field
defined in the signature:

```go
    if Type == "time.Time" {
        emit(name, "a." + name + ".UTC().Unix()")
    } ...
```

Note that we've assumed here that the slice field is read-only, so it 
doesn't matter that we've copied a reference to it rather than cloning the
slice. If that's not the case, then
[clone](https://pkg.go.dev/golang.org/x/exp/slices#Clone) the slice e.g.:

```go
    ... } else if strings.HasPrefix(Type, "[]") {
        emit(name, "slices.Clone($)")
    } ...
```

Anyway, `appleToOrange` generates the Go source code:

```go
func appleToOrange[I Insect](a Apple[I]) Orange[I] {
    return Orange[I]{
        Colour:    a.Colour,
        Picked:    a.Picked.UTC().Unix(),
        LastEaten: a.LastEaten.UTC().Unix(),
        Contains:  a.Contains,
    }
}
```

## Security Model

WARNING: It is assumed that all inputs are trusted. DO NOT accept arbitrary
input from untrusted sources under any circumstances, as this will parse
and generate arbitrary code.
