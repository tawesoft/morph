package morph

// BuiltinFieldExpression is a [FieldExpression]-like value with a constant
// builtin [FieldExpressionType].
type BuiltinFieldExpression string

var converterFieldExpressionType = &FieldExpressionType{
    Name:    "Converter",
    Targets: 2,
    Type:    FieldExpressionTypeValue,
    Default: "$dest.$ = $src.$",
    Comment: "$ converts a value of type [$src.$type.$name] to a value of type [$dest.$type.$name].",
    FieldComment: "convert $src.$.$type.$name to $dest.$.$type.$name",
    Accessor: func(f Field) string {
        return string(f.Converter)
    },
    Setter: func(f *Field, pattern string) {
        f.Converter = BuiltinFieldExpression(pattern)
    },
}

// StructConverter uses each field's defined Converter [BuiltinFieldExpression] to
// generate a function that maps values from one struct type to another struct
// type.
//
// Converter is an assignment [FieldExpression]-like value for mapping a field
// on a source struct value to a field on a destination struct.
//
// The default is to assign the source field unchanged using "=".
//
// The signature argument is the function signature for the generated function
// (omit any leading "func" keyword). This supports the $-token replacements
// described in [FieldExpression].
func StructConverter(signature string, from Struct, to Struct) (Function, error) {
    fet := converterFieldExpressionType
    return fet.formatStructBinaryFunction(fet.Name, signature, to, from)
}

var comparerFieldExpressionType = &FieldExpressionType{
    Name:    "Comparer",
    Targets: 2,
    Type:    FieldExpressionTypeBool,
    Default: "$a.$ == $b.$",
    Comment: "$ returns true if $a equals $b.",
    FieldComment: "are $a.$. and $b.$ equal?",
    Collect: "&&",
    Accessor: func(f Field) string {
        return string(f.Comparer)
    },
    Setter: func(f *Field, pattern string) {
        f.Comparer = BuiltinFieldExpression(pattern)
    },
}

// Comparer uses each field's defined Comparer [BuiltinFieldExpression]
// to generate a function that compares two struct values of the same type,
// returning true if all matching fields compare equal.
//
// Comparer is a boolean [FieldExpression]-like value for comparing two
// matching fields on struct values of the same type, returning true if they
// compare equal.
//
// A function generated using this expression returns immediately on evaluating
// any expression that evaluates to false.
//
// The default is to compare with "==".
//
// The signature argument is the function signature for the generated function
// (omit any leading "func" keyword). This supports the $-token replacements
// described in [FieldExpression].
func (s Struct) Comparer(signature string) (Function, error) {
    fet := comparerFieldExpressionType
    return fet.formatStructBinaryFunction(fet.Name, signature, s, s)
}

var copierFieldExpressionType = &FieldExpressionType{
    Name:    "Copier",
    Targets: 2,
    Type:    FieldExpressionTypeValue,
    Default: "$dest.$ == $src.$",
    Comment: "$ copies $src to $dest.",
    FieldComment: "copy $src.$ to $dest.$",
    Accessor: func(f Field) string {
        return string(f.Copier)
    },
    Setter: func(f *Field, pattern string) {
        f.Copier = BuiltinFieldExpression(pattern)
    },
}

// Copier uses each field's defined Copier [BuiltinFieldExpression] to
// copy a source struct value to a destination struct value of the same type.
//
// CopierFieldExpression is an assignment [FieldExpression]-like value for
// mapping a field value on a source struct value to a matching field value on
// a destination struct value of the same type.
//
// The default is to assign with "=".
//
// The signature argument is the function signature for the generated function
// (omit any leading "func" keyword). This supports the $-token replacements
// described in [FieldExpression].
func (s Struct) Copier(signature string) (Function, error) {
    fet := copierFieldExpressionType
    return fet.formatStructBinaryFunction(fet.Name, signature, s, s)
}

var ordererFieldExpressionType = &FieldExpressionType{
    Name:    "Orderer",
    Targets: 2,
    Type:    FieldExpressionTypeBool,
    Default: "$a.$ < $b.$",
    Collect: "&&",
    Comment: "$ returns true if $a is less than $b.",
    FieldComment: "is $a.$ less than $b.$",
    Accessor: func(f Field) string {
        return string(f.Orderer)
    },
    Setter: func(f *Field, pattern string) {
        f.Orderer = BuiltinFieldExpression(pattern)
    },
}

// Orderer uses each field's defined Orderer [BuiltinFieldExpression]
// to generate a function that compares two struct values of the same type,
// returning true if all matching fields of the first compare less than all
// matching fields of the second.
//
// Orderer is a boolean [FieldExpression]-like value for comparing two
// matching fields on struct values of the same type, returning true if the
// first is less than the second.
//
// A function generated using this expression returns immediately on evaluating
// any expression that evaluates to false.
//
// The default is to compare with "<".
//
// The signature argument is the function signature for the generated function
// (omit any leading "func" keyword). This supports the $-token replacements
// described in [FieldExpression].
func (s Struct) Orderer(signature string) (Function, error) {
    fet := ordererFieldExpressionType
    return fet.formatStructBinaryFunction(fet.Name, signature, s, s)
}

var zeroerFieldExpressionType = &FieldExpressionType{
    Name:    "Zeroer",
    Targets: 1,
    Type:    FieldExpressionTypeValue,
    Default: "skip",
    Comment: "$ sets every field on $self to the zero value.",
    FieldComment: "set $this to zero",
    Accessor: func(f Field) string {
        return string(f.Zeroer)
    },
    Setter: func(f *Field, pattern string) {
        f.Zeroer = BuiltinFieldExpression(pattern)
    },
}

// Zeroer uses each field's defined Zeroer [BuiltinFieldExpression]
// to generate a function that sets a struct value to its zero value.
//
// Zeroer is an assignment [FieldExpression]-like value for setting a single
// field to its zero value.
//
// The default is to skip the assignment, leaving the field at the zero type
// for that type as specified by the Go language.
//
// The signature argument is the function signature for the generated function
// (omit any leading "func" keyword). This supports the $-token replacements
// described in [FieldExpression].
//
// Note: to conditionally zero a field, use [Struct.Copier] instead.
func (s Struct) Zeroer(signature string) (Function, error) {
    fet := zeroerFieldExpressionType
    return fet.formatStructUnaryFunction(fet.Name, signature, s)
}

var trutherFieldExpressionType = &FieldExpressionType{
    Name:    "Truther",
    Targets: 1,
    Type:    FieldExpressionTypeBool,
    Default: "$this != (func() (_zero $this.$type) { return })()",
    Collect: "||",
    Comment: "$ returns true unless $a is equal to its zero type.",
    FieldComment: "is $this zero?",
    Accessor: func(f Field) string {
        return string(f.Truther)
    },
    Setter: func(f *Field, pattern string) {
        f.Truther = BuiltinFieldExpression(pattern)
    },
}

// Truther uses each field's defined Truther [BuiltinFieldExpression]
// to generate a function that returns true if a struct is considered true.
//
// Truther is a boolean [FieldExpression]-like value for comparing a single
// field to a concept of zero or false.
//
// A function generated using this expression returns immediately on evaluating
// any field that evaluates to non-zero.
//
// The default is to compare with the Go-defined zero value for that type.
//
// The signature argument is the function signature for the generated function
// (omit any leading "func" keyword). This supports the $-token replacements
// described in [FieldExpression].
func (s Struct) Truther(signature string) (Function, error) {
    fet := trutherFieldExpressionType
    return fet.formatStructUnaryFunction(fet.Name, signature, s)
}
