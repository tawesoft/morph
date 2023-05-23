// Package morph is a Go code generator that generates code to map between
// structs and manipulate the form of functions.
//
// All types should be considered read-only & immutable except where otherwise
// specified.
//
// Need help? Ask on morph's GitHub issue tracker or check out the tutorials
// on the morph GitHub repo. Also, paid commercial support and training is
// available via [Open Source at Tawesoft].
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

    "github.com/tawesoft/morph/internal"
)

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

// String formats the function signature as a string. It does not include
// the leading "func" keyword.
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
// [ParseFunctionSignature], the Parse...Function functions, the
// [FunctionSignature.Wrap] method, or by methods on a FunctionWrapper.
// However, it can be useful to know about the internal structure if you want
// to create your own custom wrappers.
type FunctionWrapper struct {
    Inner *FunctionWrapper
    Current FunctionSignature
    Inputs []string
    Outputs func(results ... string) []string
}

// Field represents a name and type, such as a field in a struct or a type
// constraint, or a function argument. In a struct, a field may also contain a
// field struct tag, comments, a value (e.g. for initialising
// that field on a new struct value), a comparer expression (e.g. for
// comparing two fields of the same type), a copier expression (e.g. for
// performing a deep copy), and a reverse function that performs the opposite
// of a mapping, if possible.
//
// An empty string for the value field means the zero value. An empty string
// for the comparer field means compare with "==". An empty string for the
// copier field means copy with "=". Reverse may be nil.
//
// The Tag, if any, does not include surrounding quote marks, and the Comment,
// if any, does not have a trailing new line or comment characters such as
// a starting "/*", an ending "*/", or "//" at the start of each line.
//
// Fields are always passed by value, so it is safe to mutate an input field
// argument anywhere.
type Field struct {
    Name     string
    Type     string
    Value    string
    Tag      string
    Comment  string
    Comparer string
    Reverse  FieldMapper
}

// AppendTags returns a new Field with the given tags appended to the field's
// existing tags (if any), joined with a space separator, as is the convention.
//
// Note that this does not modify the field the method is called on.
//
// Each tag in the tags list to be appended should be a single key:value pair.
//
// If a tag in the tags list to be appended is already present in the original
// struct tag string, it is not appended.
//
// If any tags do not have the conventional format, the value returned
// is unspecified.
func (f Field) AppendTags(tags ... string) Field {
    f.Tag = internal.AppendTags(f.Tag, tags...)
    return f
}

// AppendComments returns a new Field with the comments appended to the field's
// existing comment string (if any), joined with a newline separator.
//
// Note that this does not modify the field the method is called on.
func (f Field) AppendComments(comments ... string) Field {
    f.Comment = internal.AppendComments(f.Comment, comments...)
    return f
}

// FieldMapper maps fields on a struct to fields on another struct.
//
// A FieldMapper is called once for each field defined on an input struct.
// Each invocation of the emit callback function generates a field on the
// output struct.
//
// It is permitted to call the emit function zero, one, or more than one time
// to produce zero, one, or more fields from a single input field.
//
// As a special case, when emit is invoked, the character "$" is replaced in
// the name argument with the source field name, "$" is replaced in the type
// argument with the source field type, ".$" in the value argument with the
// source field name. Later, when generating a function, "$." is replaced in
// the value argument with the name of an input argument used to refer to a
// value of the struct type. In that case, use "$.$" for a fully qualified
// source field name.
//
// For example, for an input `Struct{Name: "Foo"}`, an input `Field{Name:
// "Bar", Value: "123"}`, then `emit(Field{Name: "Double$", Value: "2 * $.$"})`,
// for a function with signature `ConvertFoo(input Foo) Something`, generates a
// field `DoubleBar` with a value `2 * input.Bar`.
type FieldMapper func(input Field, emit func(output Field))

// StructMapper returns a new StructMapper that applies the given
// FieldMapper to every field on the input struct.
func (mapper FieldMapper) StructMapper() StructMapper {
    if mapper == nil { return nil }
    return func(in Struct) Struct {
        var results []Field

        emit := collector(&results)
        for _, input := range in.Fields {
            emit2 := func(output Field) {
                emit(output.Rewrite(input))
            }
            mapper(input, emit2)
        }

        in.Fields = results
        return in
    }
}

// Struct represents a Go struct - it's type name, type constraints (if using
// generics), doc comment, and fields.
//
// If the struct has been renamed, From is its previous name. Otherwise, it is
// the empty string.
type Struct struct {
    Comment    string
    Name       string
    From       string
    TypeParams []Field
    Fields     []Field
}

// Signature returns the type signature of a struct as a string, including any
// generic type constraints, omitting the "type" and "struct" keywords.
func (s Struct) Signature() string {
    return _struct_signature(s)
}

// String returns a source code representation of the given struct.
func (s Struct) String() string {
    return _struct_string(s)
}

// Copy returns a (deep) copy of a Struct, ensuring that slices aren't aliased.
func (s Struct) Copy() Struct {
    ss := Struct{
        Comment:    s.Comment,
        Name:       s.Name,
        From:       s.From,
        TypeParams: append([]Field{}, s.TypeParams...),
        Fields:     append([]Field{}, s.Fields...),
    }
    return ss
}

// Map applies each given [StructMapper] (in order of the arguments provided)
// to a struct and returns the result.
func (s Struct) Map(mappers ... StructMapper) Struct {
    ss := s
    if len(mappers) > 0 {
        ss = ss.Copy()
    }
    for _, t := range mappers {
        ss = t(ss)
    }
    return ss
}

// MapFields applies each given [FieldMapper] (in order of the arguments
// provided) to a struct and returns the result.
func (s Struct) MapFields(mappers ... FieldMapper) Struct {
    return s.Map(internal.Map(FieldMapper.StructMapper, mappers)...)
}

// StructMapper maps a Struct to another Struct.
type StructMapper func(in Struct) Struct

// Converter generates Go source code for a function that converts a value of
// the given struct type from a previous struct type.
//
// The function is generated to match the provided signature, which must
// describe a function with at least one method receiver or named argument
// matching the previous struct type (or a pointer to a struct of that type)
// (if there are several, the first such occurrence is selected), and exactly
// one return argument, which must be the type of the receiver struct (or a
// pointer to a struct of that type). Omit the leading "func" keyword.
//
// In the signature, the following tokens are rewritten:
//
//   - $From: the previous struct type name.
//   - $from: the previous struct type name, first letter in lowercase.
//   - $To: the current struct type name.
//
// If a struct has not been renamed, then the previous struct type name is the
// current struct type name.
//
// For example, the signature argument may look something like:
//
//     $FromTo$To(ctx context.Context, $from $From) $To
//     (s *$From) To$To($from *$From) *$To
//
//  Or simply, explicitly:
//
//     AppleToOrange(apple Apple) Orange
//
// In this function, the Type, Tag and Comment fields are ignored on the
// struct fields.
//
// In every field value, the token "$." is rewritten as the input argument
// name plus ".".
//
// For example, the value "$.FieldOne" will get rewritten to something like
// "in.FieldOne".
func (s Struct) Converter(
    signature string,
) (Function, error) {
    return _struct_converter(s, signature)
}

// rewriteSignature performs the special '$' replacement in a signature
func rewriteSignature(sig string, from string, to string) string {
    lower := func(x string) string {
        if len(x) == 0 { return x }
        if len(x) == 1 { strings.ToLower(x) }
        return strings.ToLower(string(x[0])) + x[1:]
    }
    sig = strings.ReplaceAll(sig, "$From", from)
    sig = strings.ReplaceAll(sig, "$To", to)
    sig = strings.ReplaceAll(sig, "$from", lower(from))
    return sig
}

// Rewrite performs the special '$' replacement described by FieldMapper.
//
// Note that a remaining "$." in a field value remains present and is not
// rewritten until generating a function, as it requires the name of a function
// argument to be known.
//
// TODO ignore $ inside string or rune literals
func (f Field) Rewrite(input Field) Field {
    return Field{
        Name:     strings.ReplaceAll(f.Name, "$", input.Name),
        Type:     strings.ReplaceAll(f.Type, "$", input.Type),
        Value:    strings.ReplaceAll(f.Value, ".$", "."+input.Name),
        Tag:      f.Tag,
        Comment:  f.Comment,
        Comparer: f.Comparer,
        Reverse:  f.Reverse,
    }
}

// postRewriteField performs the special '$' replacement described by StructMapper
//
// TODO ignore $ inside string or rune literals
func postRewriteField(argName string, output Field) Field {
    output.Value = strings.ReplaceAll(output.Value, "$.", argName + ".")
    return output
}

// collector returns an emit function that appends each emitted value to dest
func collector(dest *[]Field) func(output Field) {
    return func(output Field) {
        *dest = append(*dest, output)
    }
}

/*
// _struct implements the [Struct.Struct] method.
func (source Struct) _struct(signature string, mapper FieldMapper) (result Struct, err error) {

    signature = rewriteSignature(signature, source.Name)

    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error morphing struct %s to struct %q: %v", source.Name, signature, r)
        }
    }()

    src := `package temp; type ` + signature + ` struct {}`
    result = internal.Must(ParseStruct("temp.go", src, ""))
}
*/

// _struct_function implements the Struct.Converter method
func _struct_converter(source Struct, signature string) (f Function, err error) {
    if source.From == "" { source.From = source.Name }
    signature = rewriteSignature(signature, source.From, source.Name)

    // allow user-defined morpher to panic
    defer func() {
        if r := recover(); r != nil {
            err = fmt.Errorf("error generating morph func %q for struct %q: %v", signature, source.Name, r)
        }
    }()

    fn := internal.Must(parseFunctionSignatureFromString(signature))

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
        return (s == source.From) || (s == "*"+source.From)
    })
    if len(args) < 1 {
        panic(fmt.Errorf("could not find matching argument for source %q", source.Name))
    }
    inputArg := args[0] // first one wins

    // we also need to find the return argument
    returns, ok := fn.singleReturn()
    if !ok {
        panic(fmt.Errorf("FunctionSignature must have single return value"))
    }

    var assignments = make([]Field, len(source.Fields))

    for i, field := range source.Fields {
        assignments[i] = postRewriteField(inputArg.Name, field)
    }

    /*
    emit := collector(&assignments)
    for _, input := range source.Fields {
        emit2 := func(output Field) {
            emit(RewriteField(input, output))
        }
        mapper(input, emit2)
    }
    for i := 0; i < len(assignments); i++ {
        assignments[i] = postRewriteField(inputArg.Name, assignments[i])
    }
     */

    return Function{
        Signature: fn,
        Body:      _struct_function_Body(returns.Type, assignments),
    }, nil
}
