// Package morph generates Go code to map values between related struct types
//
// - without runtime reflection.
//
// - without stuffing a new domain-specific language into struct field tags.
//
// - with a simple, fully programmable mapping described in native Go code.
//
// - where you can map to existing types, or use Morph to automatically generate
// new types.
//
// Developed by [Tawesoft Ltd].
//
// [Tawesoft Ltd]: https://www.tawesoft.co.uk/
package morph

import (
    "bytes"
    "fmt"
    "go/format"
    "strings"

    "github.com/tawesoft/morph/internal/parse"
)

func must[T any](result T, err error) T {
    if err == nil { return result }
    panic(err)
}

// Morpher is the type of a function that implements how a field of a given
// name and type is mapped to another field.
//
// The morpher is called for each field on an input struct, and each call to
// the emit function can be used to create a mapping from that field to a field
// on another struct. It is permitted to call emit zero, one, or more than one
// time to produce zero, one, or more fields from a single input field.
//
// As a special case, when emit is called, the character "$" is replaced in the
// emit name argument with the value of the input field name, and the character
// "$" is replaced in the emit value argument with "from.Name" with Name set to
// the input field name and "from" set to the name of the input argument. This
// is lexical, meaning the output is text. For example, `emit("$Len", "int",
// "len($)")` called on a field with name "Message" means `emit("MessageLen",
// "int", "len(from.Message")`. In the value argument, a dollar sign inside a
// string literal is ignored. The name, Type, and value must parse to a valid
// identifier, type expression, and value expression respectively.
type Morpher func(name, Type string, emit func(name, Type, value string))

// Struct is a parsed description of a struct returned by [Parse].
type Struct struct {
    p *parse.Struct
}

func (s Struct) Name() string {
    return s.p.Name
}

// Signature describes the function signature of the mapping function that will
// be generated for values of the source Struct type, e.g.
// "FooToBar(from Foo) Bar".
//
// At least one of the function arguments must be a value or a pointer to a
// value of the type of the source [Struct]. In the event of multiple matches,
// the first match is considered. The name of this argument is used when
// a [Morpher] substitutes "$" for field names.
//
// There must be exactly one return value, of the desired result type, or a
// pointer to a value of that type.
//
// Generic type constraints are permitted, but are ignored when considering
// if types match.
type Signature struct {
    s *parse.FuncSig
}

func (s Signature) String() string {
    return fmt.Sprintf("%+v", s.s)
}

// ParseStruct parses a given source file, looking for a struct with the given
// name.
//
// If src != nil, ParseStruct parses the source from src and the filename is
// only used when recording position information. The type of the argument for
// the src parameter must be string, []byte, or io.Reader. If src == nil, Parse
// parses the file specified by filename. This matches the behavior of
// [go.Parser/ParseFile].
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
func ParseStruct(filename string, src any, name string) (Struct, error) {
    esc := func(err error) (Struct, error) {
        return Struct{}, fmt.Errorf("error parsing %q for struct %q: %w", filename, name, err)
    }

    var result *parse.Struct
    filter := func(s string) bool {
        return name == s
    }
    err := parse.File(filename, src, filter, func(s parse.Struct) {
        result = &s
    })
    if err != nil { return esc(err) }
    if result == nil { esc(fmt.Errorf("struct not found")) }
    return Struct{result}, nil
}

func ParseSignature(s string) (Signature, error) {
    funcSig, err := parse.FunctionSignature(s)
    return Signature{funcSig}, err
}

type assignment struct {
    Name string
    Type string
    Value string
}

// Morph generates the AST for a function that maps a value of a source struct
// type to a value of another struct type, and the AST for struct type
// definition of a destination struct type.
//
// The output function is specified by the given Signature, and the given
// Morpher function specifies a mapping from each field in the source Struct
// to fields in the output.
//
// Use [go/format.Node] to turn the returned AST into code.
func Morph(source Struct, signature Signature, morpher Morpher) (fn string, def string, err error) {
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error morphing %q: %v", source.Name(), r)
        }
    }()

    // First we need to find the name of the first input argument that has
    // a type that matches the source, ignoring type constraints.
    inputName, ok := getInputName(source.Name(), signature.s.Arguments)
    if !ok {
        panic(fmt.Sprintf("function signature missing input argument for type %q", source.Name()))
    }

    returnType, ok := getReturnType(signature.s.Results)
    if !ok {
        panic(fmt.Sprintf("function signature has invalid return arguments"))
    }

    var assignments []assignment

    for _, field := range source.p.Fields {
        emit := func(name string, Type string, value string) {
            name = strings.ReplaceAll(name, "$", field.Name)
            value = strings.ReplaceAll(value, "$", fmt.Sprintf("%s.%s", inputName, field.Name))
            value = fmt.Sprintf("%s(%s)", Type, value)
            assignments = append(assignments, assignment{
                Name:  name,
                Type:  Type,
                Value: value,
            })
        }
        morpher(field.Name, field.Type, emit)
    }

    fns := generateFunc(signature, returnType, assignments)
    defs := generateStruct(returnType, assignments)
    fn = string(must(format.Source(fns)))
    def = string(must(format.Source(defs)))

    return fn, def, nil
}

func getInputName(Type string, args []parse.Field) (string, bool) {
    // Find the first input argument that matches the given Type, allowing for
    // pointers or generic type constraints.
    for _, arg := range args {
        if arg.ShortType() == Type {
            return arg.Name, true
        }
    }
    return "", false
}

func getReturnType(returns []parse.Field) (string, bool) {
    if len(returns) != 1 { return "", false }
    r := returns[0]
    return r.Type, true
}

func generateFunc(signature Signature, returnType string, assignments []assignment) []byte {
    var sb bytes.Buffer
    fmt.Fprintf(&sb, "func %s {\n\treturn %s{\n",
        signature.s.Source,
        returnType,
    )

    for _, asgn := range assignments {
        fmt.Fprintf(&sb, "\t\t%s: %s,\n", asgn.Name, asgn.Value)
    }

    sb.WriteString("\t}\n}\n")
    return sb.Bytes()
}

func generateStruct(returnType string, assignments []assignment) []byte {
    var sb bytes.Buffer
    fmt.Fprintf(&sb, "type %s struct {\n",
        returnType,
    )

    for _, asgn := range assignments {
        fmt.Fprintf(&sb, "\t%s %s\n", asgn.Name, asgn.Type)
    }

    sb.WriteString("\t}\n")
    return sb.Bytes()
}
