package morph

import (
    "fmt"
    "go/ast"
    "go/parser"
    "go/token"
    "go/types"
    "strconv"
    "strings"

    "github.com/tawesoft/morph/internal"
)

// ParseStruct parses a given source file, looking for a struct type definition
// that defines a struct type with the given name.
//
// If name == "", ParseStruct returns the first struct found.
//
// If src != nil, ParseStruct parses the source from src and the filename is
// only used when recording position information. The type of the argument for
// the src parameter must be string, []byte, or io.Reader. If src == nil,
// instead parses the file specified by filename. This matches the behavior of
// [go/parser.ParseFile].
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
//
// ParseStruct only looks for struct type definitions in the top-level scope.
// This means that type definitions inside functions, etc. will be ignored.
func ParseStruct(filename string, src any, name string) (result Struct, err error) {
    esc := func(err error) (Struct, error) {
        return Struct{}, fmt.Errorf("error parsing %q for struct %q: %w", filename, name, err)
    }

    found := false
    pflags := parser.DeclarationErrors | parser.SkipObjectResolution | parser.ParseComments
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, filename, src, pflags)
    if err != nil {
        return esc(err)
    }

    ast.Inspect(astf, func(n ast.Node) bool {
        if found {
            return false
        }
        switch x := n.(type) {
        case *ast.GenDecl:
            if (x.Tok != token.TYPE) || (len(x.Specs) != 1) {
                return false
            }
            typeSpec := x.Specs[0].(*ast.TypeSpec)
            structType, ok := typeSpec.Type.(*ast.StructType)
            if !ok {
                return false
            }

            var structName string
            if structName = typeSpec.Name.String(); (name != "") && (name != structName) {
                return false
            }

            result = Struct{
                Name:       structName,
                Comment:    astText(x.Doc),
                TypeParams: fields(typeSpec.TypeParams),
                Fields:     fields(structType.Fields),
            }
            found = true

            return false
        case *ast.FuncDecl:
            // we want globally-scoped structs only
            return false
        }
        return true
    })

    if !found {
        return esc(fmt.Errorf("not found"))
    }
    return result, nil
}

// ParseFunctionSignature parses a given source file, looking for a function
// with the given name, and recording its signature.
//
// ParseFunctionSignature does not look for any methods on a type. For this,
// use [ParseMethodSignature] instead.
//
// If src != nil, ParseFunctionSignature parses the source from src and the
// filename is only used when recording position information. The type of the
// argument for the src parameter must be string, []byte, or io.Reader. If src
// == nil, instead parses the file specified by filename. This matches the
// behavior of [go.Parser/ParseFile].
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
func ParseFunctionSignature(filename string, src any, name string) (result FunctionSignature, err error) {
    return parseFunctionSignature(filename, src, func(sig FunctionSignature) bool {
        return (name == sig.Name) && (sig.Receiver.Name == "")
    })
}

// ParseFirstFunctionSignature is like [ParseFunctionSignature], except it will
// return the first function found (including methods).
//
// If src != nil, ParseFirstFunctionSignature parses the source from src and
// the filename is only used when recording position information. The type of
// the argument for the src parameter must be string, []byte, or io.Reader. If
// src == nil, instead parses the file specified by filename. This matches the
// behavior of [go.Parser/ParseFile].
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
func ParseFirstFunctionSignature(filename string, src any) (result FunctionSignature, err error) {
    return parseFunctionSignature(filename, src, func(sig FunctionSignature) bool {
        return true
    })
}

// ParseMethodSignature is like [ParseFunctionSignature], except it will match
// functions that are methods on the given type.
//
// If src != nil, ParseFunction parses the source from src and the filename is
// only used when recording position information. The type of the argument for
// the src parameter must be string, []byte, or io.Reader. If src == nil,
// instead parses the file specified by filename. This matches the behavior of
// [go.Parser/ParseFile].
//
// For example, to look for a method signature such as `func (foo *Bar) Baz()`,
// i.e. method Baz on type Bar with a pointer receiver, then set the name
// argument to "Baz" and the type argument to "Bar" (it does not matter that
// foo is a pointer type). Generic type constraints are ignored.
//
// Parsing is performed without full object resolution. This means parsing will
// still succeed even on some files that may not actually compile.
func ParseMethodSignature(filename string, src any, Type string, name string) (result FunctionSignature, err error) {
    return parseFunctionSignature(filename, src, func(sig FunctionSignature) bool {
        return (name == sig.Name) && sig.Receiver.matchSimpleType(Type)
    })
}

func parseFunctionSignature(
    filename string,
    src any,
    filter func(sig FunctionSignature) bool,
) (result FunctionSignature, err error) {
    esc := func(err error) (FunctionSignature, error) {
        return FunctionSignature{}, fmt.Errorf(
            "error parsing %q for function: %w", filename, err,
        )
    }

    pflags := parser.DeclarationErrors | parser.SkipObjectResolution | parser.ParseComments
    fset := token.NewFileSet()
    astf, err := parser.ParseFile(fset, "temp.go", src, pflags)
    if err != nil {
        return esc(err)
    }

    found := false
    ast.Inspect(astf, func(n ast.Node) bool {
        if found || (n == nil) {
            return false
        }

        funcDecl, ok := n.(*ast.FuncDecl)
        if !ok {
            return true
        }

        sig := FunctionSignature{
            Name:      funcDecl.Name.String(),
            Comment:   astText(funcDecl.Doc),
            Type:      args(funcDecl.Type.TypeParams),
            Arguments: args(funcDecl.Type.Params),
            Returns:   args(funcDecl.Type.Results),
            Receiver:  internal.FirstOrDefault(args(funcDecl.Recv), Argument{}),
        }
        if filter(sig) {
            found = true
            result = sig
        }
        return false
    })

    if !found {
        return esc(fmt.Errorf("not found"))
    }
    return result, nil
}

// singleReturn returns the return type for a FunctionSignature and true when
// there is exactly one return value, or (_, false) otherwise.
func (fs FunctionSignature) singleReturn() (Argument, bool) {
    if len(fs.Returns) != 1 {
        return Argument{}, false
    }
    return fs.Returns[0], true
}

// parseFunctionSignatureFromString parses the source code of a single function
// signature, such as `Foo(a A) B`.
//
// Parsing is performed without full object resolution.
func parseFunctionSignatureFromString(signature string) (result FunctionSignature, err error) {
    // ParseExpr doesn't work because we can't make a named function an expression,
    // so we create a whole dummy AST for a file.
    src := `package temp; func ` + signature + ` {}`
    return ParseFirstFunctionSignature("", src)
}

// astText returns the result of calling the Text() method on anything with
// that interface, or the empty string if the input is nil. Also trims spaces.
func astText(x interface{ Text() string }) string {
    if x == nil { return "" } else { return strings.TrimSpace(x.Text()) }
}

// fields converts an ast.FieldList into []Field. Returns nil for a nil input.
//
// A field with a type but no name is treated as a struct's embedded type with
// its name inherited from the type name.
func fields(fieldList *ast.FieldList) []Field {
    return astFieldListToFields(fieldList, true)
}

// args converts an ast.FieldList into []Argument. Returns nil for a nil input.
func args(fieldList *ast.FieldList) []Argument {
    fs := astFieldListToFields(fieldList, false)
    return internal.Map(fieldToArgument, fs)
}

func astFieldListToFields(fieldList *ast.FieldList, allowEmbedded bool) []Field {
    if fieldList == nil {
        return nil
    }
    result := []Field{}
    for _, field := range fieldList.List {
        fieldType := types.ExprString(field.Type)
        var tag string
        if field.Tag != nil {
            tag = internal.Must(strconv.Unquote(field.Tag.Value))
        }
        comment := astText(field.Doc)
        if comment == "" { comment = astText(field.Comment) }

        for _, fieldName := range field.Names {
            result = append(result, Field{
                Name: fieldName.String(),
                Type: fieldType,
                Tag: tag,
                Comment: comment,
            })
        }
        if len(field.Names) == 0 {
            var name = ""
            // e.g. embedded field Foo in struct Bar:
            //     type Foo struct { ... }
            //     type Bar struct { Foo }
            // This is treated as a field with name Foo.
            if allowEmbedded {
                name = fieldType
            }
            result = append(result, Field{
                Name: name,
                Type: fieldType,
                Tag: tag,
                Comment: comment,
            })
        }
    }
    if len(result) == 0 { return nil }
    return result
}
