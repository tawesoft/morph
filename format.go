package morph

import (
    "bytes"
    "fmt"
    "go/ast"
    "go/format"
    "go/token"
    "strings"

    "github.com/tawesoft/morph/tag"
)

func formatSource(source string) (string, error) {
    s, err := format.Source([]byte(source))
    if err != nil { return "", err }
    return strings.TrimSpace(string(s)), nil
}

// simpleTypeExpr returns a type formatted as a string if the type is simple
// (i.e. not a map, slice, channel etc.). Otherwise, returns (_, false).
//
// This is used to find the first FunctionSignature argument (or receiver) that
// matches a given type.
func simpleTypeExpr(x ast.Expr) (string, bool) {
    var buf bytes.Buffer
    ok := writeSimpleTypeExpr(&buf, x)
    return buf.String(), ok
}

// writeSimpleTypeExpr is a shortened version of [types.ExprString] used by
// [simpleTypeExpr] to format a type as a string, excluding several features
// not needed for our purposes such as map types.
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
        if !ok {
            return false
        }
        buf.WriteByte('.')
        buf.WriteString(x.Sel.Name)

    case *ast.IndexExpr, *ast.IndexListExpr:
        ixX, _, ixIndices, _ := unpackIndexExpr(x)
        ok := writeSimpleTypeExpr(buf, ixX)
        if !ok {
            return false
        }
        buf.WriteByte('[')
        ok = writeSimpleTypeExprList(buf, ixIndices)
        if !ok {
            return false
        }
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
        if !ok {
            return false
        }
    }
    return true
}

// _function_string implements the Function.String method
func _function_string(fn Function) string {
    var sb strings.Builder
    comment := fn.Signature.Comment
    if len(comment) > 0 {
        for _, line := range strings.Split(comment, "\n") {
            sb.WriteString(fmt.Sprintf("// %s\n", line))
        }
    }
    sb.WriteString("func ")
    sb.WriteString(fn.Signature.String())
    sb.WriteString(" {\n")
    sb.WriteString(fn.Body)
    sb.WriteString("\n}")
    source := sb.String()
    out, err := formatSource(source)
    if err != nil {
        panic(fmt.Errorf(
            "error formatting function %q: %w",
            source, err,
        ))
    }
    return string(out)
}

// _functionSignature_string implements the FunctionSignature.String method.
//
// It omits the leading "func" keyword.
func _functionSignature_string(fn FunctionSignature) string {
    var sb bytes.Buffer
    if fn.Receiver.Type != "" {
        reciever := fmt.Sprintf("(%s %s) ", fn.Receiver.Name, fn.Receiver.Type)
        sb.WriteString(reciever)
    }
    sb.WriteString(fn.Name)
    if len(fn.Type) > 0 {
        sb.WriteRune('[')
        for i, arg := range fn.Type {
            sb.WriteString(arg.Name)
            sb.WriteRune(' ')
            sb.WriteString(arg.Type)
            if (i < len(fn.Type) - 1) {
                sb.WriteRune(',')
            }
        }
        sb.WriteRune(']')
    }
    sb.WriteRune('(')
    for _, arg := range fn.Arguments {
        sb.WriteString(arg.Name)
        sb.WriteRune(' ')
        sb.WriteString(arg.Type)
        sb.WriteRune(',')
    }
    sb.WriteRune(')')
    if len(fn.Returns) > 0 {
        sb.WriteString(" (")
        for _, arg := range fn.Returns {
            sb.WriteString(arg.Name)
            sb.WriteRune(' ')
            sb.WriteString(arg.Type)
            sb.WriteRune(',')
        }
        sb.WriteRune(')')
    }
    return sb.String()
}

// _struct_signature implements the Struct.Signature method.
func _struct_signature(s Struct) string {
    var sb strings.Builder
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
    return sb.String()
}

// _struct_string implements the Struct.String method.
func _struct_string(s Struct) string {
    var sb bytes.Buffer
    if len(s.Comment) > 0 {
        for _, line := range strings.Split(s.Comment, "\n") {
            sb.WriteString(fmt.Sprintf("// %s\n", line))
        }
    }
    sb.WriteString("type ")
    sb.WriteString(s.Signature())
    sb.WriteString(" struct {\n")

    for _, field := range s.Fields {
        multilineComment := strings.ContainsRune(field.Comment, '\n')
        if multilineComment {
            for _, line := range strings.Split(field.Comment, "\n") {
                sb.WriteString("\t// ")
                sb.WriteString(line)
                sb.WriteRune('\n')
            }
        }

        sb.WriteString("\t")
        sb.WriteString(field.Name)
        sb.WriteRune(' ')
        sb.WriteString(field.Type)
        if len(field.Tag) > 0 {
            sb.WriteRune(' ')
            sb.WriteString(tag.Quote(field.Tag))
        }
        if (!multilineComment) && (len(field.Comment) > 0) {
            sb.WriteString(" // ")
            sb.WriteString(field.Comment)
        }
        sb.WriteRune('\n')
    }

    sb.WriteString("}")
    bs := sb.Bytes()
    out, err := format.Source(bs)
    if err != nil {
        panic(fmt.Errorf(
            "error formatting struct %q: %w",
            string(bs), err,
        ))
    }
    return string(out)
}

// _struct_function_Body creates the Body argument for a Function created
// by the Struct.Function method.
func _struct_function_Body(returnType string, assignments []Field) string {
    // source code representation
    var sb bytes.Buffer
    sb.WriteString("\treturn ")

    // For type *Foo, return &Foo
    if strings.HasPrefix(returnType, "*") {
        sb.WriteRune('&')
        sb.WriteString(returnType[1:])
    } else {
        sb.WriteString(returnType)
    }

    sb.WriteString("{\n")
    for _, asgn := range assignments {
        if asgn.Value == "" {
            sb.WriteString(fmt.Sprintf("\t\t// %s is the zero value.\n", asgn.Name))
        } else {
            sb.WriteString(fmt.Sprintf("\t\t%s: %s,\n", asgn.Name, asgn.Value))
        }
    }
    sb.WriteString("\t}")
    return sb.String()
}
