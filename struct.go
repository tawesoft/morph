package morph

import (
    "bytes"
    "fmt"
    "go/ast"
    "go/format"
    "go/parser"
    "go/token"
    "go/types"
    "strings"
)

type Field struct {
    Name string
    Type string
    Tag  string
}

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

// fields converts an ast.FieldList into []Field
func fields(fieldList *ast.FieldList) []Field {
    if fieldList == nil { return nil }
    result := []Field{}
    for _, field := range fieldList.List {
        fieldType := types.ExprString(field.Type)

        for _, fieldName := range field.Names {
            var tag string
            if field.Tag != nil { tag = field.Tag.Value }
            result = append(result, Field{fieldName.String(), fieldType, tag})
        }
        if len(field.Names) == 0 {
            // e.g. embedded field Foo in struct Bar:
            //     type Foo struct { ... }
            //     type Bar struct { Foo }
            // This is treated as a field with name Foo.
            result = append(result, Field{fieldType, fieldType, ""})
        }
    }
    return result
}

func filterFields(fields []Field, filter func(f Field) bool) []Field {
    result := []Field{}
    for _, f := range fields {
        if filter(f) {
            result = append(result, f)
        }
    }
    return result
}

// ParseStruct parses a given source file, looking for a struct with the given
// name.
//
// If name == "", ParseStruct returns the first struct found.
//
// If src != nil, ParseStruct parses the source from src and the filename is
// only used when recording position information. The type of the argument for
// the src parameter must be string, []byte, or io.Reader. If src == nil, Parse
// parses the file specified by filename. This matches the behavior of
// [go.Parser/ParseFile].
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
func ParseStruct(filename string, src any, name string) (result Struct, err error) {
    esc := func(err error) (Struct, error) {
        return Struct{}, fmt.Errorf("error parsing %q for struct %q: %w", filename, name, err)
    }

    found := false
    pflags := parser.DeclarationErrors | parser.SkipObjectResolution
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, filename, src, pflags)
    if err != nil { return esc(err) }

    ast.Inspect(astf, func(n ast.Node) bool {
        if found { return false }
        switch x := n.(type) {
            case *ast.GenDecl:
                if (x.Tok != token.TYPE) || (len(x.Specs) != 1) { return false }
                typeSpec := x.Specs[0].(*ast.TypeSpec)
                structType, ok := typeSpec.Type.(*ast.StructType)
                if !ok { return false }

                structName := typeSpec.Name.String()
                if (name != "") && (name != structName) { return false }

                result = Struct{
                    Name: structName,
                    TypeParams: fields(typeSpec.TypeParams),
                    Fields: fields(structType.Fields),
                }
                found = true

                return false
            case *ast.FuncDecl:
                // globally-scoped structs only
                return false
        }
        return true
    })

    if !found { return esc(fmt.Errorf("not found")) }
    return result, nil
}

// StructDefinition generates Go source code for a named struct type definition
// based on a source struct type definition.
//
// The generated struct's identifier (and type arguments if the type is generic)
// is controlled by the signature argument e.g. "Apple" or "Orange[X any]". Omit
// the "type" and "struct" keywords.
//
// The user-defined morpher is called for each field defined on the input
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
func StructDefinition(source Struct, signature string, morpher StructDefinitionMorpher) (Struct, error) {
    return structDefinition(source, signature, morpher)
}

// StructDefinitionMorpher is the type of a user-defined function that
// implements how a field in a source struct definition is mapped to field(s)
// in a destination struct definition. See [StructDefinition].
//
// The name, Type and tag arguments are the source field name, type and tag
// respectively. The emit function generates destination fields.
type StructDefinitionMorpher func(name, Type, tag string, emit func(name, Type, tag string))

// StructValue generates Go source code for a function that maps a value of a
// source struct type to a value of another struct type.
//
// The function is generated using the provided signature, which must contain
// at least one argument matching the source struct type (or a pointer to a
// struct of that type) (the first such occurrence is selected, including any
// method receiver), and one return argument, which must be the type of the
// generated struct (or a pointer to a struct of that type). The function
// signature may be a method. Omit the leading "func" keyword. For example, "(s
// *Store) AppleToOrange(ctx context.Context, a Apple) Orange".
//
// The user-defined morpher is called for each field defined on the input
// struct with the name, type, and tag of the field, and an emit function. Each
// invocation of the emit function generates a field on the output struct.
//
// It is permitted to call emit zero, one, or more than one time to produce
// zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name (e.g. "foo"), and in the
// value argument with the qualified source field name (e.g. "from.foo").
func StructValue(source Struct, signature string, morpher StructValueMorpher) (string, error) {
    return structValue(source, signature, morpher)
}

// StructValueMorpher is the type of a user-defined function that implements
// how a field value in a source struct is mapped to field value(s) in a
// destination struct's values. See [StructValue].
//
// The name, Type and tag arguments are the source field name, type and tag
// respectively. The emit function generates destination values.
type StructValueMorpher func(name, Type, tag string, emit func(name, value string))

func structDefinition(source Struct, signature string, morpher StructDefinitionMorpher) (result Struct, err error) {
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
        morpher(field.Name, field.Type, field.Tag, emit)
    }

    return result, nil
}

func structValue(source Struct, signature string, morpher StructValueMorpher) (generated string, err error) {
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
        panic(fmt.Errorf("function must have single return value"))
    }

    type assignment struct {
        Name string
        Value string
    }

    asgns := []assignment{}
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
        morpher(field.Name, field.Type, field.Tag, emit)
    }

    // source code representation
    var sb bytes.Buffer
    sb.WriteString("func ")
    sb.WriteString(fn.Source)
    sb.WriteString(" {\n")
    sb.WriteString("\treturn ", )
    // TODO convert pointer if needed
    sb.WriteString(returns.Type)
    sb.WriteString("{\n")
    for _, asgn := range asgns {
        sb.WriteString(fmt.Sprintf("\t\t%s: %s,\n", asgn.Name, asgn.Value))
    }
    sb.WriteString("\t}\n")
    sb.WriteString("}\n")
    bytes := sb.Bytes()
    out, err := format.Source(bytes)
    if err != nil { panic(fmt.Errorf("error formatting function %s: %w", string(bytes), err)) }
    return string(out), nil
}
