[![Morph](../morph.png)](https://github.com/tawesoft/morph)

# Apples To Oranges: mapping between Go structs with Morph

> **Example code:** [ApplesToOranges]

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


## Parsing a struct from source code

First, we need to tell morph about the struct with [morph.ParseStruct]. This
can read from a `string`, `[]byte`, `io.Reader`, or a file by filename.

```go
apple := must(morph.ParseStruct("apple.go", nil, "Apple"))
```

Morph doesn't need to fully resolve identifiers like `weight` and `price`, so
we don't need to tell morph anything about them.

> **Tip:** use the Go compiler to ensure that morph generates correct code.


## Mapping to a new struct

There are two ways morph can change our parsed Apple struct into our
desired Orange struct.

* Using a [morph.StructMapper] and the [morph.Struct.Map] method.
* Using a [morph.FieldMapper] and the [morph.Struct.MapFields] method.

### Using a StructMapper to rename a struct

A StructMapper is a function that takes a copy of a struct as input, and 
outputs a new struct.

The [structmappers package] contains useful things for working with
StructMappers, like the [structmappers.Rename] function that returns a 
new StructMapper that sets a struct's name to the provided argument.

```go
orange := apple.Map(structmappers.Rename("Orange"))
```


### Using a FieldMapper to change a struct's fields

A FieldMapper is a function that is called on every field on an input struct, 
and generates output fields on an output struct.

For our Apple, that means a FieldMapper will be called once each for the fields
`Picked`, `LastEaten`, `Weight`, and `Price`.

The [fieldmappers package] contains useful things for working with 
FieldMappers, like the FieldMapper implementation [fieldmappers.TimeToInt64].

This, as its name suggests, maps `time.Time` fields to `int64` values. Other
fields are left unchanged, and output normally.

```go
orange = orange.MapFields(fieldmappers.TimeToInt64)
```

FieldMappers are often general purpose enough that they can be reused and 
combined in different situations.

### Creating a custom FieldMapper

We still want to map our `Weight` and `Price` fields, so it's time to make
our own FieldMapper.

We haven't provided type information for these, but let's assume there are 
`weight` and `price` packages that export the following features:

```go
package weight

type Weight struct {}
func (w Weight) Grams() int64
func FromGrams(grams int64) Weight
```

```go
package price

type Price struct {}
func (p Price) Pence() int64
func FromPence(pennies int64) Price
```

These packages are just for us -- morph doesn't need to know about them.

A FieldMapper has the following signature:

```go
type morph.FieldMapper func(input morph.Field, emit func(output morph.Field))
```

The `input` argument lets us discriminate what fields we want our mapper to 
apply to. The `emit` function controls what fields are generated on the output.
If we don't emit the field, that field is removed. If we emit more than once,
we can emit multiple fields per input field.

The simplest FieldMapper is one that does nothing:

```go
func Passthrough(input morph.Field, emit func(output morph.Field)) {
    emit(input)
}
```

Let's implement our FieldMapper for weight and price separately, and let's
discriminate on field type, rather than name, so that the mappers are more
general-purpose and reusable. We'll see another reason why we have separate 
price and weight implementations, later.

```go
func WeightToInt64(input morph.Field, emit func(output morph.Field)) {
    if input.Type == "weight.Weight" {
        output: = input // copy
        output.Type = "int64" // rewrite the type
        emit(output)
    } else {
        emit(input)
    }
}
```

> **Practice:** implement a FieldMapper for the Price field in the example 
> code.

We'll add more to this later, but it's enough for generating a struct
definition for our Orange.

Now, by composing multiple StructMappers and FieldMappers we get our Orange:

```go
orange = apple.Map(
    structmappers.Rename("orange"),
).MapFields(
    fieldmappers.TimeToInt64,
    WeightToInt64,
    // PriceToInt64 - left as exercise for reader
)

fmt.Println(apple)
fmt.Println(ornage)

// output:
// type Apple struct {
//     Picked    time.Time
//     LastEaten time.Time
//     Weight    weight.Weight
//     Price     price.Price
// }
// type Orange struct {
//     Picked    int64 // time in seconds since Unix epoch
//     LastEaten int64 // time in seconds since Unix epoch
//     Weight    int64
//     Price     price.Price
// }
```

> **Tip:** that's a Go source code fragment - we can write that string to a 
> file and get an automatically generated struct.
> 
> If we re-run that process every time Apple changes, then Orange will change 
> appropriately too.

If you'll notice, `fieldmappers.TimeToInt64` set a nice comment for us too.

## Generating struct conversion functions

> **Tip:** this section is still useful even if you have two existing struct
> type definitions and aren't automatically generating one from the other.

We've now got a definition for our new Orange struct type:

```
type Orange struct {
    Picked    int64 // time in seconds since Unix epoch
    LastEaten int64 // time in seconds since Unix epoch
    Weight    int64
    Price     int64
}
```

That's struct types. What about struct values?

Let's say we have an apple value:

```go
apple := Apple{
    Picked: time.Parse(time.DateTime, "2023-05-29 09:00:00"")
    // LastEaten: zero value
    Weight: weight.FromString("50 kg")
    Price: price.FromString("£1.23")
}
```

We want to be able to convert between them, like so:

```go
orange := apple.ToOrange()
appleAgain := orange.ToApple()
```

This is achieved through the [morph.Struct.Converter] method, which generates
source code for a function that converts a struct based on its mapping.

```go
appleToOrange, err := orange.Converter("(apple Apple) ToOrange() Orange")
```

The generated function matches the supplied function signature, which is
quite flexible. The function signature must have at least one argument or 
receiver value of the source type (here, Apple), and exactly one return 
value of the result type (here, Orange). Pointer values are fine, too.

We can actually use a shortcut here, and use `$from`, `$From` and `$To`
tokens to fill in some of the name for us, like so:

```go
const converterFormat = "($from $From) To$To() $To"
appleToOrange, err := orange.Converter(converterFormat)
```

This generates an output like so, with a helpful comment automatically
generated:

```go
// ToOrange converts [Apple] to [Orange].
func (apple Apple) ToOrange() Orange {
	return Orange{
		Picked:    apple.Picked.UTC().Unix(),
		LastEaten: apple.LastEaten.UTC().Unix(),
		// Weight is the zero value.
		// Price is the zero value.
	}
}
```

Note that the Weight and Price fields haven't been updated. We have to
go back to our custom FieldMapper implementation.

```
func WeightToInt64(input morph.Field, emit func(output morph.Field)) {
    if input.Type == "weight.Weight" {
        output: = input // copy
        output.Type = "int64" // rewrite the type
        output.Value = "$.$.Grams()" // rewrite the value
        emit(output)
    } else {
        emit(input)
    }
}
```

Here we see `$` tokens again. In a FieldMapper, they have the following 
meanings when they appear in a `morph.Field.Value`:

| Token | Description                              | Example                                       |
|-------|------------------------------------------|-----------------------------------------------|
| `$.`  | Input struct name                        | `$.LastEaten` replaced with `apple.LastEaten` |
| `$.$` | Input struct name and current field name | `$.$` replaced with `apple.Weight`            |

This is enough to generate the forward mapping, `Apple` to `Orange`. Again,
the Price mapping has been left as an exercise.

```go
func (apple Apple) ToOrange() Orange {
	return Orange{
		Picked:    apple.Picked.UTC().Unix(),
		LastEaten: apple.LastEaten.UTC().Unix(),
		Weight:    apple.Weight.Grams(),
		Price:     apple.Price.Pence()
	}
}
```

But what about the reverse, `ToApple`?

### Generating a reverse struct conversion function

We want to generate a function, `func (orange Orange) ToApple() Apple`.

We need to perform the reverse steps to map an Orange morph.Struct back into an 
Apple morph.Struct, (even if we never save the generated type definition 
anywhere).

We could do this by applying FieldMappers that undo the changes we've made.
That would be a bit awkward, however, because all the fields on our Orange
are of type `int64` and it's difficult to discriminate between them.

Fortunately, some StructMappers and FieldMappers are reversible, so we can
have morph do this for us!

```go
appleFromOrange := orange.Reverse()
orangeToApple, err := appleFromOrange.Converter(converterFormat)
```

> **Tip:** the const `converterFormat` template string from earlier came in 
> handy! We didn't have to specify the new function signature.

But before we do, let's go back to our custom FieldMappers one last time,
and make them reversible.

A reverse function is simple to write. It is just a FieldMapper, but as it is 
only called on a field that has explicitly set it as its reverse function, 
it does not need to discriminate on name or type. The original mapper then 
registers the Reverse function on the emitted field.

This makes our full custom mapper look as follows:

```
func WeightToInt64(input morph.Field, emit func(output morph.Field)) {
    int64ToWeight := func(input morph.Field, emit func(output morph.Field)) {
        output = input
        output.Type = "weight.Weight"
        output.Value = "weight.FromGrams($.$)"
        emit(output)
    }

    if input.Type == "weight.Weight" {
        output := input // copy
        output.Type = "int64" // rewrite the type
        output.Value = "$.$.Grams()" // rewrite the value
        output.Reverse = fieldmappers.Compose(int64ToWeight, output.Reverse)
        output.Comment
        emit(output)
    } else {
        emit(input)
    }
}
```

***// TODO: should compose with existing reverse automatically***

So finally:

```go
fmt.Println(orangeToApple)
```

And our generated source code:

```go
// ToApple converts [Orange] to [Apple].
func (orange Orange) ToApple() Apple {
	return Apple{
		Picked:    time.Unix(orange.Picked, 0).UTC(),
		LastEaten: time.Unix(orange.LastEaten, 0).UTC(),
		Weight:    weight.FromGrams(orange.Weight),
		Price:     price.FromPence(orange.Price),
	}
}
```


[ApplesToOranges]: https://pkg.go.dev/github.com/tawesoft/morph#example-applesToOranges
[morph.ParseStruct]: https://pkg.go.dev/github.com/tawesoft/morph#ParseStruct
[morph.StructMapper]: https://pkg.go.dev/github.com/tawesoft/morph#StructMapper
[morph.FieldMapper]: https://pkg.go.dev/github.com/tawesoft/morph#FieldMapper
[morph.Struct.Map]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.Map
[morph.Struct.MapFields]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.MapFields
[morph.Struct.Converter]: https://pkg.go.dev/github.com/tawesoft/morph#Struct.Converter
[structmappers package]: https://pkg.go.dev/github.com/tawesoft/morph/structmappers
[structmappers.Rename]: https://pkg.go.dev/github.com/tawesoft/morph/structmappers#Rename
[fieldmappers package]: https://pkg.go.dev/github.com/tawesoft/morph/fieldmappers
[fieldmappers.TimeToInt64]: https://pkg.go.dev/github.com/tawesoft/morph/fieldmappers#TimeToInt64
