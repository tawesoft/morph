package morph

import (
    "bytes"
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
)

type function struct {
    Source string // e.g. Foo[T any](x T) T
    Name string
    Type []Field
    Arguments []Field
    Returns  []Field
    Receiver Field
}

// singleReturn returns the return type for a function and true when there is
// exactly one return value, or (_, false) otherwise.
func (f function) singleReturn() (Field, bool) {
    if len(f.Returns) != 1 { return Field{}, false }
    return f.Returns[0], true
}

// parseFunctionSignature parses the source code of a single function
// signature, such as `Foo(a A) B`.
//
// Parsing is performed without full object resolution.
func parseFunctionSignature(signature string) (result function, err error) {

    esc := func(err error) (function, error) {
        return function{}, fmt.Errorf("error parsing function signature %q: %w", signature, err)
    }

    // ParseExpr doesn't work because we can't make a named function an
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

        result = function{
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
// This is used to find the first function argument (or receiver) that matches
// a given type.
func simpleTypeExpr(x ast.Expr) (string, bool) {
    var buf bytes.Buffer
    ok := writeSimpleTypeExpr(&buf, x)
    return buf.String(), ok
}

// writeSimpleTypeExpr is a shortened version of [types.ExprString] used by
// [simpleType] to format a type as a string, excluding several features not
// needed for our purpsoes such as map types.
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
