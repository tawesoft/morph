// Package morph generates Go code to map between structs...
//
// - without runtime reflection.
//
// - without stuffing a new domain-specific language into struct field tags.
//
// - with a simple, fully programmable mapping described in native Go code.
//
// - where you can map to existing types, or use Morph to automatically generate
//  new types.
//
// Developed by [Tawesoft Ltd].
//
// [Tawesoft Ltd]: https://www.tawesoft.co.uk/
//
// # Security Model
//
// WARNING: It is assumed that all inputs are trusted. DO NOT accept arbitrary
// input from untrusted sources under any circumstances, as this will parse
// and generate arbitrary code.
package morph

import (
    "bytes"
    "fmt"
    "go/format"
    "go/parser"
    "strings"
)

func must[T any](result T, err error) T {
    if err == nil { return result }
    panic(err)
}

// functionSignature represents a parsed function signature, including
// arguments, return types, method reciever, generic type constraints, etc.
//
// Source is the signature as it appears in the source code e.g.
// "Foo[T any](x T) T" or "(x *Foo[T]) Bar() (a Apple, B Banana)".
type functionSignature struct {
    Source string
    Name string
    Type []Field
    Arguments []Field
    Returns  []Field
    Receiver Field
}

// Field represents a name and value, such as a field in a struct, or a type
// constraint, or a functionSignature argument. In a struct, a field may also contain a
// tag.
type Field struct {
    Name string
    Type string
    Tag  string
}

// Struct represents a Go struct - it's name, type constraints (if using
// generics), and fields.
type Struct struct {
    Name string
    TypeParams []Field
    Fields []Field
}

// String returns a source code representation of the given struct.
func (s Struct) String() string {
    var sb bytes.Buffer
    sb.WriteString("type ")
    sb.WriteString(s.Name)
    if len(s.TypeParams) > 0 {
        sb.WriteRune('[')
        for i, tp := range s.TypeParams {
            if i > 0 { sb.WriteString(", ") }
            sb.WriteString(tp.Name)
            sb.WriteRune(' ')
            sb.WriteString(tp.Type)
        }
        sb.WriteRune(']')
    }
    sb.WriteString(" struct {\n")

    for _, field := range s.Fields {
        sb.WriteString("\t")
        sb.WriteString(field.Name)
        sb.WriteRune(' ')
        sb.WriteString(field.Type)
        if len(field.Tag) > 0 {
            sb.WriteString(fmt.Sprintf(" %q", field.Tag))
        }
        sb.WriteRune('\n')
    }

    sb.WriteString("}\n")
    bytes := sb.Bytes()
    out, err := format.Source(bytes)
    if err != nil { panic(fmt.Errorf("error formatting struct %s: %w", bytes, err)) }
    return string(out)
}

// Struct generates Go source code for a new struct type definition based on a
// source struct type definition.
//
// The generated struct's identifier (and type arguments if the type is generic)
// is controlled by the signature argument e.g. "Apple" or "Orange[X any]". Omit
// the "type" and "struct" keywords.
//
// The user-defined generator is called for each field defined on the input
// struct with the name, type, and tag of the field, and an emit function. Each
// invocation of the emit function generates a field on the output struct.
//
// It is permitted to call emit zero, one, or more than one time to
// produce zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name, in the Type argument with the
// source field type, and in the tag argument with the source field's tag.
//
// Note that, matching the behaviour of the Go parser, the emitted tag should
// include any surrounding quote marks.
func (source Struct) Struct(
    signature string,
    generator StructGenerator,
) (Struct, error) {
    return _struct(source, signature, generator)
}

// StructFunc is like [Struct.Struct], except that it takes a function instead
// of an interface.
func (source Struct) StructFunc(signature string, generator StructGeneratorFunc) (Struct, error) {
    return source.Struct(signature, generator)
}

// StructGenerator is the user-supplied argument to [Struct.Struct] that is
// called for every input field.
type StructGenerator interface {
    Field(name, Type, tag string, emit func(name, Type, tag string))
}

// StructGeneratorFunc is a function that satisfies the single method interface
// of [StructGenerator].
type StructGeneratorFunc func (name, Type, tag string, emit func(name, Type, tag string))
func (f StructGeneratorFunc) Field(name, Type, tag string, emit func(name, Type, tag string)) {
    f(name, Type, tag, emit)
}

// Function generates Go source code for a function that maps a value of a
// source struct type to a value of another struct type.
//
// The function is generated using the provided signature, which must contain
// at least one method receiver or argument matching the source struct type (or
// a pointer to a struct of that type) (the first such occurrence is selected,
// including any method receiver), and one return argument, which must be the
// type of the generated struct (or a pointer to a struct of that type). Omit
// the leading "func" keyword. For example, "(s *Store) AppleToOrange(ctx
// context.Context, a Apple) Orange".
//
// The user-defined generator is called for each field defined on the input
// struct with the name, type, and tag of the field, and an emit function. Each
// invocation of the emit function generates a field on the output struct.
//
// It is permitted to call emit zero, one, or more than one time to produce
// zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name (e.g. "foo"), and in the
// value argument with the qualified source field name (e.g. "from.foo").
func (source Struct) Function(signature string, generator FunctionGenerator) (string, error) {
    return _function(source, signature, generator)
}

// FunctionFunc is like [Struct.Function], except that it takes a function
// instead of an interface.
func (source Struct) FunctionFunc(signature string, generator FunctionGeneratorFunc) (string, error) {
    return source.Function(signature, generator)
}

// FunctionGenerator is the user-supplied argument to [Struct.Function] that is
// called for every input field.
type FunctionGenerator interface {
    Field(name, Type, tag string, emit func(name, value string))
}

// FunctionGeneratorFunc is a function that satisfies the single method interface
// of [FunctionGenerator].
type FunctionGeneratorFunc func(name, Type, tag string, emit func(name, value string))
func (f FunctionGeneratorFunc) Field(name, Type, tag string, emit func(name, value string)) {
    f(name, Type, tag, emit)
}

func _struct(source Struct, signature string, generator StructGenerator) (result Struct, err error) {
    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error morphing struct %q to struct %q: %v", source.Name, signature, r)
        }
    }()

    src := `package temp; type `+signature+` struct {}`
    result = must(ParseStruct("temp.go", src, ""))

    for _, field := range source.Fields {
        emit := func(name, Type, tag string) {
            // TODO escape sequence for $
            name = strings.ReplaceAll(name, "$", field.Name)
            Type = strings.ReplaceAll(Type, "$", field.Type)
            tag  = strings.ReplaceAll(tag,  "$", field.Tag)
            result.Fields = append(result.Fields, Field{
                Name:  name,
                Type:  Type,
                Tag:   tag,
            })
        }
        generator.Field(field.Name, field.Type, field.Tag, emit)
    }

    return result, nil
}

func _function(source Struct, signature string, morpher FunctionGenerator) (generated string, err error) {
    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error generating morph func %q for struct %q: %v", signature, source.Name, r)
        }
    }()

    fn := must(parseFunctionSignature(signature))

    // First we need to find the name of the first input argument that has
    // a type that matches the source, ignoring type constraints.
    var args []Field
    args = append(args, fn.Receiver)
    args = append(args, fn.Arguments...)
    args = filterFields(args, func(f Field) bool {
        x, err := parser.ParseExpr(f.Type)
        if err != nil { return false }
        s, ok := simpleTypeExpr(x)
        if !ok { return false }
        // trim type constraints to ignore them
        {
            idx := strings.IndexByte(s, '[')
            if idx > 0 {
                s = s[0:idx]
            }
        }
        return (s == source.Name) || (s == "*" + source.Name)
    })
    if len(args) < 1 {
        panic(fmt.Errorf("could not find matching argument for source"))
    }
    inputArg := args[0]

    // we also need to find the return argument
    returns, ok := fn.singleReturn()
    if !ok {
        panic(fmt.Errorf("functionSignature must have single return value"))
    }

    type assignment struct {
        Name string
        Value string
    }

    var asgns []assignment
    for _, field := range source.Fields {
        emit := func(name, value string) {
            // TODO escape sequence for $
            name = strings.ReplaceAll(name, "$", field.Name)
            value = strings.ReplaceAll(value, "$", inputArg.Name+"."+field.Name)
            asgns = append(asgns, assignment{
                Name:  name,
                Value: value,
            })
        }
        morpher.Field(field.Name, field.Type, field.Tag, emit)
    }

    // source code representation
    var sb bytes.Buffer
    sb.WriteString("func ")
    sb.WriteString(fn.Source)
    sb.WriteString(" {\n")
    sb.WriteString("\treturn ", )

    // For type *Foo, return &Foo
    if strings.HasPrefix(returns.Type, "*") {
        sb.WriteRune('&')
        sb.WriteString(returns.Type[1:])
    } else {
        sb.WriteString(returns.Type)
    }

    sb.WriteString("{\n")
    for _, asgn := range asgns {
        sb.WriteString(fmt.Sprintf("\t\t%s: %s,\n", asgn.Name, asgn.Value))
    }
    sb.WriteString("\t}\n")
    sb.WriteString("}\n")
    bytes := sb.Bytes()
    out, err := format.Source(bytes)
    if err != nil { panic(fmt.Errorf("error formatting functionSignature %s: %w", string(bytes), err)) }
    return string(out), nil
}
