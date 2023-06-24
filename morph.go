// Package morph is a Go code generator that generates code to map between
// structs and manipulate the form of functions.
//
// All types should be considered read-only & immutable except where otherwise
// specified.
//
// Need help? Ask on morph's GitHub issue tracker or check out the tutorials
// in the README. Also, paid commercial support and training is available
// via [Open Source at Tawesoft].
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
    "strings"

    "github.com/tawesoft/morph/internal"
)

// FunctionSignature represents a parsed function signature, including any
// arguments, return types, method receiver, generic type constraints, etc.
type FunctionSignature struct {
    Comment   string
    Name      string
    Type      []Argument
    Arguments []Argument
    Returns   []Argument
    Receiver  Argument
}

// Copy returns a (deep) copy of a FunctionSignature, ensuring that slices
// aren't aliased.
func (fs FunctionSignature) Copy() FunctionSignature {
    out := fs
    out.Type      = append([]Argument(nil), fs.Type...)
    out.Arguments = append([]Argument(nil), fs.Arguments...)
    out.Returns   = append([]Argument(nil), fs.Returns...)
    return out
}

// Inputs returns a slice containing a Argument for each input specified by the
// function signature, including the method receiver (if any) as the first
// argument.
func (fs FunctionSignature) Inputs() []Argument {
    var args []Argument
    if fs.Receiver.Type != "" {
        args = append(args, fs.Receiver)
    }
    args = append(args, fs.Arguments...)
    return args
}

/*
// matchingInput searches the function receiver and input arguments, in order,
// for the first instance of an input matching the provided type, or a pointer
// of the provided type. The provided Type must be a simple type, and type
// constraints are ignored.
func (fs FunctionSignature) matchingInput(Type string) (match Argument, found bool) {
    args := filterArguments(fs.Inputs(), argumentTypeFilterer(Type))
    if len(args) < 1 {
        found = false
        return
    }
    return args[0], true // first one wins
}

// matchingInputs searches the function receiver and input arguments, in order,
// for the first two instances of an input matching the provided type, or a
// pointer of the provided type. The provided Type must be a simple type, and
// type constraints are ignored.
func (fs FunctionSignature) matchingInputs(Type string) (match1 Argument, match2 Argument, found bool) {
    args := filterArguments(fs.Inputs(), argumentTypeFilterer(Type))
    if len(args) < 2 {
        found = false
        return
    }
    return args[0], args[1], true // first two win
}
*/

// Function contains a parsed function signature and the raw source code of
// its body, excluding the enclosing "{" and "}" braces.
type Function struct {
    Signature FunctionSignature
    Body string
}

// Argument represents a named and typed argument e.g. a type constraint or
// an argument to a function.
type Argument struct {
    Name string
    Type string
}

func fieldToArgument(f Field) Argument {
    return Argument{
        Name: f.Name,
        Type: f.Type,
    }
}

// matchSimpleType returns true if an argument matches the provided simple
// type, or matches a pointer to the provided simple type, ignoring type
// constraints.
func (a Argument) matchSimpleType(Type string) bool {
    return internal.MatchSimpleType(a.Type, Type)
}

// argumentTypeFilterer returns a function that returns true for any Argument
// whose type is a simple type that matches the provided Type.
func argumentTypeFilterer(Type string) func (f Argument) bool {
    return func(a Argument) bool {
        return a.matchSimpleType(Type)
    }
}

// Variable represents a named and typed variable with a value.
type Variable struct {
    Name  string
    Type  string
    Value string
}

// Field represents a named and typed field in a struct, with optional struct
// tags, comments, and various "expressions" describing operations on fields
// of that type.
//
// Fields should be considered readonly, except inside a FieldMapper.
//
// The Tag, if any, does not include surrounding quote marks, and the Comment,
// if any, does not have a trailing new line or comment characters such as
// "//", a starting "/*", or an ending "*/".
//
// A field appearing in a struct may also define a reverse function, which is a
// [FieldMapper] that performs the opposite conversion of a FieldMapper applied
// to the field. It is not necessary that the result of applying a reverse
// function is itself reversible. Not all mappings are reversible, so this may
// be set to nil.
//
// For example, if a FieldMapper applies a transformation with
//     Converter == "$dest.$ = $src.$ * 2"
// Then the reverse function should apply a transformation with
//     Converter == "$dest.$ = $src.$ / 2"
//
// When creating a reverse function, care should be taken to not overwrite a
// field's existing reverse function. This can be achieved by composing the two
// functions with the Compose function in the fieldmappers sub package, for
// example:
//     myNewField.Reverse = fieldmappers.Compose(newfunc, myOldField.Reverse))
// (this is safe even when myOldField.Reverse is nil).
type Field struct {
    Name      string
    Type      string
    Tag       string
    Comment   string

    // For fields appearing in structs that have been mapped only...
    Reverse   FieldMapper
    Converter ConverterFieldExpression
    Comparer  ComparerFieldExpression
    Copier    CopierFieldExpression
    Orderer   OrdererFieldExpression
    Zeroer    ZeroerFieldExpression
    Custom    []FieldExpression
}

// Copy returns a (deep) copy of a Field, ensuring that slices aren't aliased.
func (f Field) Copy() Field {
    out := f
    out.Custom = append([]FieldExpression(nil), f.Custom...)
    return out
}

// matchSimpleType returns true if a field matches the provided simple
// type, or matches a pointer to the provided simple type, ignoring type
// constraints.
func (f Field) matchSimpleType(Type string) bool {
    return internal.MatchSimpleType(f.Type, Type)
}

func filterFields(fields []Field, filter func(f Field) bool) []Field {
    var result []Field
    for _, f := range fields {
        if filter(f) {
            result = append(result, f.Copy())
        }
    }
    return result
}

/*
// fieldTypeFilterer returns a function that returns true for any Field whose
// type is a simple type that matches the provided Type.
func fieldTypeFilterer(Type string) func (f Field) bool {
    return func(f Field) bool {
        return f.matchSimpleType(Type)
    }
}
*/

// AppendTags appends a tag to the field's existing tags (if any), joined with
// a space separator, as is the convention.
//
// Note that this modifies the field in-place, so should be done on a copy
// where appropriate.
//
// Each tag in the tags list to be appended should be a single key:value pair.
//
// If a tag in the tags list to be appended is already present in the original
// struct tag string, it is not appended.
//
// If any tags do not have the conventional format, the value returned
// is unspecified.
func (f *Field) AppendTags(tags ... string) {
    f.Tag = internal.AppendTags(f.Tag, tags...)
}

// AppendComments appends a comment to the field's existing comment string (if
// any), joined with a newline separator.
//
// Note that this modifies the field in-place, so should be done on a copy
// where appropriate.
func (f *Field) AppendComments(comments ... string) {
    f.Comment = internal.AppendComments(f.Comment, comments...)
}

// Rewrite performs the special '$' replacement of a field's Name and Type
// described by FieldMapper.
//
// Note that this modifies the field in-place, so should be done on a copy
// where appropriate.
func (f *Field) Rewrite(input Field) {
    // naive strings.Replace is fine here because "$" cannot appear in a
    // valid identifier.
    f.Name  = strings.ReplaceAll(f.Name, "$", input.Name)
    f.Type  = strings.ReplaceAll(f.Type, "$", input.Type)
}

// SetCustomExpression adds a custom field expression to a field's slice of
// custom expressions or, if one with that name already exists, replaces that
// expression.
func (f *Field) SetCustomExpression(expression FieldExpression) {
    for i, x := range f.Custom {
        if x.Type.Name == expression.Type.Name {
            f.Custom[i] = expression
            return
        }
    }
    f.Custom = append(f.Custom, expression)
}

// GetCustomExpression retrieves a named custom field expression from a field's
// slice of custom expressions, or nil if not found.
func (f Field) GetCustomExpression(name string) *FieldExpression {
    for i, x := range f.Custom {
        if x.Type.Name == name {
            return &f.Custom[i]
        }
    }
    return nil
}

// FieldExpression is an instruction describing how to assign, compare, or
// inspect, or perform some other operation on a field or fields.
//
// Most users will just call [Struct.Converter], [Struct.Comparer], etc.
// and do not need to use this unless using Morph to make their own custom
// function generators.
//
// A field expression's Type defines what the expression does e.g. compare,
// convert, etc.
//
// An expression may apply to either a single field on one struct value, or on
// two matching fields on two struct values, depending on the Type.
//
// Note one important limitation: expressions of the same Type overwrite each
// other and do not nest. Two field mappings are incompatible if they touch
// the same expressions on the same fields. When making multiple incompatible
// mappings, you must create intermediate temporary struct types.
//
// A field expression's Pattern defines how a field expression of that Type is
// applied to a specific field or fields.
//
// These are either boolean comparison expressions (like [Field.Comparer] and
// [Field.Orderer]) that return true or false, value assignment expressions
// (like [Field.Converter] and [Field.Copier]) which assign values to a
// destination value, or void inspection expressions that don't return
// anything (but may panic). Fields are inspected in the order they appear in
// a source struct.
//
// In any expression, if the pattern is the special value "skip", then
// it means explicitly ignore that field for expressions of that type
// instead of applying the default pattern for that expression type.
//
// A field expression's pattern may contain "$-token" replacements that are
// replaced with a computed string when generating the source code for a new
// function:
//
//  * "$a" and "$b" are replaced inside two-target boolean expressions with the
//    name of two input struct value arguments.
//
//  * "$src" and "$dest" are replaced inside two-target value assignment
//    expressions with the name of the input and output struct value arguments.
//
//  * "$self" is replaced inside a single-target expression with the name of
//    the single target struct value argument.
//
//  * "$this" is replaced inside a single-target boolean or void expression
//    with the qualified field name on the single input for the field currently
//    being mapped.
//
//  * The tokens "$a", "$b", "$src", "$dest", and "$self" may be followed by a
//    dot and another token specifying a named field (e.g. "$src.Foo"), which
//    is then replaced with a qualified field name on the appropriate target
//    (e.g. "myStruct.Foo").
//
//  * Alternatively, instead of a named field, the tokens "$a", "$b", "$src",
//    "$dest", and "$self" may instead be followed by a dot and "$" when not
//    followed by any Go identifier (e.g. "$src.$.foo" is okay, but "$src.$foo"
//    is not), which is then replaced with the qualified field name on the
//    appropriate target for the source field currently being mapped.
//
// Additionally:
//
//  * Any token that would otherwise be replaced by any previous pattern may
//    also be followed by a dot and "$type", in which case they are instead
//    replaced with the type of the referenced struct value or field value.
//
//  * Any token that would otherwise be replaced by any previous pattern,
//    including with the ".$type" suffix, may also be followed by a dot and
//    "$title" or "$untitle", in which case the replacement has the first
//    character forced into an uppercase or lowercase variant, respectively.
//
// In cases where a $-token is ambiguous, use parentheses e.g. "$(foo)Bar".
// For example, to call a method on "$src", use "$(src).Method()", and to call
// a function that is a field, use "$(src.Method)()".
type FieldExpression struct {
    Type *FieldExpressionType
    Pattern string // e.g. "$dest.$ = append([]$src.$.$type(nil), $src.$)"
}

func (fe FieldExpression) getType() *FieldExpressionType {
    return fe.Type
}
func (fe *FieldExpression) setPattern(pattern string) {
    fe.Pattern = pattern
}

const (
    FieldExpressionTypeVoid  = "void"
    FieldExpressionTypeBool  = "bool"
    FieldExpressionTypeValue = "value"
)

// FieldExpressionType describes some operation (e.g. copier, comparer,
// appender) on a struct value or on two struct values that can be implemented
// for a specific field by a [FieldExpression]. This controls how to generate
// the Go source code for a function that performs that operation.
//
// Most users will just call [Struct.Converter], [Struct.Comparer], etc.
// and do not need to use this unless using Morph to generate their own custom
// types of functions.
type FieldExpressionType struct {
    // Targets specifies that this is an operation over this many input and/or
    // output struct values.
    //
    // Don't count any incidental input or output arguments e.g. database
    // handles or an error return value. These can be specified in a
    // function signature later.
    //
    // Allowed values are 1 and 2.
    Targets int

    // Name uniquely identifies the operation e.g. "Append". A [Field] can
    // only ever have one custom FieldExpression per given Name, and every
    // FieldExpression in a struct's list of fields that share this name must
    // have type fields that all point to the same FieldExpressionType.
    Name string

    // Default is a pattern applied if a pattern on a FieldExpression is the
    // empty string (indicating default behaviour).
    //
    // All "$"-tokens are replaced according to the rules specified by the
    // [FieldExpression] doc comment.
    //
    // An empty Default is treated as "skip".
    Default string

    // Returns specifies if the function is a boolean comparison expression,
    // value assignment expression, or a void inspection expression
    // (see [FieldExpression]).
    //
    // Allowed values are FieldExpressionTypeVoid, FieldExpressionTypeBool,
    // and FieldExpressionTypeValue.
    //
    // If the type is FieldExpressionTypeVoid, then Targets must be less than
    // 2.
    Type string

    // Comment is an optional comment set on a generated function e.g.
    // "$ converts [$src.$type] to [$dest.$type]". Leading "//" tokens are not
    // required, and the comment may contain linebreaks.
    //
    // The "$"-tokens "$a", "$b", "$src", "$dest", and "$self" appearing in
    // Comment are replaced according to the rules specified by the
    // [FieldExpression] doc comment.
    //
    // Additionally, the token "$", when not preceded by a dot, is (at a later
    // time) replaced by the name of the generated function when formatting a
    // function.
    Comment string

    // FieldComment is an optional comment generated above each expression for
    // each field e.g. "does $a.$ equal $b.$?". Leading "//" tokens are not
    // required, and the comment may contain linebreaks.
    //
    // All "$"-tokens are replaced according to the rules specified by the
    // [FieldExpression] doc comment.
    FieldComment string

    // Collect is a logical operator applied to all boolean results when
    // Returns is set to "bool". This must be set to a string representing
    // the Go boolean logical operator "&&" or "||". If set to "||", the
    // generated function returns true immediately on the first true value. If
    // set to "&&", the generated function returns false immediately on the
    // first false value.
    Collect string

    // Accessor returns a pointer to the field expression of this type on a
    // given field, if one exists. If nil, calls [Field.GetCustomExpression]
    // with the FieldExpressionType.Name as an argument.
    Accessor func(f Field) *FieldExpression
}

func (fet FieldExpressionType) defaultAccessor() func(f Field) *FieldExpression {
    if fet.Accessor != nil { return fet.Accessor }
    var Type = fet.Name
    return func(f Field) *FieldExpression {
        return f.GetCustomExpression(Type)
    }
}

    /* Possible future extension to FieldExpressionType:
    // Prepend, if not nil, is a function that generates code to be inserted
    // at the top of a generated function based on the first field (if
    // Targets == 1) or fields on arguments a and b or source and dest
    // respectively (if Targets == 2).
    //
    // State is a newly initialised map for the generated function that can
    // be used to maintain state.
    Prepend func(state map[string]any, a, b Field) string
    */


// FieldMapper maps fields on a struct to fields on another struct.
//
// A FieldMapper is called once for each field defined on an input struct.
// Each invocation of the emit callback function generates a field on the
// output struct.
//
// As a shortcut, a "$" appearing in an emitted Field's Name or Type is
// replaced with the name or type of the input Field, respectively.
//
// It is permitted to call the emit function zero, one, or more than one time
// to produce zero, one, or more fields from a single input field.
//
// For example, for an input:
//    Field{Name: "Foo", Type: "int64"},
// Then calling emit with the argument:
//     emit(Field{Name: "$Slice", Type: "[]$"})
// Would generate a Field with the name "FooSlice" and type `[]int64`.
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
                output.Rewrite(input)
                emit(output.Copy())
            }
            mapper(input, emit2)
        }

        out.Fields = results
        oldReverse := out.Reverse
        out.Reverse = func(in2 Struct) Struct {
            out2 := in2.MapFields(func (input2 Field, emit2 func(output Field)) {
                if input2.Reverse != nil {
                    input2.Reverse(input2.Copy(), emit2)
                } else {
                    emit2(input2.Copy())
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

func (s Struct) namedField(name string) (Field, bool) {
    for _, f := range s.Fields {
        if f.Name == name {
            return f.Copy(), true
        }
    }
    return Field{}, false
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


// returnsError returns true if the last return value is an error type.
func (fs FunctionSignature) returnsError() bool {
    last, ok := internal.Last(fs.Returns)
    if !ok { return false }
    return last.Type == "error"
}

// matchingOutput returns the first argument that is either:
//  * a return value of a matching type (or a pointer of that type)
//  * the first method receiver or input argument that is a pointer of
//    that type.
func (fs FunctionSignature) matchingOutput(Type string) (arg Argument, isReturnValue bool, ok bool) {
    // matching return value
    if output, ok := internal.First(
        internal.Filter(
            argumentTypeFilterer(Type),
            fs.Returns,
        ),
    ); ok {
        return output, true, true
    }

    // matching input argument that must be a pointer
    isPointer := func(x Argument) bool {
        return strings.HasPrefix(x.Type, "*")
    }
    if output, ok := internal.First(
        internal.Filter(
            argumentTypeFilterer(Type),
            internal.Filter(
                isPointer,
                fs.Inputs(),
            ),
        ),
    ); ok {
        return output, false, true
    }

    return Argument{}, false, false
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
    Capture []Variable
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
// func (fs FunctionSignature) Bind(name string, xargs []Field) (Function, error) {
//     return bind(fs, name, xargs, nil)
// }

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
//
// TODO remove this and build a funcwrapper instead.
// func (f Function) Bind(name string, xargs []Field) (Function, error) {
//     return bind(f.Signature, name, xargs, &f)
//}
