package parse

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "go/types"
    "strings"
)

type Field struct {
    Name string
    Type string
}

// ShortType returns a field's type up to the first array or type delimiter '[',
// excluding any leading pointer specifiers '*'+.
//
// This can be used to match a field's type while ignoring type constraints.
func (f Field) ShortType() string {
    idx := strings.IndexByte(f.Type, '[')
    if idx < 0 { return f.Type }
    s := f.Type[0:idx]
    return strings.TrimLeft(s, "* \t")
}

type Struct struct {
    Name string
    Fields []Field
}

type FuncSig struct {
    Source string // e.g. Foo[T any](x T) T
    Name string
    Type []Field
    Arguments []Field
    Results []Field
}

// fields converts an ast.FieldList into []parse.Field
func fields(fieldList *ast.FieldList) []Field {
    if fieldList == nil { return nil }
    result := []Field{}
    for _, field := range fieldList.List {
        fieldType := types.ExprString(field.Type)

        for _, fieldName := range field.Names {
            result = append(result, Field{fieldName.String(), fieldType})
        }
        if len(field.Names) == 0 {
            // e.g. embedded field Foo in struct Bar:
            //     type Foo struct { ... }
            //     type Bar struct { Foo }
            // This is treated as a field with name Foo.
            result = append(result, Field{fieldType, fieldType})
        }
    }
    return result
}

/* no longer used

// Fragment parses a fragment of a Go program into an AST.
//
// Parsing is performed without full object resolution.
func Fragment(src string) ([]ast.Decl, error) {
    src = "package temp\n"+src+"\n"
    pflags := parser.DeclarationErrors | parser.SkipObjectResolution
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, "temp.go", src, pflags)
    if err != nil {
        return nil, fmt.Errorf("error parsing fragment: %v", err)
    }
    return astf.Decls, nil
}

// Function parses a single function definition into an AST
func Function(src string) (*ast.FuncDecl, error) {
    decls, err := Fragment(src)
    if err != nil { return nil, err }
    if len(decls) != 1 {
        return nil, fmt.Errorf("error parsing function: expected exactly one ast.Decl")
    }
    funcDecl, ok := decls[0].(*ast.FuncDecl)
    if !ok {
        return nil, fmt.Errorf("error parsing function: not a ast.funcDecl")
    }
    return funcDecl, nil
}
*/

// FunctionSignature parses the source code of a single function signature, such as
// "Foo(a A) B", and returns the AST.
//
// Parsing is performed without full object resolution.
func FunctionSignature(signature string) (*FuncSig, error) {

    // ParseExpr doesn't work because we can't make a named function an
    // expression, so we have to create a whole dummy AST for a file.
    src := `package temp; func `+signature+` {}`

    pflags := parser.DeclarationErrors | parser.SkipObjectResolution
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, "temp.go", src, pflags)
    if err != nil {
        return nil, fmt.Errorf("error parsing function signature %q: %v", signature, err)
    }

    var result *FuncSig

    ast.Inspect(astf, func(n ast.Node) bool {
        if result != nil { return false }
        if n == nil { return false }

        funcDecl, ok := n.(*ast.FuncDecl)
        if !ok { return true }

        result = &FuncSig{
            Source:    signature,
            Name:      funcDecl.Name.String(),
            Type:      fields(funcDecl.Type.TypeParams),
            Arguments: fields(funcDecl.Type.Params),
            Results:   fields(funcDecl.Type.Results),
            // TODO funcDecl.Recv for method receiver type if needed
        }

        return true
    })

    if result == nil {
        return nil, fmt.Errorf("error parsing function signature %q: no function found", signature)
    }

    return result, nil
}

// File parses the source code of a single source file and records top-level
// struct type definitions, calling the record function for each found.
//
// If filter != nil, struct type definitions are only collected if filter(name)
// returns true for the struct type definition with that name.
//
// If src != nil, File parses the source from src and the filename is only used
// when recording position information. The type of the argument for the src
// parameter must be string, []byte, or io.Reader. If src == nil, File
// parses the file specified by filename. This matches the behavior of
// [go.Parser/ParseFile].
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
func File(filename string, src any, filter func(name string) bool, record func(Struct)) error {
    pflags := parser.DeclarationErrors | parser.SkipObjectResolution
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, filename, src, pflags)
    if err != nil {
        return fmt.Errorf("error parsing %q: %v", filename, err)
    }

    ast.Inspect(astf, func(n ast.Node) bool {
        switch x := n.(type) {
            case *ast.GenDecl:
                if (x.Tok != token.TYPE) || (len(x.Specs) != 1) { return false }
                typeSpec := x.Specs[0].(*ast.TypeSpec)
                structType, ok := typeSpec.Type.(*ast.StructType)
                if !ok { return false }

                structName := typeSpec.Name.String()
                if (filter != nil) && (!filter(structName)) { return false }

                s := Struct{
                    Name: structName,
                    Fields: fields(structType.Fields),
                }

                record(s)

                // Typed struct (not needed)...
                // e.g. "type Foo[TypeParams...] struct"
                // if typeSpec.TypeParams != nil {

                return false
            case *ast.FuncDecl:
                // globally-scoped structs only
                return false
        }
        return true
    })

    return nil
}
