package morph_test

import (
    "fmt"

    "github.com/tawesoft/morph"
)

func must[T any](t T, err error) T {
    if err != nil { panic(err) }
    return t
}

func ExampleMorph() {
    // Here's some Go source code that defines two struct types that describe
    // the same conceptual thing, but in different ways.
    //
    // In this example, "Thing" represents a value in the form that's most
    // useful for our program at runtime, and "ThingDisk" represents a value in
    // the form that's most useful for serialising to and from disk.
    //
    // Rather than hard-code the conversion between them, we want to use Morph
    // to generate a mapping from any value of type "Thing" to any value
    // of type "ThingDisk".
    src := `
package Example

type Thing struct {
    Name string
    Created, Modified, Accessed time.Time
}

type ThingDisk struct {
    NameSize int32
    Name string
    Created, Modified, Accessed int64 // seconds since unix epoch
}
`

    // First we parse the source code for the source struct. Here we're parsing
    // from a string literal, but see [morph.Parse] for other ways.
    thing := must(morph.ParseStruct("example.go", src, "Thing"))

    // Then we provide a morpher function that is called for each input field
    // described in the source code, and emits any number of output fields.
    //
    // See [morph.Morpher] for use of "$" as shorthand.
    morpher := func(name, Type string, emit func(name, Type, value string)) {
        if Type == "string" {
            emit("$Size",       "int32",    "len($)")
            emit("$",          "string",    "$")
        } else if Type == "time.Time" {
            emit("$",           "int64",    "$.UTC().Unix()")
        }
    }

    // Here's the function signature we want to generate.
    //
    // If we need to, we're free to add extra input values, make the input or
    // return value a pointer type, or add generic type constraints.
    //
    // We just need to make sure that there's at least one input of the
    // source struct type, and exactly one output of the destination struct
    // type.
    signature := must(morph.ParseSignature("ThingToThingDisk(from *Thing[int]) ThingDisk"))

    // Now we apply the morpher to generate the code for a function that can
    // map from Thing to ThingDisk at runtime.
    fn, def, err := morph.Morph(thing, signature, morpher)
    if err != nil { panic(err) }

    // And when we're done, we convert the AST for the function to source code.
    fmt.Println(fn)

    // We defined ThingDisk ourselves, but we could also use morph to generate
    // it for us:
    fmt.Println(def)

    // output:
    // func ThingToThingDisk(from *Thing[int]) ThingDisk {
    //	return ThingDisk{
    //		NameSize: int32(len(from.Name)),
    //		Name:     string(from.Name),
    //		Created:  int64(from.Created.UTC().Unix()),
    //		Modified: int64(from.Modified.UTC().Unix()),
    //		Accessed: int64(from.Accessed.UTC().Unix()),
    //	}
    // }
    //
    // type ThingDisk struct {
    //	NameSize int32
    //	Name     string
    //	Created  int64
    //	Modified int64
    //	Accessed int64
    // }
}
