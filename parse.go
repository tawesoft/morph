package morph

import (
    "bytes"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "go/types"
)

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

// singleReturn returns the return type for a functionSignature and true when
// there is exactly one return value, or (_, false) otherwise.
func (f functionSignature) singleReturn() (Field, bool) {
    if len(f.Returns) != 1 { return Field{}, false }
    return f.Returns[0], true
}

// parseFunctionSignature parses the source code of a single functionSignature
// signature, such as `Foo(a A) B`.
//
// Parsing is performed without full object resolution.
func parseFunctionSignature(signature string) (result functionSignature, err error) {

    esc := func(err error) (functionSignature, error) {
        return functionSignature{}, fmt.Errorf("error parsing functionSignature signature %q: %w", signature, err)
    }

    // ParseExpr doesn't work because we can't make a named functionSignature an
    // expression, so we have to create a whole dummy AST for a file.
    src := `package temp; func `+signature+` {}`

    pflags := parser.DeclarationErrors | parser.SkipObjectResolution
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, "temp.go", src, pflags)
    if err != nil { return esc(err) }

    found := false
    ast.Inspect(astf, func(n ast.Node) bool {
        if found { return false }
        if n == nil { return false }

        funcDecl, ok := n.(*ast.FuncDecl)
        if !ok { return true }

        result = functionSignature{
            Source:    signature,
            Name:      funcDecl.Name.String(),
            Type:      fields(funcDecl.Type.TypeParams),
            Arguments: fields(funcDecl.Type.Params),
            Returns:   fields(funcDecl.Type.Results),
        }
        if funcDecl.Recv != nil {
            result.Receiver = fields(funcDecl.Recv)[0]
        }
        found = true
        return false
    })

    if !found { return esc(fmt.Errorf("not found")) }
    return result, nil
}

// simpleTypeExpr returns a type formatted as a string if the type is simple
// (i.e. not a map, slice, channel etc.). Otherwise, returns (_, false).
//
// This is used to find the first functionSignature argument (or receiver) that
// matches a given type.
func simpleTypeExpr(x ast.Expr) (string, bool) {
    var buf bytes.Buffer
    ok := writeSimpleTypeExpr(&buf, x)
    return buf.String(), ok
}

// writeSimpleTypeExpr is a shortened version of [types.ExprString] used by
// [simpleTypeExpr] to format a type as a string, excluding several features
// not needed for our purpsoes such as map types.
func writeSimpleTypeExpr(buf *bytes.Buffer, x ast.Expr) bool {
    unpackIndexExpr := func(n ast.Node) (x ast.Expr, lbrack token.Pos, indices []ast.Expr, rbrack token.Pos) {
        switch e := n.(type) {
            case *ast.IndexExpr:
                return e.X, e.Lbrack, []ast.Expr{e.Index}, e.Rbrack
            case *ast.IndexListExpr:
                return e.X, e.Lbrack, e.Indices, e.Rbrack
        }
        return nil, token.NoPos, nil, token.NoPos
    }

    switch x := x.(type) {
        default:
            return false

        case *ast.Ident:
            buf.WriteString(x.Name)

        case *ast.BasicLit:
            buf.WriteString(x.Value)

        case *ast.SelectorExpr:
            ok := writeSimpleTypeExpr(buf, x.X)
            if !ok { return false }
            buf.WriteByte('.')
            buf.WriteString(x.Sel.Name)

        case *ast.IndexExpr, *ast.IndexListExpr:
            ixX, _, ixIndices, _ := unpackIndexExpr(x)
            ok := writeSimpleTypeExpr(buf, ixX)
            if !ok { return false }
            buf.WriteByte('[')
            ok = writeSimpleTypeExprList(buf, ixIndices)
            if !ok { return false }
            buf.WriteByte(']')

        case *ast.StarExpr:
            buf.WriteByte('*')
            return writeSimpleTypeExpr(buf, x.X)
    }
    return true
}

func writeSimpleTypeExprList(buf *bytes.Buffer, list []ast.Expr) bool {
    for i, x := range list {
        if i > 0 {
            buf.WriteString(", ")
        }
        ok := writeSimpleTypeExpr(buf, x)
        if !ok { return false }
    }
    return true
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
