// Package morph is a Go code generator that generates code to map between
// structs and manipulate the form of functions.
//
// Paid commercial support available via [Open Source at Tawesoft].
//
// [Open Source at Tawesoft]: https://www.tawesoft.co.uk/products/open-source-software
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
    if err == nil {
        return result
    }
    panic(err)
}

// FunctionSignature represents a parsed function signature, including
// arguments, return types, method receiver, generic type constraints, etc.
//
// Raw is the signature as it appears in the source code e.g.
// "Foo[T any](x T) T" or "(x *Foo[T]) Bar() (a Apple, B Banana)".
type FunctionSignature struct {
    Raw       string
    Name      string
    Type      []Field
    Arguments []Field
    Returns   []Field
    Receiver  Field
}

// Field represents a name and value, such as a field in a struct, or a type
// constraint, or a FunctionSignature argument. In a struct, a field may also contain a
// tag.
type Field struct {
    Name string
    Type string
    Tag  string
}

// Struct represents a Go struct - it's name, type constraints (if using
// generics), and fields.
type Struct struct {
    Name       string
    TypeParams []Field
    Fields     []Field
}

// String returns a source code representation of the given struct.
func (s Struct) String() string {
    return _struct_string(s)
}

// Struct generates Go source code for a new struct type definition based on a
// source struct type definition.
//
// The generated struct's identifier, and type arguments if the type is
// generic, are controlled by the signature argument. Omit the "type" and
// "struct" keywords.
//
// For example, the signature argument may look something like:
//
//     Orange[X any]
//
// or simply,
//
//     Orange
//
// The user-defined generator is called once for each field defined on the
// input struct with the name, type, and tag of the field, and an emit callback
// function. Each invocation of the emit callback function generates a field on
// the output struct.
//
// It is permitted to call emit zero, one, or more than one time to
// produce zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name, in the Type argument with the
// source field type, and in the tag argument with the source field's tag.
//
// Note that, matching the behaviour of the Go parser, the emitted field tag,
// if any, should include surrounding quote marks.
func (source Struct) Struct(
    signature string,
    generator func(name, Type, tag string, emit func(name, Type, tag string)),
) (Struct, error) {
    return _struct_struct(source, signature, generator)
}

// Function generates Go source code for a function that maps a value of a
// source struct type to a value of another struct type.
//
// The function is generated to match the provided signature, which must
// describe a function with at least one method receiver or argument matching
// the source struct type (or a pointer to a struct of that type) (the first
// such occurrence is selected, including any method receiver), and one return
// argument, which must be the type of the generated struct (or a pointer to a
// struct of that type). Omit the leading "func" keyword.
//
// For example, the signature argument may look something like:
//
//     (s *Store) AppleToOrange(ctx context.Context, a Apple) Orange
//
// The user-defined generator is called once for each field defined on the
// input struct with the name, type, and tag of the field, and an emit callback
// function. Each invocation of the emit callback function generates a field on
// the output struct.
//
// It is permitted to call emit zero, one, or more than one time to produce
// zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name (e.g. "foo"), and in the
// value argument with the qualified source field name (e.g. "from.foo").
func (source Struct) Function(
    signature string,
    generator func(name, Type, tag string, emit func(name, value string)),
) (string, error) {
    return _struct_function(source, signature, generator)
}

type _struct_struct_generator func(name, Type, tag string, emit func(name, Type, tag string))
type _struct_function_generator func(name, Type, tag string, emit func(name, value string))

// _struct_string implements the Struct.String method.
func _struct_string(s Struct) string {
    var sb bytes.Buffer
    sb.WriteString("type ")
    sb.WriteString(s.Name)
    if len(s.TypeParams) > 0 {
        sb.WriteRune('[')
        for i, tp := range s.TypeParams {
            if i > 0 {
                sb.WriteString(", ")
            }
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
    bs := sb.Bytes()
    out, err := format.Source(bs)
    if err != nil {
        panic(fmt.Errorf("error formatting struct %s: %w", bs, err))
    }
    return string(out)
}

// _struct_struct implements the Struct.Struct method
func _struct_struct(source Struct, signature string, generator _struct_struct_generator) (result Struct, err error) {
    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error morphing struct %q to struct %q: %v", source.Name, signature, r)
        }
    }()

    src := `package temp; type ` + signature + ` struct {}`
    result = must(ParseStruct("temp.go", src, ""))

    for _, field := range source.Fields {
        emit := func(name, Type, tag string) {
            // TODO escape sequence for $
            name = strings.ReplaceAll(name, "$", field.Name)
            Type = strings.ReplaceAll(Type, "$", field.Type)
            tag = strings.ReplaceAll(tag, "$", field.Tag)
            result.Fields = append(result.Fields, Field{
                Name: name,
                Type: Type,
                Tag:  tag,
            })
        }
        generator(field.Name, field.Type, field.Tag, emit)
    }

    return result, nil
}

// _struct_function implements the Struct.Function method
func _struct_function(source Struct, signature string, generator _struct_function_generator) (generated string, err error) {
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
        if err != nil {
            return false
        }
        s, ok := simpleTypeExpr(x)
        if !ok {
            return false
        }
        // trim type constraints to ignore them
        {
            idx := strings.IndexByte(s, '[')
            if idx > 0 {
                s = s[0:idx]
            }
        }
        return (s == source.Name) || (s == "*"+source.Name)
    })
    if len(args) < 1 {
        panic(fmt.Errorf("could not find matching argument for source"))
    }
    inputArg := args[0]

    // we also need to find the return argument
    returns, ok := fn.singleReturn()
    if !ok {
        panic(fmt.Errorf("FunctionSignature must have single return value"))
    }

    type assignment struct {
        Name  string
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
        generator(field.Name, field.Type, field.Tag, emit)
    }

    // source code representation
    var sb bytes.Buffer
    sb.WriteString("func ")
    sb.WriteString(fn.Raw)
    sb.WriteString(" {\n")
    sb.WriteString("\treturn ")

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
    bs := sb.Bytes()
    out, err := format.Source(bs)
    if err != nil {
        panic(fmt.Errorf("error formatting FunctionSignature %s: %w", string(bs), err))
    }
    return string(out), nil
}
