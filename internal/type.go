package internal

import (
    "bytes"
    "go/ast"
    "go/parser"
    "go/token"
    "strings"
)

// SimpleTypeExpr takes a type expression and returns that type formatted as a
// string iff the type is simple (i.e. not a map, slice, function value,
// channel etc.), with any generic type constraints removed.
//
// Otherwise, returns ("", false).
//
// This is used to find the first FunctionSignature argument (or receiver) that
// matches a given type.
func SimpleTypeExpr(x ast.Expr) (string, bool) {
    var buf bytes.Buffer
    ok := writeSimpleTypeExpr(&buf, x)
    if !ok { return "", false }
    s := buf.String()
    // trim type constraints to ignore them
    idx := strings.IndexByte(s, '[')
    if idx > 0 {
        s = s[0:idx]
    }
    return s, true
}

// MatchSimpleType returns true if `Type` is a match for the simple
// type `matches`, which is a "simple" type (i.e. not a map, channel, slice,
// etc.). All type constraints on the field are ignored, and the type is still
// a match if the `Type` type is a pointer version of `matches`.
func MatchSimpleType(Type, matches string) bool {
    x, err := parser.ParseExpr(Type)
    if err != nil {
        return false
    }
    s, ok := SimpleTypeExpr(x)
    if !ok {
        return false
    }
    return (s == matches) || (s == "*"+matches)
}

// writeSimpleTypeExpr is a shortened version of [types.ExprString] used by
// [SimpleTypeExpr] to format a type as a string, excluding several features
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

// ParseTypeList parses a comma-separated list of types, including
// parenthesised sublists of types, calling visit once for each type or
// sublist.
//
// Sublists tuples are not recursively passed by this function but are simply
// indicated by calling visit on the entire sublist with "more" as true when
// calling the visit function.
//
// For example:
//
//     ParseTypeList(0, "a, (b, (c, d)), x, y, func (e, f)", visit)
//
// Calls visit with these arguments:
//
//     visit("a", false)
//     visit("b, (c, d)", true)
//     visit("x", false)
//     visit("y", false)
//     visit("func (e, f)", false)
//
// Returns false on parse error such as unpaired parentheses.
func ParseTypeList(types string, visit func(x string, more bool) bool) bool {
    types += "," // simplify end of string handling
    bracketDepth := 0
    token := make([]rune, 0)
    ok := true

    for _, c := range types {
        // skip leading space
        if (len(token) == 0) && runeIsHSpace(c) {
            continue
        }
        token = append(token, c)

        if c == '(' {
            bracketDepth++
        } else if c == ')' {
            bracketDepth--
            if bracketDepth < 0 {
                return false
            }
        } else if (c == ',') && (bracketDepth == 0) {
            if len(token) == 0 { return false }
            x := strings.TrimSpace(string(token[0:len(token)-1]))
            if len(x) == 0 { return false }

            if (x[0] == '(') && (x[len(x)-1] == ')') {
                ok = ok && visit(strings.TrimSpace(x[1:len(x)-1]), true)
            } else {
                ok = ok && visit(x, false)
            }
            token = token[0:0]
        }
    }
    return ok && (bracketDepth == 0)
}

// ParseTypeListRecursive parses a comma-separated list of types, including
// parenthesised sublists of types, calling visit once for each type.
//
// Parenthesised sublists are recursively parsed by this function, with
// the sublist nesting depth indicated by each call to visit.
//
// For example:
//
//     ParseTypeList(0, "a, (b, (c, d)), func (e, f)", visit)
//
// Calls visit with these arguments:
//
//     visit(0, "a")
//     visit(1, "b")
//     visit(2, "c")
//     visit(2, "d")
//     visit(0, "func (e, f)")
//
// Returns false on parse error such as unpaired brackets.
func ParseTypeListRecursive(types string, visit func(depth int, x string) bool) bool {
    ok := true

    visit_flat := func(x string, more bool) bool {
        if more {
            visit2 := func(depth int, x string) bool {
                ok = ok && visit(depth + 1, x)
                return ok
            }
            ok = ok && ParseTypeListRecursive(x, visit2)
        } else {
            ok = ok && visit(0, x)
        }
        return ok
    }

    return ok && ParseTypeList(types, visit_flat)
}

// SplitTypeTuple parses a comma-separated list of types, which must not
// contain parenthesised sublists, and returns each token as a string.
func SplitTypeTuple(types string) ([]string, bool) {
    var results []string
    ok := ParseTypeList(types, func(x string, more bool) bool {
        results = append(results, x)
        return more == false
    })
    if len(results) == 0 { return nil, false }
    return results, ok
}
