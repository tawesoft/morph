package morph

// ConverterFieldExpression is an assignment [FieldExpression]-like value for
// mapping a field value on a source struct value to a field value on a
// destination struct value, where each struct value or field value may be of
// different types or otherwise have different semantics.
//
// The default is to assign the source field unchanged.
type ConverterFieldExpression string
func (ConverterFieldExpression) Type() FieldExpressionType {
    return FieldExpressionType{ // TODO const
        Name:    "(builtin) Converter",
        Targets: 2,
        Type:    FieldExpressionTypeValue,
        Default: "$dest.$ = $src.$",
        Comment: "$ converts a value of type [$src.$type] to a value of type [$dest.$type].",
    }
}

// ComparerFieldExpression is a boolean [FieldExpression]-like value for
// comparing two matching field values on struct values of the same type.
//
// A function generated using this expression returns immediately on evaluating
// any expression that evaluates to false.
//
// The default is to compare with "==".
type ComparerFieldExpression string
func (ComparerFieldExpression) Type() FieldExpressionType {
    return FieldExpressionType{ // TODO const
        Name:    "(builtin) Comparer",
        Targets: 2,
        Type:    FieldExpressionTypeBool,
        Default: "$a.$ == $b.$",
        Comment: "$ returns true if $a equals $b.",
        Collect: "&&",
    }
}

// CopierFieldExpression is an assignment [FieldExpression]-like for mapping a
// field value on a source struct value to a matching field value on a
// destination struct value of the same type.
//
// The default is to assign with "=".
type CopierFieldExpression string
func (x *CopierFieldExpression) setPattern(pattern string) {
    *x =  CopierFieldExpression(pattern)
}
func (CopierFieldExpression) getType() *FieldExpressionType {
    return &copierFieldExpressionType
}
var copierFieldExpressionType = FieldExpressionType{
    Name:    "(builtin) Copier",
    Targets: 2,
    Type:    FieldExpressionTypeValue,
    Default: "$dest.$ = $src.$",
    Comment: "$ copies $src to $dest.",
}

// OrdererFieldExpression is a boolean [FieldExpression]-like value for
// defining a sorting order by comparing the values of two matching fields on
// two struct values of the same type.
//
// A function generated using this expression returns immediately on evaluating
// any expression that evaluates to false.
//
// The default is to compare with "<" (less than).
type OrdererFieldExpression string
func (OrdererFieldExpression) getType() *FieldExpressionType {
    return &ordererFieldExpressionType
}
var ordererFieldExpressionType = FieldExpressionType{
    Name:    "(builtin) Orderer",
    Targets: 2,
    Type:    FieldExpressionTypeBool,
    Default: "$a.$ < $b.$",
    Comment: "$ returns true if $a is less than $b.",
    Collect: "&&",
}

// ZeroerFieldExpression is an assignment [FieldExpression]-like value for
// assigning zero to a destination field value, for some concept of a zero
// value for that type.
//
// A function generated using this expression returns immediately on evaluating
// any expression that evaluates to false.
//
// The default is to assign with an automatically generated "zero" stub
// function.
type ZeroerFieldExpression string
func (ZeroerFieldExpression) Type() *FieldExpressionType {
    return &FieldExpressionType{ // TODO const
        Name:    "(builtin) Zeroer",
        Targets: 1,
        Type:    FieldExpressionTypeValue,
        Default: "$this = (func() (_zero $this.$type) { return })()",
        Comment: "$ sets every field on $self to the zero value.",
    }
}

// IsZeroFieldExpression is a boolean [FieldExpression]-like value for
// comparing each field on an input to some concept of a zero value for that
// type.
//
// A function generated using this expression returns immediately on evaluating
// any expression that evaluates to false.
//
// The default is to compare with the Go-defined zero value for that type.
type IsZeroFieldExpression string
func (IsZeroFieldExpression) Type() *FieldExpressionType {
    return &FieldExpressionType{ // TODO const
        Name:    "(builtin) IsZero",
        Targets: 1,
        Type:    FieldExpressionTypeBool,
        Default: "$this == (func() (_zero $this.$type) { return })()",
        Comment: "$ returns true if every field on $self is zero.",
        Collect: "&&",
    }
}
