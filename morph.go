// Package morph is a Go code generator that generates code to map between
// structs and manipulate the form of functions.
//
// All types should be considered read-only.
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
    "fmt"
    "go/parser"
    "strings"
)

func must[T any](result T, err error) T {
    if err == nil { return result } else { panic(err) }
}

// FunctionSignature represents a parsed function signature, including any
// arguments, return types, method receiver, generic type constraints, etc.
type FunctionSignature struct {
    Comment   string
    Name      string
    Type      []Field
    Arguments []Field
    Returns   []Field
    Receiver  Field
}

// Function contains a parsed function signature and the raw source code of
// its body, excluding the enclosing "{" and "}" braces.
type Function struct {
    Signature FunctionSignature
    Body string
}

func (fn Function) String() string {
    return _function_string(fn)
}

func (fn FunctionSignature) String() string {
    return _functionSignature_string(fn)
}

// FunctionWrapper represents a constructed wrapper around a user-supplied
// inner function.
//
// If Inner is nil, the Current FunctionWrapper represents the original,
// user-supplied, function to be called at the inner-most level. the Inputs
// and Outputs are nil.
//
// Otherwise, the FunctionWrapper wraps another function, supplying Inputs that
// appear as source code as arguments to that function, and rewriting its
// result as source code using Outputs.
//
// For example, the function `Divide(a float64, b float64) (float64, error)`,
// which returns an error if dividing by zero, is represented as:
//
//     divideSig := FunctionSignature{
//         Name: "Divide",
//         Arguments: []Field{
//             {Name: "x", Type: "float64"},
//             {Name: "y", Type: "float64"},
//         },
//         Returns: []Field{
//             {Type: "float64"},
//             {Type: "error"},
//         },
//     }
//    divide := FunctionWrapper{
//        Current: divideSig,
//    }
//
// And a derived function `Halver(x float64) float64`, which returns `x / 2`,
// might be represented as:
//
//     halverSig := FunctionSignature{
//         Name: "Halver",
//         Arguments: []Field{
//             {Name: "x", Type: "float64"},
//         },
//         Returns: []Field{
//             {Type: "float64"},
//         },
//     }
//     halver := FunctionWrapper{
//         Inner: &inner,
//         Current: halverSig,
//         Inputs: []string{"x", "2"},
//         Outputs: func(results ... string) []string {
//             // divide by 2 won't panic, so we can drop results[1]
//             return []string{
//                 results[0],
//             }
//         }
//     }
//
// You won't usually have to instantiate these function signatures or wrapper
// structs directly, as they are usually constructed for you e.g. by
// [ParseFunctionSignature], [ParseFunction], the [FunctionSignature.Wrap]
// method, or by the FunctionWrapper methods. However, it can be useful to know
// about the internal structure if you want to create your own custom wrappers.
type FunctionWrapper struct {
    Inner *FunctionWrapper
    Current FunctionSignature
    Inputs []string
    Outputs func(results ... string) []string
}

// Field represents a name and type, such as a field in a struct or a type
// constraint, or a function argument. In a struct, a field may also contain a
// field struct tag or comments and may contain a value e.g. for initialising
// that field on a new struct value.
//
// Note that unlike the Go parser, the Tag, if any, does not include
// surrounding quote marks, and the Comment, if any, does not have a trailing
// new line.
type Field struct {
    Name    string
    Type    string
    Value   string
    Tag     string
    Comment string
}

// Struct represents a Go struct - it's type name, type constraints (if using
// generics), and fields.
//
// If the struct is used as the input to a function, ArgName represents the
// name of the argument of the struct type.
type Struct struct {
    Comment    string
    Name       string
    ArgName    string
    TypeParams []Field
    Fields     []Field
}

// String returns a source code representation of the given struct.
func (s Struct) String() string {
    return _struct_string(s)
}

// Copy returns a (deep) copy of a Struct, ensuring that slices aren't aliased.
func (s Struct) Copy() Struct {
    return Struct{
        Comment:    s.Comment,
        Name:       s.Name,
        ArgName:    s.ArgName,
        TypeParams: append([]Field{}, s.TypeParams...),
        Fields:     append([]Field{}, s.Fields...),
    }
}

// Mapper maps fields on a struct to fields on another struct.
//
// A Mapper is called once for each field defined on the input struct. Each
// invocation of the emit callback function generates a field on the output
// struct.
//
// It is permitted to call emit zero, one, or more than one time to produce
// zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name, and in the value argument with
// the qualified source field name.
//
// For example, for an input `Struct{Name: "Foo"}`, an input `Field{Name:
// "Bar", Value: "123"}`, then `emit(Field{Name: "Double$", Value: "2 * $"})`,
// for a function with signature `ConvertFoo(input Foo) Something`, generates a
// field `DoubleBar` with a value `2 * foo.Bar`.
type Mapper func(source Struct, input Field, emit func(output Field))

// Struct generates Go source code for a new struct type definition based on a
// source struct type definition and a [Mapper].
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
// In this function, the Value field is ignored on the output fields emitted by
// the mapper.
func (source Struct) Struct(
    signature string,
    mapper Mapper,
) (Struct, error) {
    return _struct_struct(source, signature, mapper)
}

// Function generates Go source code for a function that converts a value of a
// source struct type to a value of another struct type, based on the source
// struct type defintion and a [Mapper].
//
// The function is generated to match the provided signature, which must
// describe a function with at least one method receiver or named argument
// matching the source struct type (or a pointer to a struct of that type) (if
// there are several, the first such occurrence is selected), and exactly one
// return argument, which must be the type of the generated struct (or a
// pointer to a struct of that type). Omit the leading "func" keyword.
//
// For example, the signature argument may look something like:
//
//     (s *Store) AppleToOrange(ctx context.Context, a Apple) Orange
//
// In this function, the Type, Tag and Comment fields are ignored on the output
// fields emitted by the mapper.
func (source Struct) Function(
    signature string,
    mapper Mapper,
) (Function, error) {
    return _struct_function(source, signature, mapper)
}

// rewrite performs the special '$' replacement described by Mapper
func rewrite(source Struct, input Field, output Field) Field {
    return Field{
        Name:    strings.ReplaceAll(output.Name, "$", input.Name),
        Type:    strings.ReplaceAll(output.Type, "$", input.Type),
        Value:   strings.ReplaceAll(output.Value, "$", source.ArgName+"."+input.Name),
        Tag:     output.Tag,
        Comment: output.Comment,
    }
}

// collector returns an emit function that appends each emitted value to dest
func collector(dest *[]Field) func(output Field) {
    return func(output Field) {
        *dest = append(*dest, output)
    }
}

// _struct_struct implements the Struct.Struct method
func _struct_struct(source Struct, signature string, mapper Mapper) (result Struct, err error) {
    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error morphing struct %s to struct %q: %v", source.Name, signature, r)
        }
    }()

    src := `package temp; type ` + signature + ` struct {}`
    result = must(ParseStruct("temp.go", src, ""))
    var results []Field

    emit := collector(&results)
    for _, input := range source.Fields {
        emit2 := func(output Field) {
            emit(rewrite(source, input, output))
        }
        mapper(source, input, emit2)
    }

    result.Fields = results
    return result, nil
}

// _struct_function implements the Struct.Function method
func _struct_function(source Struct, signature string, mapper Mapper) (f Function, err error) {
    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error generating morph func %q for struct %q: %v", signature, source.Name, r)
        }
    }()

    fn := must(parseFunctionSignatureFromString(signature))

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
    inputArg := args[0] // first one wins

    // we also need to find the return argument
    returns, ok := fn.singleReturn()
    if !ok {
        panic(fmt.Errorf("FunctionSignature must have single return value"))
    }

    var assignments []Field

    emit := collector(&assignments)
    source2 := source // shallow copy is fine
    source2.ArgName = inputArg.Name
    for _, input := range source.Fields {
        emit2 := func(output Field) {
            emit(rewrite(source2, input, output))
        }
        mapper(source2, input, emit2)
    }

    return Function{
        Signature: fn,
        Body:      _struct_function_Body(returns.Type, assignments),
    }, nil
}
