[![Morph](../morph.png)](https://github.com/tawesoft/morph)

# Deep copy, deep equals, and ordering without runtime reflection using Morph

In the [previous tutorial], we learnt about parsing a struct type definition
from Go source code and applying StructMapper and FieldMapper transformations
to automatically derive new structs.

[previous tutorial]: ./mapping-go-structs-with-morph.md

We learned how Morph can generate new struct type definitions, and can
generate new functions that convert struct values between struct types
using [morph.Struct.Converter].

[morph.Struct.Converter]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.Converter

In a similar way, we can use FieldMappers to set the comparison expression on
each field, and use Morph to generate a "deep equals" and "deep copy"
functions, like [reflect.DeepEqual], but without using runtime reflection,
and with greater control over the implementation.

[reflect.DeepEqual]: https://pkg.go.dev/reflect#DeepEqual

> **Why do we want to avoid runtime reflection?**
> Runtime reflection is slow, difficult to write, and can cause runtime errors.
> 
> Additionally, the reflect package is a large import that
> isn't available everywhere (e.g. not completely implemented in TinyGo), so
> runtime reflection makes code less portable.

## Generating an initial equality function

Let's assume we have the following struct type definition, defined in Go source 
code or as a string literal somewhere:

```go
type Person struct {
    ID string
    Name string
}
```

The first step is to parse the struct type definition with morph:

```go
person := must(morph.ParseStruct("test.go", source, "Person"))
```

We can generate an initial comparison function using [morph.Struct.Comparer].

[morph.Struct.Comparer]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.Comparer

But first, we need to define a function signature for the function we want
to generate. We don't need to specify the bool return argument.

```go
sig := `PersonEquals(first Person, second Person)`
```

Or, more generally:

```go
sig := `$Equals(first $, second $)`
```

Then:

```go
fmt.Println(must(person.Comparer(sig)))
```

This outputs something like:

```go
// PersonEquals returns true if two [Person] values are equal.
func PersonEquals(first Person, second Person) {
    // (note: actual code generated might look different
    // to this, but will do exactly the same thing.)
    return (first.ID == second.ID) &&
        (first.Name == second.Name)
}
```

Now let's customise this further.

> **Tip:** Morph might generate code that looks different to this, but that
> gives the same result. The generated code is more verbose so that mistakes 
> are easier to spot and diagnose. The Go compiler is able to optimise this
> into an efficient representation, regardless.


## Generating a custom equality function

Let's imagine that we want two Person values to compare equal if their ID 
values compare equal, case insensitively, and ignoring their name.

To do so, we have to set the Comparer expression on each field. This is an
expression that must return true or false.

Recall that the parsed `person` is a `morph.Struct` containing a slice of 
fields:

```go
person.Fields == []Field{
    {Name: "ID",   Type: "string", Comparer: "", ...},
    {Name: "Name", Type: "string", Comparer: "", ...},
}
```

For the sake of demonstration, we're going to modify the Comparer on these 
fields directly. Normally, we'd write a more general-purpose FieldMapper, or 
use an existing one from the [fieldmappers package] or [fieldops package].

[fieldmappers package]: https://pkg.go.dev/github.com/tawesoft/morph/fieldmappers
[fieldops package]: https://pkg.go.dev/github.com/tawesoft/morph/fieldmappers/fieldops

When a comparer is set to an empty string, morph treats it as a value that 
can be compared for equality in the normal way, with the Go equality
operator `==`.

To ignore a comparison, the comparer expression can simply be set to a string
containing the Go keyword `true` (i.e. the field always compares equal).

```go
person.Fields[1].Comparer = "true" // ignore Name
```

Generally, however, we need to write a new expression describing how to 
compare a field on two struct values.

Here's how to compare the ID field, ignoring case:

```go
person.Fields[0].Comparer = "strings.EqualFold($a.$, $b.$)"
```

Here we see `$` tokens again. They have the following meanings when they 
appear in a `morph.Field.Comparer`:

| Token  | Description                              | Example                                         |
|--------|------------------------------------------|-------------------------------------------------|
| `$x.`  | Input struct name                        | `$a.LastEaten` replaced with `first.LastEaten`  |
| `$x.$` | Input struct name and current field name | `$b.$` replaced with `second.Name`              |

Replace `x` with `a` for the first argument in the function signature,
or with `b` for the second argument in the function signature. In this way,
a comparer expression can be written for any two inputs, regardless of what
name they are given in the function signature.

If we generate a comparer function again, we'll have something that looks
more like this:

```go
// PersonEquals returns true if two [Person] values are equal.
func PersonEquals(first Person, second Person) {
    return strings.EqualsFold(first.ID, second.ID)
}
```


## Generating a custom "deep-equals" function

Let's consider a few container types.

### Tree

```go
type Tree[X comparable] struct {
    Value X
    Children Tree[X]
}
```

We'll consider the tree equal if the values are equal and each child
is equal (recursively).

This is trivial using [slices.EqualFunc]:

[slices.EqualFunc]: https://pkg.go.dev/golang.org/x/exp/slices#EqualFunc

```go
tree := must(morph.ParseStruct("test.go", source, "Tree"))
sig := `TreesEqual[X comparable](x $[X], y $[X])`
tree.Fields[1].Comparer = "slices.EqualFunc($a.$, $b.$, TreesEqual)"
```

### Linked list with cycles

```go
type List[X comparable] struct {
    Value X
    Next *List[X]
}
```

We'll consider a list equal to another list if the values are equal and
each subsequent element, pointed to by Next, are either both nil or both
non-nil elements that compare equal (recursively).

To make this more difficult, let's allow our list to contain an infinite loop.

To implement this while handling cycles, subsequent times that we compare two 
pointer values that have been compared before, we have to assume that we are
comparing infinite sequences. To match the behaviour of [reflect.DeepEqual],
we'll assume infinite sequences that compare equal continue to compare equal
as long as all finite parts compare equal.

[reflect.DeepEqual]: https://pkg.go.dev/reflect#DeepEqual

This requires some machinery to go along with the generated code:

```go
type visit struct {
    a, b unsafe.Pointer
}

type visitor map[visit]struct{}

func (v visitor) seen(a, b unsafe.Pointer) bool {
    if uintptr(a) > uintptr(b) {
        a, b = b, a
    }
    _, exists := v[visit{a, b}]
    return exists
}

func (v visitor) mark(a, b unsafe.Pointer) {
    if uintptr(a) > uintptr(b) {
        a, b = b, a
    }
    v[visit{a, b}] = struct{}{}
}

func compare[T any](a *T, b *T, v visitor, cmp func(*T, *T, visitor) bool) bool {
    if a == nil && b == nil { return true }
    if (a == nil || b == nil) && (a != nil || b != nil) { return false }
    if v.seen(unsafe.Pointer(a), unsafe.Pointer(b)) { return true }
    v.mark(unsafe.Pointer(a), unsafe.Pointer(b))
    return cmp(a, b, v)
}

func ListsEqual[X comparable](as *List[X], bs *List[X]) bool {
    // ListsEqual is a wrapper for listsEqual,
    // which is generated using morph.
    return listsEqual(as, bs, visitor(make(map[visit]struct{})))
}
```

Then:

```go
list := must(morph.ParseStruct("test.go", source, "List"))
sig := `listsEqual[X comparable](x *$[X], y *$[X], v visitor)`
list.Fields[1].Comparer = "compare($a.$, $b.$, v, listsEqual[X])"
```

This might seem like a lot of setup and writing the code manually might
be quicker. And that's true for a single, simple case. But the point is to
make this reusable and composable so that it can quickly be applied to any
struct type with any number of fields.


## Generating a copy function.

Generating a copy function is much the same as creating an equality function: 
parse a struct type definition, set the Copier expression on each relevant
field, and call [morph.Struct.Copier] to generate the copy function.

[morph.Struct.Copier]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.Copier

An empty Copier expression means to copy using the native Go assignment 
operator, "=".

Otherwise, a copier expression refers to source and destination values
with `$` token notation. These have the following meanings when they 
appear in a `morph.Field.Copier`:

| Token       | Description                               | Example                                         |
|-------------|-------------------------------------------|-------------------------------------------------|
| `$target.`  | Target struct name                        | `$src.LastEaten` replaced with `from.LastEaten` |
| `$target.$` | Target struct name and current field name | `$dest.$` replaced with `to.Name`               |

Replace `target` with `src` for the input argument in the function signature,
or with `dest` for the named or unnamed return value. In this way, a copier 
expression can be written for any input and output, regardless of what name 
they are given (or not given) in the function signature.

## Generating a "deep copy" function:

For example, let's revisit `Tree`:

```go
type Tree[X comparable] struct {
    Value X
    Children Tree[X]
}
```

We can implement a "deep copy" quite simply:

```go
tree := must(morph.ParseStruct("test.go", source, "Tree"))
sig := `(from $[X Comparable]) Copy() $`
tree.Fields[1].Copier = "$dest.$ = append(Tree[X]{}, Map(Tree.Copy, $src.$))"
fmt.Println(must(tree.Copier(sig)))
```

Again, for the sake of example, this is achieved by explicitly setting a 
field's Copier directly. Normally, we'd write a more general-purpose FieldMapper,
or use an existing one from the [fieldmappers package] or [fieldops package].

This supposes a generic higher-order function called `Map`. You can use 
[tawesoft/golib/v2/fun/slices.Map] or perform a cursory 
Google search for "golang generic slice map".

[tawesoft/golib/v2/fun/slices.Map]: https://pkg.go.dev/github.com/tawesoft/golib/v2/fun/slices#Map

A "deep copy" that also allows for cycles can be performed using the visitor 
technique described in a previous section.


## Generating an ordering function.

An ordering function can be used to sort a collection of items. Like Go, we
describe an ordering by implementing a "less" function i.e. a function 
that returns true when `a < b`.

Generating an ordering function is much the same as creating an equality 
function: parse a struct type definition, set the Orderer expression on each
relevant field, and call [morph.Struct.Orderer] to generate the ordering 
function.

[morph.Struct.Orderer]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.Orderer

An empty orderer expression means to compare using the native Go less-than
operator, `<`.

Otherwise, an orderer expression refers to first and second values
with `$` token notation. These have the following meanings when they 
appear in a `morph.Field.Orderer`:

| Token  | Description                              | Example                                         |
|--------|------------------------------------------|-------------------------------------------------|
| `$x.`  | Input struct name                        | `$a.LastEaten` replaced with `first.LastEaten`  |
| `$x.$` | Input struct name and current field name | `$b.$` replaced with `second.Name`              |

Replace `x` with `a` for the first argument in the function signature,
or with `b` for the second argument in the function signature. In this way,
an orderer expression can be written for any two inputs, regardless of what
name they are given in the function signature.

## Generating a "deep ordering" function.

For example, let's revisit `Tree`:

```go
type Tree[X constraints.Ordered] struct {
    Value X
    Children Tree[X]
}
```

Note that we've had to narrow the type constraint.

We can implement a "deep ordering" quite simply:

```go
tree := must(morph.ParseStruct("test.go", source, "Tree"))
sig := `$LessThan(first $, second $)`
tree.Fields[1].Orderer = "LessThanFunc($a.$, $b.$, TreeLessThan)"
fmt.Println(must(tree.Orderer(sig)))
```

Again, for the sake of example, this is achieved by explicitly setting a 
field's Orderer directly. Normally, we'd write a more general-purpose 
FieldMapper,
or use an existing one from the [fieldmappers package] or [fieldops package].

This supposes a generic higher-order function called `LessThanFunc` that
looks something like this:

```go
func [X constraints.Ordered] LessThanFunc(as, bs []X, lt func(X, X)) bool {
    // let nils compare equal, and less than non-nils 
    if as == nil && bs == nil { return false }
    if as == nil { return true }
    if bs == nil { return false }
    
    // return true or false on the first non-equal value
    for i := 0; i < len(as); i++ {
        if lt(a, b) { return true }
        if lt(b, a) { return false }
    }
    
    // equal length and equal values
    if len(a) == len(b) { return false }
    
    // shared prefix, but shorter array is less than longer array
    if len(a) < len(b) { return true }
    return false
}
```


A "deep ordering" that also allows for cycles can be performed using the 
visitor technique described in a previous section.
