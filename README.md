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





------

Motivation
----------

For example, imagine you are creating a web application to manage an 
educational institution. Your code may have several distinct models to 
represent a course in different ways:

* A model of textboxes, buttons, and other GUI components, where an 
  administrator can edit the course.
* A model of the course as a HTTP POST request when the administrator saves
  their changes.
* A model of the course as it relates to business logic, where you can do
  things like validate that the course isn't oversubscribed.
* A model of the course with fewer details e.g. for public display.
* A model of the course as JSON, as part of an API.
* A model of the course that can be written to, or read from, a disk or a 
  database so that changes persist.

It is possible to overload a single model to achieve all these use-cases,
for example in the way that Go structs can have annotations describing 
their JSON or XML mapping. But it is often a better design to have multiple 
models and convert between them. For example, you might have separate structs 
named "CourseWebform", "CourseHTTPPost", "Course", "CoursePublic", 
"CourseJSON", and "CourseDisk".

Though trivial, mapping between values of these types involves a lot of
tedious boilerplate where it is easy for mistakes to slip in. Additionally, 
reflection based on annotations can be slow and many errors are often
caught only at runtime, rather than compile-time.

That's why we created Morph.


Walkthrough
-----------

For example, a multimedia document may have the following representation at
runtime:

```go
type Document struct {
    Title string
    Contents EditableText // some efficient data structure e.g. a Rope
    Created, Modified, Accessed time.Time
    Pictures []Picture
    SomeRuntimeOnlyInfo *Foo
}

type Picture struct {
    Label string
    File string
}
```

But, when it comes to saving to disk, you may want it represented as:

```go
type DocumentDisk struct {
    RecordSize int64
    Created, Modified, Accessed int64 // seconds since Unix epoch
    ContentsSize int64
    TitleSize int32
    
    Contents string
    Title string
    
    Pictures []PictureDisk
}

type PictureDisk struct {
    LabelSize int32
    Label string
    
    Data func() io.ReadCloser // Reads from disk
}
```

Or for storing in a database, you may want it represented as:

```go
type DocumentSqlite struct {
    GUID [16]byte
    Created, Modified, Accessed int64 // seconds since Unix epoch
    Contents string
}

type DocumentPictureSqlite {
    ParentGUID [16]byte // parent key modelling a one-to-many relationship
    Label string
    File string
}
```

Though trivial, mapping between values of these types involves a lot of 
boilerplate where it is easy for mistakes to slip in:

```go
func map_Document_DocumentDisk(from Document) DocumentDisk {
    // the old way of doing this!
    
    size := 123 // some calculation...
    for _, p := range from.Pictures {
        size += (p * 123)
    }
    
    return DocumentDisk{
        RecordSize: size,
        Created: from.Created.Unix(),
        Modified: from.Modified.Unix(),
        Accessed: from.Accessed.Unix(),
        ContentsSize: strlen(from.Contents),
        TitleSize: strlen(from.Title),
        Contents: from.Contents.String(),
        Title: from.Title,
        Pictures: mapSlice(map_Picture_PictureDisk)(from.Pictures),
    }
}
```

Additionally, going back and adding a new field now requires updating the 
related models all over the place. If you forget to update one, you might not
detect any errors as a field will silently be treated as missing.

Instead, we can generate this mapping code automatically by defining (in Go)
a function that operates on each field's name and type and calls a function
f(name, type, value) which generates conversion code.

While this may appear "stringly-typed", it generates code that can be checked
at compile-time.

The dollar sign "$" may appear in the third argument to represent the input 
value at any time.

```go
func _Document_to_DocumentDisk_field(name string, t string, f(string, string, string)) {
    if name == "" {
        // When name is empty, we can specify additional fields
        f("RecordSize", "int64", "morph.SizeOf($)"))
    } else if t == "time.Time" {
        // If the input is any time type, return a new field with the same name
        // that calls the .Unix() method
        f(name, "int64", "$.Unix()")
    } else if t == "string" {
        // If the input is a string, return two new fields, one with the same
        // name and content, and one with "Size" appended to the name
        f(name + "Size", "int64",  "strlen($)")
        f(name,          "string", "$"
    } else if t == "EditableText" {
        // Another string-like type
        f(name + "Size", "int64",  "$.Length()", name)
        f(name,          "string", "$.String()", name)
    } else if name == "Pictures" {
        f(name, "[]PictureDisk", "map(_Picture_to_PictureDisk, $)")
    } else if name == "SomeRuntimeOnlyInfo" {
        // ignored
    } else {
        panic(fmt.Sprintf("unrecognised field %q of type %q", name, t))
    }
}

func _Picture_to_PictureDisk_field(name string, t string, f(string, string, string)) {
    if name == "Label" {
        f(name + "Size", "int64",  "strlen($)")
        f(name,          "string", "$"
    } else if name == "File" {
        f("Data", "io.ReadCloser", "must(os.Open($))"
    }
}
```

Here, SIZEOF expands to a string that contains runtime code to calculate the 
size of the structure (taking into account any padding or variable length 
content such as strings) plus the size of any child structures.

Similarly, for the database example:

```go
func _Document_to_DocumentSqlite_field(name string, t string, f(string, string, string)) {
    if t == "time.Time" {
        // If the input is any time type, return a new field with the same name
        // that calls the .Unix() method
        f(name, "int64", "$.Unix()")
    } else if t == "EditableText" {
        // Another string-like type
        f(name, "string", "$.String()", name)
    } else if name == "SomeRuntimeOnlyInfo" || name == "Pictures" {
        // ignored
    } else {
        panic(fmt.Sprintf("unrecognised field %q of type %q", name, t))
    }
}

func _Picture_to_DocumentPictureSqlite_field(parent DocumentSqlite, name string, t string, f(string, string, string)) {
    f(name, t, "from.$")
}

func _Picture_to_DocumentPictureSqlite_additional(f(string, string, string)) {
    f("ParentGUID", [16]byte, "parent.GUID")
}

func _Document_to_DocumentSqlite_additional(f(string, string, string)) {
    f("GUID", "[16]byte", "generateUUID()")
}
```

Here, the DocumentPictureSqlite structures are not serialized as they are not 
contained by the DocumentSqlite structure itself. Because they require a
reference to a parent, the function signature may be overridden:

```
func _Picture_to_PictureSqlite(parent DocumentSqlite, from Picture, to DocumentPictureSqlite) {
    // The body of this function is generated automatically.
}
```

The mapping is reversed like so, assuming some "resources" value that contains
(or can access( the DocumentPictureSqlite records associated with a document:

```go
func _DocumentSqlite_to_Document_additional(name string, t string, f(string, string, string)) {
    f("pictures", "[]Pictures", "resources.GetPicturesForDocumentSqlite(from.GUID)")
}

func _DocumentSqlite_to_Document(resources Resources, from DocumentSqlite, to Document) {
    // The body of this function is generated automatically.
}
```
