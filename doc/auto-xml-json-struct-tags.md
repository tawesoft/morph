[![Morph](../morph.png)](https://github.com/tawesoft/morph)

## Automatically generate custom XML or JSON struct tags for Go structs using Morph

In a [previous tutorial], we learnt about parsing a struct type definition
from Go source code and applying StructMapper and FieldMapper transformations
to automatically derive new structs.

In [another tutorial], we also learnt how to generate custom equality and copy 
functions.

Let's put this all together and see how it lets us work with XML and JSON.

[previous tutorial]: ./mapping-go-structs-with-morph.md
[another tutorial]: ./deep-copy-equals-without-reflection.md

Let's start with a source file containing these struct type definitions:

```go
type Address struct {
    Since       time.Time // when they started living there
    FlatNumber  string
    HouseNumber string
    Street      string
    City        string
    Country     string
    Postcode    string
}

type Person struct {
    Title      string
    GivenName  string
    FamilyName string
    Address    Address
}
```

We'll be generating new, private, struct types, so we don't have to add XML
or JSON struct tags here.

We'll parse and prepare these with morph:

```go
person := must(morph.ParseStruct("example.go", source, "Person"))
address := must(morph.ParseStruct("example.go", source, "Address"))

person = person.MapFields(
    // make string comparisons case-insensitive
    fieldops.StringsEqualFold,
)

address = address.MapFields(
    // make string comparisons case-insensitive
    fieldops.StringsEqualFold
    
    // set time.Time fields to use the time.Equals method, not '=='.
    fieldops.Time,
)
```

## Creating XML and JSON mappings

Now let's create mappings from our existing Address and Person types to 
specialised Json 
and Xml types: "addressJson", "personJson", "addressXml" and "personXml".

We're using a lowercase letter to start these identifiers to keep them
private to the package. You'll see why later.

Let's assume all fields are optional, with zero values meaning a field can be 
left out when we output JSON and XML (including the whole Address struct, if 
it's the zero value).

Let's also suppose we want fields printed in lowercase e.g. a generated
JSON object for Address should have the fields "since", "flatnumber", "city", 
etc. and not "Since", "FlatNumber", or "City".

### A simple mapping

First let's define an initial mapping from Address to addressJson:

```go
addressJson := address.Map(
    structmappers.Rename("addressJson"),
).MapFields(
    // don't duplicate comments
    fieldmappers.StripComments,
    
    // annotate every field with a lowercase JSON key name
    fieldmappers.JsonRename(strings.ToLower),
    
    // annotate every field with JSON omit empty
    fieldmappers.JsonOmitEmpty,
)
```

And from Address to AddressXml, here refactoring common mappers into a single
mapper formed by composing them:

```go
xmlMappers := fieldmappers.Compose(
    // annotate every field with a lowercase XML element name
    fieldmappers.XmlRename(strings.ToLower),
    
    // annotate every field with XML omit empty
    fieldmappers.XmlOmitEmpty,
)

addressXml := address.Map(
    structmappers.Rename("addressXml"),
).MapFields(
    fieldmappers.StripComments,
    xmlMappers,
)
```

We can print out our generated struct type definitions:

```go
fmt.Println(addressJson)
// Output:
// type addressJson struct {
//     Since       time.Time  `json:""`
//     FlatNumber  string `json:""`
//     HouseNumber string `json:""`
//     Street      string `json:""`
//     City        string `json:""`
//     Country     string `json:""`
//     Postcode    string `json:""`
// }

fmt.Println(addressXml)
// Output:
// type addressXml struct {
//     Since       time.Time  `xml:""`
//     FlatNumber  string `xml:""`
//     HouseNumber string `xml:""`
//     Street      string `xml:""`
//     City        string `xml:""`
//     Country     string `xml:""`
//     Postcode    string `xml:""`
// }
```

Let's define a common signature for a conversion function, like we've done in 
previous tutorials:

```go
converterSig := "$fromTo$TO($from $From) $To"
```

Now we can generate source code for functions that convert an Address 
struct value to an addressJson or addressXml struct value:

```go
addressToAddressJson := must(addressJson.Converter(converterSig))
addressToAddressXml := must(addressXml.Converter(converterSig))

fmt.Println(addressToAddressJson)
// Output:
// func addressToAddressJson(address Address) addressJson {
//     return addressJson{
//         // elided for readability...
//     }
// }

fmt.Println(addressToAddressXml)
// Output:
// func addressToAddressXml(address Address) addressXml {
//     return addressXml{
//         // elided for readability...
//     }
// }
```

And we can also generate the reverse mappings:

```go
addressJsonToAddress := must(addressJson.Map(structmappers.Reverse).Converter(converterSig))
addressXmlToAddress := must(addressXml.Map(structmappers.Reverse).Converter(converterSig))

fmt.Println(addressJsonToAddress)
// Output:
// func addressJsonToAddress(address AddressJson) Address {
//     return Address{
//         // elided for readability...
//     }
// }

fmt.Println(addressXmlToAddress)
// Output:
// func addressXmlToAddress(address AddressXml) Address {
//     return Address{
//         // elided for readability...
//     }
// }
```


### Nested mappings

Our Person struct type definition is slightly more complicated because
we have to convert the Address field to the generated addressJson or addressXml
type.

Here's an initial implementation:

```go
// map Person to a new PersonJson type
personJson := person.Map(
    structmappers.Rename("personJson"),
).MapFields(
    // convert Address field to type AddressJson
    fieldmappers.Conditionally(
        fieldmappers.FilterTypes("Address"),
        fieldmappers.RewriteType(
            "$Json",
            "$fromTo$TO($.$)",
            "$fromTo$TO($.$)",
        ),
    ),
)
```

We can print the Converter function for a Person to personJson, and the reverse:

```go
fmt.Println(must(personJson.Converter(converterSig)))
fmt.Println(must(personJson.Map(structmappers.Reverse).Converter(converterSig)))
```

This prints:

```go
// personToPersonJson converts [Person] to [personJson].
func personToPersonJson(person Person) personJson {
    return personJson{
        Title:      person.Title,
        GivenName:  person.GivenName,
        FamilyName: person.FamilyName,
        Address:    addressToAddressJson(person.Address),
    }
}
// personJsonToPerson converts [personJson] to [Person].
func personJsonToPerson(personJson personJson) Person {
    return Person{
        Title:      person.Title,
        GivenName:  person.GivenName,
        FamilyName: person.FamilyName,
        Address:    addressJsonToAddress(personJson.Address),
    }
}
```

Notice that these conversion functions, `addressToAddressJson" and 
"addressJsonToAddress" are ones we generated in the previous section.

XML variants are similar.


### Omitting zero-valued structs

We're almost there, but we want to omit a zero-valued Address from a Person. 
We'll do this by representing the value as a pointer.

First we need to some way to detect if an Address is zero. We've learnt
previously how to generate an Equals function (implementing the concept
of being "deeply equal", if necessary):

```go
comparerSig := "$Equals(a $, b $)"

fmt.Println(address.Comparer(comparerSig))
```

This generates a function:

```go
// AddressEquals returns true if two [Address] values are equal.
func AddressEquals(a Address, b Address) bool {
    // omitted for readability
}
```

To detect if an Address is zero, we just need to call AddressEquals with
an address and a zero value.

We'll need a bit of extra machinery for our generated code:

```go
// zeroFn takes a function that returns true if two values are equal, and
// returns a new function that returns true if a value is equal to the zero
// value.
func zeroFn[X comparable](equal func(a X, b X) bool) func(X) bool {
    return func(in X) bool {
        var zero X
        return equal(in, zero)
    }
}

// toPtr returns nil if the input is equal to the zero value, or otherwise
// returns a pointer to the value returned by applying the conversion function
// to the input.
func toPtr[In, Out comparable](in In, zero func(In) bool, convert func(In) Out) *Out {
    if zero(in) { return nil }
    out := convert(in)
    return &out
}

// fromPtr returns the zero value (of type Out) if the input is nil, otherwise
// returns the result of applying the conversion function to the value obtained
// by dereferencing the input pointer.
func fromPtr[In, Out comparable](in *In, convert func(In) Out) Out {
    var zero Out
    if in == nil { return zero }
    out := convert(*in)
    return out
}
```

With that, we have our final personJson mapping:

```go
    personJson = person.Map(
        structmappers.Rename("personJson"),
    ).MapFields(
        fieldmappers.StripComments,
        fieldops.Copy,
        // convert Address field to type *AddressJson
        fieldmappers.Conditionally(
            fieldmappers.FilterTypes("Address"),
            fieldmappers.RewriteType(
                "*$Json",
                "toPtr($.$, zeroFn($FromEquals), $fromTo$TO)",
                "fromPtr($.$, $fromTo$TO)",
            ),
        ),
    )
```

With this, we can generate code:

```go
    addressJsonEquals := must(addressJson.Comparer(comparerSig))
    addressToAddressJson := must(addressJson.Converter(converterSig))
    addressJsonToAddress := must(addressJson.Map(structmappers.Reverse).Converter(converterSig))

    personToPersonJson := must(personJson.Converter(converterSig))
    personJsonToPerson := must(personJson.Map(structmappers.Reverse).Converter(converterSig))

    fmt.Println(personJson)
    fmt.Println(personToPersonJson)
    fmt.Println(personJsonToPerson)

    fmt.Println(addressJson)
    fmt.Println(addressJsonEquals)
    fmt.Println(addressToAddressJson)
    fmt.Println(addressJsonToAddress)
```

Giving generated code like:

```go
type addressJson struct { /* ... */ }

// addressJsonEquals returns true if two [addressJson] values are equal.
func addressJsonEquals(a addressJson, b addressJson) bool { /* ... */ }

// addressToAddressJson converts [Address] to [addressJson].
func addressToAddressJson(address Address) addressJson { /* ... */ }

// addressJsonToAddress converts [addressJson] to [Address].
func addressJsonToAddress(addressJson addressJson) Address { /* ... */ }
    
type personJson struct {
    /* ... */
    Address    *AddressJson // from Address
}

// personToPersonJson converts [Person] to [personJson].
func personToPersonJson(person Person) personJson {
    return personJson{
        /* ... */
        Address: toPtr(person.Address, zeroFn(AddressEquals), addressToAddressJson),
    }
}

// personJsonToPerson converts [personJson] to [Person].
func personJsonToPerson(personJson personJson) Person {
    return Person{
        /* ... */
        Address: fromPtr(personJson.Address, addressJsonToAddress),
    }
}
```

And so on for XML.

We generated types with lowercase first letter names, keeping them private to
the package.

We can tie this all up neatly behind custom marshallers and unmarshallers:

```go
func (p Person) MarshalJSON() ([]byte, error) {
    return json.Marshal(personToPersonJson(p))
}

func (p *Person) UnmarshalJSON(b []byte) error {
    var dest personJson
    err := json.Unmarshal(b, &dest)
    if err == nil {
        *p = personJsonToPerson(dest)
    }
    return err
}
```

---

**Next:** [Mapping to column-orientated data types.](./column-orientated-structs.md)
