[![Morph](../morph.png)](https://github.com/tawesoft/morph)

# Morph recipes for any occasion

## Automatically generate custom XML or JSON struct tags for Go structs

Teach morph about your original struct. Either pass a 
file name and a struct name to [morph.ParseStruct]:

```go
s, err := morph.ParseStruct("filename.go", nil, "MyStruct")
```

... or parse a struct directly from a string containing Go source code:

```go
source := `package temp
type MyStruct struct {
    Foo, Bar string
}`
s, err := morph.ParseStruct("temp.go", source, "")
```

Based on this struct, morph can generate source code for a new `type struct 
MyStructJson` and `type struct MyStructXml`, as well as mapping functions to
convert to and from your original `type struct MyStruct`.

For JSON, use the Json helpers from [morph/fields]. You might like 
to examine their implementations if they don't do everything you want.

```go
// combine multiple mappers into one
jsonMapper = fields.Compose(
    fields.JsonOmitEmpty, // automatically add "omitempty" json struct tag
    fields.JsonLowercase, // automatically lowercase the json key name
)

// generate struct with the same name but ending in "Json"
src, err := s.Struct("$Json", jsonMapper) ...

// and generate functions to convert between the two
src, err = s.Function("func (from $) ToJson() $Json", jsonMapper) ...
src, err = s.Function("func $FromJson($Json) $", fields.StripTags) ...
``` 

For XML, use the Xml helpers from [morph/fields].

```go
// combine multiple mappers into one
xmlMapper = fields.Compose(
    fields.XmlOmitEmpty, // automatically add "omitempty" xml struct tag
    fields.XmlLowercase, // automatically lowercase the xml key name
)

// generate struct with the same name but ending in "Xml"
src, err := s.Struct("$Xml", xmlMapper)

// and generate functions to convert between the two
src, err = s.Function("func (from $) ToXml() $Xml", xmlMapper)
src, err = s.Function("func $FromXml($Xml) $", fields.StripTags)
```

Of course, if you really wanted to, you could combine the two and generate
XML and JSON tags on a single struct.

If you have fields that don't easily serialize to a JSON or XML value,
you could write your own morph.Mapper to convert their types, perhaps to a type 
with a [custom MarshalXml implementation]. See the tutorial, [Mapping 
Between Go Structs with Morph], for help with this.

***TODO*** *Implement the ...OmitEmpty and ...Lowercase mappers*

***TODO*** *What if morph could automatically generate code for reflection-free 
XML and JSON parsers?*

## Automatically generate SQL field types for Go structs

***TODO***

## Automatically initialise a Go struct with default values

***TODO***

## Deep copy and deep equals without runtime reflection

***TODO***

## Serialize variable-length Go structs to disk

***TODO***

[morph.ParseStruct]: https://pkg.go.dev/github.com/tawesoft/morph#ParseStruct
[morph/fields]: https://pkg.go.dev/github.com/tawesoft/morph/fields
[Mapping Between Go Structs with Morph]: mapping-between-go-structs.md
[custom MarshalXml implementation]: https://pkg.go.dev/encoding/xml#example-package-CustomMarshalXML
