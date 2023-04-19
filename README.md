Morph
=====

Morph generates Go code to map values between related struct types

- without runtime reflection.

- without stuffing a new domain-specific language into struct field tags.

- with a simple, fully programmable mapping described in native Go code.

- where you can map to existing types, or use Morph to automatically generate 
  new types. 

**Release status:** feature complete but needs tests, tidying. API subject 
to change. (April 2023)


Simple Example
--------------

We want to map between values of these two struct types:

```go
MyStruct {
    Foo, Bar time.Time
    Fizz Maybe[string]
}

MyStructForSQL {
    Foo, Bar int64 // epoch seconds
    Fizz sql.NullString
}
```

We can create some morpher function that operates on the fields:

```go
func MorphToSQL(name, Type string, emit func(name, Type, value string)) {
    if Type == "time.Time" {
        emit("$", "int64", "$.UTC().Unix()")
    } else if Type == "Maybe[string]" {
        emit("$", "sql.NullString", "MaybeToSqlNullString($)")
    }
}
```

And we use Morph to generate a Go function, which looks like:

```go
func MyStructToMyStructForSQL(from MyStruct) MyStructForSQL {
    return MyStructForSQL{
        Foo: from.Foo.UTC().Unix(),
        Bar: from.Bar.UTC().Unix(),
        Fizz: MaybeToSqlNullString(from.Fizz),
    }
}
```

For the sake of the example, we defined `StructForSQL` ourselves, but Morph 
can also be used to generate that type for us automatically.


Usage
-----

See the example in `morph_example_test.go`.

FAQ
---

### What about generics?

Morph works with generic code, but with a few limitations that you're 
unlikely to run into unless you do something weird.

Morph currently doesn't know how to output generate a generic type when 
automatically generating a struct definition, but could possibly do so in 
future if there's any demand.
