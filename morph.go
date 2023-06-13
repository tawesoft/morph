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

// Copy returns a (deep) copy of a FunctionSignature, ensuring that slices
// aren't aliased.
func (fs FunctionSignature) Copy() FunctionSignature {
    var args, returns, types []Field

    if len(fs.Type) > 0 {
        types = append([]Field{}, fs.Type...)
    }
    if len(fs.Arguments) > 0 {
        args = append([]Field{}, fs.Arguments...)
    }
    if len(fs.Returns) > 0 {
        returns = append([]Field{}, fs.Returns...)
    }

    return FunctionSignature{
        Comment:   fs.Comment,
        Name:      fs.Name,
        Type:      types,
        Arguments: args,
        Returns:   returns,
        Receiver:  fs.Receiver,
    }
}

// matchSimpleType returns true if a field's type is a match for the simple
// type specified, which is a "simple" type (i.e. not a map, channel, slice,
// etc.). All type constraints on the field are ignored, and the type is still
// a match if the field's type is a pointer of the specified type.
func (f Field) matchSimpleType(Type string) bool {
    x, err := parser.ParseExpr(f.Type)
    if err != nil {
        return false
    }
    s, ok := simpleTypeExpr(x)
    if !ok {
        return false
    }
    return (s == Type) || (s == "*"+Type)
}

// Inputs returns a slice containing a Field for each input specified by the
// function signature, including the method reciever (if any) as the first
// argument).
func (fs FunctionSignature) Inputs() []Field {
    var args []Field
    if fs.Receiver.Type != "" {
        args = append(args, fs.Receiver)
    }
    args = append(args, fs.Arguments...)
    return args
}


// fieldTypeFilterer returns a function that returns true for any Field whose
// type is a simple type that matches the provided Type.
func fieldTypeFilterer(Type string) func (f Field) bool {
    return func(f Field) bool {
        return f.matchSimpleType(Type)
    }
}

// matchingInput searches the function receiver and input arguments, in order,
// for the first instance of an input matching the provided type, or a pointer
// of the provided type. Type constraints are ignored.
func (fs FunctionSignature) matchingInput(Type string) (match Field, found bool) {
    args := filterFields(fs.Inputs(), fieldTypeFilterer(Type))
    if len(args) < 1 {
        return Field{}, false
    }
    return args[0], true // first one wins
}

// matchingInputs searches the function receiver and input arguments, in order,
// for the first two instances of an input matching the provided type, or a
// pointer of the provided type. Type constraints are ignored.
func (fs FunctionSignature) matchingInputs(Type string) (match1 Field, match2 Field, found bool) {
    args := filterFields(fs.Inputs(), fieldTypeFilterer(Type))
    if len(args) < 2 {
        return Field{}, Field{}, false
    }
    return args[0], args[1], true // first two win
}

// Function contains a parsed function signature and the raw source code of
// its body, excluding the enclosing "{" and "}" braces.
type Function struct {
    Signature FunctionSignature
    Body string
}

// Field represents a name and type, such as a field in a struct or a type
// constraint, or a function argument.
//
// In a struct, a field may also contain a
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
//
// When creating a Reverse FieldMapper, care should be taken to not overwrite
// a field's existing Reverse. This can be achieved by composing the two
// functions with the Compose function in the fieldmappers sub package, e.g.
// `NewField.Reverse = fieldmappers.Compose(newfunc, OldField.Reverse))`.
type Field struct {
    Name     string
    Type     string
    Value    string
    Tag      string
    Comment  string
    Reverse  FieldMapper

    Comparer string // equality
    Orderer string  // less than
    Copier string   // assignment
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

// Rewrite performs the special '$' replacement described by FieldMapper.
//
// Note that a remaining "$." in a field value remains present and is not
// rewritten until generating a function, as it requires the name of a function
// argument to be known.
//
// TODO ignore $ inside string or rune literals
func (f Field) Rewrite(input Field) Field {
    f.Name  = strings.ReplaceAll(f.Name, "$", input.Name)
    f.Type  = strings.ReplaceAll(f.Type, "$", input.Type)
    f.Value = strings.ReplaceAll(f.Value, ".$", "."+input.Name)
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

// StructMapper returns a new StructMapper that applies the given FieldMapper
// to every field on the input struct.
//
// If the FieldMapper is reversible, then so is the returned StructMapper.
func (mapper FieldMapper) StructMapper() StructMapper {
    if mapper == nil { return nil }
    return func(in Struct) Struct {
        var results []Field
        out := in.Copy()

        emit := collector(&results)
        for _, input := range out.Fields {
            emit2 := func(output Field) {
                emit(output.Rewrite(input))
            }
            mapper(input, emit2)
        }

        out.Fields = results
        oldReverse := out.Reverse
        out.Reverse = func(in2 Struct) Struct {
            out2 := in2.MapFields(func (input2 Field, emit2 func(output Field)) {
                if input2.Reverse != nil {
                    input2.Reverse(input2, emit2)
                } else {
                    emit2(input2)
                }
            })
            if oldReverse != nil {
                out2 = oldReverse(out2.Copy())
            }
            return out2
        }
        return out
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
    Reverse StructMapper
}

// Copy returns a (deep) copy of a Struct, ensuring that slices aren't aliased.
func (s Struct) Copy() Struct {
    var typeParams, fields []Field

    if len(s.TypeParams) > 0 {
        typeParams = append([]Field{}, s.TypeParams...)
    }
    if len(s.Fields) > 0 {
        fields = append([]Field{}, s.Fields...)
    }

    ss := Struct{
        Comment:    s.Comment,
        Name:       s.Name,
        From:       s.From,
        TypeParams: typeParams,
        Fields:     fields,
        Reverse:    s.Reverse,
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
        if t == nil { continue }
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
// one return argument of the same type (or a pointer to a struct of that
// type). Omit the leading "func" keyword.
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
// name plus ".", the token ".$" is rewritten to the field name prefixed by ".",
// and the token "$.$" is rewritten as the input argument name plus "." plus
// the field name.
//
// For example, the value "$.FieldOne" will get rewritten to something like
// "in.FieldOne".
func (s Struct) Converter(
    signature string,
) (Function, error) {
    if s.From == "" { s.From = s.Name } // shallow copy is fine here
    signature = rewriteSignatureString(signature, s.From, s.Name)

    esc := func(err error) (Function, error) {
        return Function{}, fmt.Errorf(
            "error creating morph.Struct.Converter function for type %q -> %q: %w",
            s.From, s.Name, err,
        )
    }

    fs, err := parseFunctionSignatureFromString(signature)
    if err != nil {
        return esc(fmt.Errorf("error parsing function signature: %w", err))
    }

    inputArg, ok := fs.matchingInput(s.From)
    if !ok {
        return esc(fmt.Errorf(
            "function signature %q must include an input of type %q or *%q",
            signature, s.From, s.From,
        ))
    }

    returns, ok := fs.singleReturn()
    if (!ok) || ((returns.Type != s.Name) && (returns.Type != "*"+s.Name)) {
        return esc(fmt.Errorf(
            "function signature %q must have a single return value of type %q or *%q; got %q",
            signature, s.Name, s.Name, returns.Type,
        ))
    }

    assignments := internal.Map(func(f Field) Field {
        return postRewriteField(inputArg.Name, f)
    }, s.Fields)

    body := formatStructConverter(returns.Type, assignments)

    fs.Comment = fmt.Sprintf("%s converts [%s] to [%s].", fs.Name, s.From, s.Name)
    return Function{
        Signature: fs,
        Body:      body,
    }, nil
}

// Comparer generates Go source code for a function that compares if two
// struct values are equal.
//
// The function is generated to match the provided signature, which must
// describe a function with at least two method receivers or named arguments
// matching the struct type (or a pointer to a struct of that type) (if there
// are several, the first two such occurrences are selected, in order). Omit
// the leading "func" keyword. Omit the boolean return value.
//
// In the signature, the following tokens are rewritten:
//
//   - $: the struct type name.
//
// For example, the signature argument may look something like:
//
//     $Equals(first $, second $)
//     (source *$) Equals(target *$)
//
//  Or simply, explicitly, something like:
//
//     Equals(first Thing, second Thing)
//
// In this function, the Type, Tag and Comment fields are ignored on the
// struct fields.
//
// In every field comparer, the token "$c." is rewritten as the input argument
// name plus "." for c == "a" as the first input and c == "b" as the second
// input. The token ".$" is rewritten to the field name prefixed by ".",
// and the token "$c.$" is rewritten as the input argument name plus "." plus
// the field name, for (again) c == "a" or "b".
//
// For example, the comparer "$a.$ == $b.X + $b.Y" will get rewritten to
// something like "first.Foo == second.X + second.Y".
func (s Struct) Comparer(
    signature string,
) (Function, error) {
    signature = strings.ReplaceAll(signature, "$", s.Name)

    esc := func(err error) (Function, error) {
        return Function{}, fmt.Errorf(
            "error creating morph.Struct.Comparer function for type %q -> %q: %w",
            s.From, s.Name, err,
        )
    }

    fs, err := parseFunctionSignatureFromString(signature)
    if err != nil {
        return esc(fmt.Errorf("error parsing function signature: %w", err))
    }

    arg1, arg2, ok := fs.matchingInputs(s.Name)
    if !ok {
        return esc(fmt.Errorf(
            "function signature %q must include two inputs of type %q or *%q",
            signature, s.Name, s.Name,
        ))
    }

    if len(fs.Returns) > 0 {
        return esc(fmt.Errorf(
            "function signature %q must not have return values specified",
            signature,
        ))
    }
    fs.Returns = []Field{
        {Type: "bool",},
    }

    comparisons := internal.Map(func(f Field) Field {
        return rewriteComparer(arg1.Name, arg2.Name, f)
    }, s.Fields)

    body := formatStructComparer(arg1.Name, arg2.Name, comparisons)

    fs.Comment = fmt.Sprintf("%s returns true if two [%s] values are equal.", fs.Name, s.Name)
    return Function{
        Signature: fs,
        Body:      body,
    }, nil
}

// Copier generates Go source code for a function that copies a source struct
// value to a destination struct value,
//
// The function is generated to match the provided signature, which must
// describe a function with at least one method receiver or named argument
// matching the source struct type (or a pointer to a struct of that type)
// (if there are several, the first such occurrence is selected), and exactly
// one return argument of the same type (or a pointer to a struct of that
// type), which may be named or unnamed. Omit the leading "func" keyword.
//
// In the signature, the following tokens are rewritten:
//
//   - $: the struct type name.
//
// For example, the signature argument may look something like:
//
//     $Copy(from $) $
//     (from *$) Copy() $
//
//  Or simply, explicitly, something like:
//
//     ThingCopy(from Thing) (to Thing)
//
// In this function, the Type, Tag and Comment fields are ignored on the
// struct fields.
//
// In every field copier, the token "$target." is rewritten as the target
// argument name plus "." for target == "src" as the source input and target ==
// "dest" as the output. The token ".$" is rewritten to the field name prefixed
// by ".", and the token "$target.$" is rewritten as the target argument name
// plus "." plus the field name, for (again) target == "src" or "dest".
//
// For example, the copier "$dest.$ = $src.X + $src.Y" will get rewritten to
// something like "to.Foo == from.X + from.Y".
func (s Struct) Copier(
    signature string,
) (Function, error) {
    signature = strings.ReplaceAll(signature, "$", s.Name)

    esc := func(err error) (Function, error) {
        return Function{}, fmt.Errorf(
            "error creating morph.Struct.Copier function for type %q -> %q: %w",
            s.From, s.Name, err,
        )
    }

    fs, err := parseFunctionSignatureFromString(signature)
    if err != nil {
        return esc(fmt.Errorf("error parsing function signature: %w", err))
    }

    inputArg, ok := fs.matchingInput(s.Name)
    if !ok {
        return esc(fmt.Errorf(
            "function signature %q must include an inputs of type %q or *%q",
            signature, s.Name, s.Name,
        ))
    }

    if len(fs.Returns) != 1 {
        return esc(fmt.Errorf(
            "function signature %q must have exactly one return value",
            signature,
        ))
    }

    copies := internal.Map(func(f Field) Field {
        return rewriteCopier(inputArg.Name, "_out", f)
    }, s.Fields)

    body := formatStructCopier(inputArg.Name, "_out", fs.Returns[0].Type, copies)

    fs.Comment = fmt.Sprintf("%s returns a copy of the [%s] %s.", fs.Name, s.Name, inputArg.Name)
    return Function{
        Signature: fs,
        Body:      body,
    }, nil
}

// Orderer generates Go source code for a function that compares two struct
// values and returns true if the first is less than the second.
//
// The function is generated to match the provided signature, which must
// describe a function with at least two method receivers or named arguments
// matching the struct type (or a pointer to a struct of that type) (if there
// are several, the first two such occurrences are selected, in order). Omit
// the leading "func" keyword. Omit the boolean return value.
//
// In the signature, the following tokens are rewritten:
//
//   - $: the struct type name.
//
// For example, the signature argument may look something like:
//
//     $LessThan(first $, second $)
//     (source *$) LessThan(target *$)
//
//  Or simply, explicitly, something like:
//
//     ThingLessThan(first Thing, second Thing)
//
// In this function, the Type, Tag and Comment fields are ignored on the
// struct fields.
//
// In every field orderer, the token "$c." is rewritten as the input argument
// name plus "." for c == "a" as the first input and c == "b" as the second
// input. The token ".$" is rewritten to the field name prefixed by ".",
// and the token "$c.$" is rewritten as the input argument name plus "." plus
// the field name, for (again) c == "a" or "b".
//
// For example, the comparer "$a.$ < ($b.X + $b.Y)" will get rewritten to
// something like "first.Foo < (second.X + second.Y)".
func (s Struct) Orderer(
    signature string,
) (Function, error) {
    signature = strings.ReplaceAll(signature, "$", s.Name)

    esc := func(err error) (Function, error) {
        return Function{}, fmt.Errorf(
            "error creating morph.Struct.Orderer function for type %q -> %q: %w",
            s.From, s.Name, err,
        )
    }

    fs, err := parseFunctionSignatureFromString(signature)
    if err != nil {
        return esc(fmt.Errorf("error parsing function signature: %w", err))
    }

    arg1, arg2, ok := fs.matchingInputs(s.Name)
    if !ok {
        return esc(fmt.Errorf(
            "function signature %q must include two inputs of type %q or *%q",
            signature, s.Name, s.Name,
        ))
    }

    if len(fs.Returns) > 0 {
        return esc(fmt.Errorf(
            "function signature %q must not have return values specified",
            signature,
        ))
    }
    fs.Returns = []Field{
        {Type: "bool",},
    }

    comparisons := internal.Map(func(f Field) Field {
        return rewriteOrderer(arg1.Name, arg2.Name, f)
    }, s.Fields)

    body := formatStructOrderer(arg1.Name, arg2.Name, comparisons)

    fs.Comment = fmt.Sprintf("%s returns true if the first [%s] is less than the second.", fs.Name, s.Name)
    return Function{
        Signature: fs,
        Body:      body,
    }, nil
}

// rewriteSignatureString performs the special '$' replacement in a function
// signature specified as a string.
func rewriteSignatureString(sig string, from string, to string) string {
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

// postRewriteField performs the special '$.' replacement described by
// FieldMapper when the field appears in a function as a value.
//
// TODO ignore "$." inside string or rune literals
func postRewriteField(argName string, output Field) Field {
    output.Value = strings.ReplaceAll(output.Value, "$.", argName + ".")
    return output
}

// rewriteComparer performs the special '$.' replacement described by
// [Struct.Comparer].
//
// TODO ignore "$." inside string or rune literals
func rewriteComparer(argName1 string, argName2 string, output Field) Field {
    output.Comparer = strings.ReplaceAll(output.Comparer, ".$", "."+output.Name)
    output.Comparer = strings.ReplaceAll(output.Comparer, "$a.", argName1 + ".")
    output.Comparer = strings.ReplaceAll(output.Comparer, "$b.", argName2 + ".")
    return output
}

func rewriteOrderer(argName1 string, argName2 string, output Field) Field {
    output.Orderer = strings.ReplaceAll(output.Orderer, ".$", "."+output.Name)
    output.Orderer = strings.ReplaceAll(output.Orderer, "$a.", argName1 + ".")
    output.Orderer = strings.ReplaceAll(output.Orderer, "$b.", argName2 + ".")
    return output
}

// rewriteCopier performs the special '$.' replacement described by
// [Struct.Copier].
//
// TODO ignore "$." inside string or rune literals
func rewriteCopier(inputName string, outputName string, output Field) Field {
    output.Copier = strings.ReplaceAll(output.Copier, ".$", "."+output.Name)
    output.Copier = strings.ReplaceAll(output.Copier, "$src.", inputName + ".")
    output.Copier = strings.ReplaceAll(output.Copier, "$dest.", outputName + ".")
    return output
}

// collector returns an emit function that appends each emitted value to dest
func collector(dest *[]Field) func(output Field) {
    return func(output Field) {
        *dest = append(*dest, output)
    }
}

// WrappedFunction represents a constructed wrapper around a user-supplied
// inner function.
//
// If Inner is nil, the FunctionSignature stored at Current represents the
// original, user-supplied, function to be called at the inner-most level. the
// Inputs and Outputs are nil.
//
// Otherwise, Inner represents an inner WrappedFunction, supplying Inputs that
// appear as source code as arguments to that function, and rewriting its
// result as source code using Outputs. Higher-order functions are implemented
// through a chain of Inner values.
//
// The Inputs field represents inputs to the inner function. This string can
// contain an arbitrary Go expression to rewrite inputs and add or remove
// inputs e.g. "a * 2, math.Floor(b), 13". Variables take the names from the
// function signature argument names.
//
// The Outputs field represents collected outputs from a call to the inner
// function. This string can also contain an arbitrary Go expression to rewrite
// outputs and add or remove outputs e.g. "($0 * 2), $1 != nil". The token "$n"
// for n = '0' to '9' is replaced with the result at that position.
//
// TODO: allow substitution of named return values with $name tokens.
//
// For example, the function `Divide(a float64, b float64) (float64, error)`,
// which returns an error if dividing by zero, is represented as:
//
//     divideSig := Function{
//         Signature: FunctionSignature{
//             Name: "Divide",
//             Arguments: []Field{
//                 {Name: "x", Type: "float64"},
//                 {Name: "y", Type: "float64"},
//             },
//             Returns: []Field{
//                 {Type: "float64"},
//                 {Type: "error"},
//             },
//         },
//         Body: "...", // omitted for clarity
//     }
//    divide := WrappedFunction{
//        Current: divideSig,
//    }
//
// And a derived function `Halver(n float64) float64`, which returns `n / 2`,
// might be represented as:
//
//     halverSig := FunctionSignature{
//         Name: "Halver",
//         Arguments: []Field{
//             {Name: "n", Type: "float64"},
//         },
//         Returns: []Field{
//             {Type: "float64"},
//         },
//     }
//     halver := WrappedFunction{
//         Inner: &divide,
//         Current: halverSig,
//         // Inputs to the inner function (in this case "Divide(x, y)")
//         Inputs: "n, 2",
//         // divide by 2 won't panic, so we can drop the result at index 1 and
//         // emit only the result at index 0.
//         Outputs: "$0",
//     }
//
// You won't usually have to instantiate these function signatures or wrapper
// structs directly, as they are usually constructed for you e.g. by
// [ParseFunctionSignature], the Parse...Function functions, the
// [FunctionSignature.Wrap] method, or by methods on a FunctionWrapper.
// However, it can be useful to know about the internal structure if you want
// to create your own custom wrappers.

/*type WrappedFunction struct {
    Inner   *WrappedFunction
    Current Function
    Inputs  string
    Outputs string
}*/

type FunctionWrapper func(in WrappedFunction) (WrappedFunction, error)

type WrappedFunction struct {
    Signature FunctionSignature
    Inputs   ArgRewriter // Rewritten inputs to wrapped function
    Outputs  ArgRewriter // Rewritten outputs from wrapped function
    Wraps *WrappedFunction
}

// Wrap turns a function into a wrapped function, ready for further wrapping.
func (f Function) Wrap() WrappedFunction {
    return WrappedFunction{
        Signature: f.Signature,
        Wraps:     nil,
    }
}

// Wrap applies function wrappers to a given wrapped function.
//
// The earliest wrappers in the list are the first to be applied.
//
// For example, analogous to function composition,
// `x.Wrap(f, g)` applies `g(f(x))`.
//
// Returns the first error encountered.
func (f WrappedFunction) Wrap(wrappers ... FunctionWrapper) (WrappedFunction, error) {
    current := f
    var err error

    for _, wrapper := range wrappers {
        current, err = wrapper(current)
        if err != nil { return WrappedFunction{}, err }
    }

    return current, nil
}

// ArgRewriter either describes how an outer function rewrites its inputs
// before passing them to a wrapped inner function, or how an outer function
// rewrites the outputs of a wrapped inner function before returning them from
// the outer function.
//
// It does this through capturing arguments into temporary variables, and
// formatting a new list of arguments or return values.
//
// This is a low-level implementation for advanced use. The functions in
// morph/funcwrappers.go provide nicer APIs for many use cases.
//
// A WrappedFunction uses two ArgRewriters in the following sequence, in
// pseudocode:
//
//     func outerfunc(args) (returns) { // step 0, a call to outerfunc
//         inputs... = Input ArgRewriter captured args // step 1
//         results... = innerfunc(Input ArgWriter formats inputs) // step 2
//         outputs = Output ArgRewriter captured results // step 3
//         return Output ArgRewriter formats outputs // step 4
//     }
//
// At each step, an ArgRewriter can retrieve the results of a previous step
// (and only the immediately previous step) by "$" token notation, accessing a
// named argument by "$name", and an argument by index by "$n" for some decimal
// n. At each step, a WrappedFunction uses an ArgRewriter to transform the
// result of the previous values.
//
// In step 1, the results retrieved by "$" token notation are the input
// arguments to the outerfunc. In step 2, they are the captured arguments from
// step 1. In step 3, they are the return values from the innerfunc call. In
// step 4, they are the captured arguments from step 3.
//
// Capture is a list of Field elements which describe how to temporarily store
// inputs to or outputs of invocations of the inner function. Only the Name,
// Type, and Value fields on each capture are used, and each is optional.
//
// If a Name is given, a subsequent step can refer to the result by "$name",
// not just by index "$n".
//
// The Value of the capture is a code expression of how it stores an input or
// output. In most cases, this will be simply correspond to the value of a
// matching input or output, unchanged. Here, inputs and named outputs can be
// referred to using "$" token notation by name e.g. "$name", or by index e.g.
// "$0". A value need not refer to any previous step, e.g. can be a literal
// like "2", or some function call like "time.Now()". For a single-valued
// result that is used only once, it may be better to do this in Formatter.
//
// A capture with an empty string for a Value at position i in the capture
// slice captures the argument at position i in the previous step. Likewise,
// a capture with an empty Type at position i in the capture slice has the
// type of the argument at position i in the previous step.
//
// The capture arg's Type is a comma-separated list of types that the
// Value produces. If the type list has more than one entry, e.g. "int,
// error" then the captured value captures a tuple result type instead of a
// single value. Each field in the tuple can be accessed by index by adding
// ".N" for any decimal N to a "$" replacement token e.g. "$name.0" or "$0.3".
//
// All inputs and outputs must be accounted for. This can be achieved by
// capturing a value, or part of a tuple value, to a capture field with a type
// of "_". For a tuple value, each discarded element of the tuple must be
// explicitly discarded, for example with a capture type specified like
// "int, _, _".
//
// Formatter describes how the captured arguments or results are rewritten as the
// input arguments to the inner function, or as the outer function's return
// arguments.
//
// For example, given some function `Foo(x float64) float64`, you can imagine
// some outer wrapping function using an ArgRewriters that converts it into
// `func(x float64) (float64, float64) => math.Modf(Foo(x))`, with the
// following ArgRewriter values:
//
//     Input: ArgRewriter{
//         Capture: []Field{{Name: "x", Type: "float64", Value: "$x"},},
//         Formatter: "$0",
//     }
//
//     Output: ArgRewriter{
//         Capture: []Field{
//             {Type: "float64, float64", Value: "math.Modf($x)"},
//         },
//         Formatter: "$0.0, $0.1",
//     }
//
type ArgRewriter struct {
    Capture []Field
    Formatter string
}

// Bind constructs a new higher-order function that returns the result of
// the input function with the specified args automatically bound to arguments
// on the input.
//
// Each arg's Name must match a name on the input FunctionSignature. Each Type,
// if specified, must match the Type of that arg on the input FunctionSignature
// with a matching name.
//
// The name argument sets the name of the created function.
func (fs FunctionSignature) Bind(name string, xargs []Field) (Function, error) {
    return bind(fs, name, xargs, nil)
}

// Bind constructs a new higher-order function that returns the result of
// the input function with the specified args automatically bound to arguments
// on the input.
//
// Each arg's Name must match a name on the input FunctionSignature. Each Type,
// if specified, must match the Type of that arg on the input FunctionSignature
// with a matching name.
//
// The name argument sets the name of the created function.
//
// The function is inlined in the new function. To avoid this, use
// [FunctionSignature.Bind] instead.
func (f Function) Bind(name string, xargs []Field) (Function, error) {
    return bind(f.Signature, name, xargs, &f)
}
