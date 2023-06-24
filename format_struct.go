package morph

import (
    "bytes"
    "fmt"
    "go/format"
    "strings"

    "github.com/tawesoft/morph/internal"
    "github.com/tawesoft/morph/tag"
)

// rewriteString2 performs the special '$'-token replacement in
// a field expression, function signature or comment, as described by
// [FieldExpression], for a 2-target expression.
//
// The standalone token "$", which may appear in a function signature or
// function comment, is left unchanged.
//
// leftToken and rightToken control $-token the replacement style such as "$a",
// "$b" or "$src", "$dest", for leftToken and rightToken "a" and "b" or "src"
// and "dest", respectively.
//
// The provided Field may be the zero value, in which case '$'-token
// replacements are not applied where they involve a concept of a current
// field.
func (fet FieldExpressionType) rewriteString2(
    sig string,
    operation string,
    leftToken string,
    leftStruct Struct,
    leftArgument Argument,
    leftField Field,
    rightToken string,
    rightStruct Struct,
    rightArgument Argument,
) (string, error) {
    if (fet.Targets != 2) {
        panic(fmt.Errorf(
            "invalid FieldExpressionType.rewriteString2 call on a target of "+
            "type %q with %d target(s) (expected 2 targets)",
            fet.Type, fet.Targets,
        ))
    }
    tr := internal.TokenReplacer{
        Single: func() (string, bool) {
            return operation, len(operation) > 0
        },
        ByName: func(name string) (string, bool) {
            if name == leftToken {
                return leftArgument.Name, true
            } else if name == rightToken {
                return rightArgument.Name, true
            } else {
                return "", false
            }
        },
        FieldByName: func(structName string, fieldName string) (string, bool) {
            if (structName == leftToken) && (leftField.Type != "" ) {
                return leftArgument.Name + "." + fieldName, true
            } else if (structName == rightToken) && (leftField.Type != "") {
                return rightArgument.Name + "." + fieldName, true
            } else {
                return "", false
            }
        },
        Modifier: func(kw string, target string) (string, bool) {
            if kw == "" {
                // "struct.$" for current field
                if (target == leftArgument.Name) && (leftField.Type != "" ) {
                    return target + "." + leftField.Name, true
                } else if (target == rightArgument.Name) && (leftField.Type != "" ) {
                    return target + "." + leftField.Name, true
                }
            } else if kw == "type" {
                // "struct.$type" or
                if s, f, ok := strings.Cut(target, "."); ok {
                    // "struct.field.$type"
                    if s == leftArgument.Name {
                        f, ok := leftStruct.namedField(f)
                        return f.Type, ok
                    } else if s == rightArgument.Name {
                        f, ok := rightStruct.namedField(f)
                        return f.Type, ok
                    }
                } else {
                    // "struct.$type"
                    if target == leftArgument.Name {
                        return leftArgument.Type, true
                    } else if target == rightArgument.Name {
                        return rightArgument.Type, true
                    }
                }
                return "", false
            } else if kw == "title" {
                if len(target) > 0 {
                    // TODO unicode
                    return strings.ToUpper(string(target[0])) + target[1:], true
                }
                return "", true
            } else if kw == "untitle" {
                if len(target) > 0 {
                    // TODO unicode
                    return strings.ToLower(string(target[0])) + target[1:], true
                }
                return "", true
            }
            return "", false
        },
    }
    tr.SetDefaults()
    return tr.Replace(sig)
}

/*
// rewriteStringSrcDest performs the special '$'-token replacement in
// a field expression, function signature or comment, as described by
// [FieldExpression], for a 2-target assignment expression.
//
// The standalone token "$", which may appear in a function signature or
// function comment, is left unchanged.
func (fet FieldExpressionType) rewriteStringSrcDest(
    sig string,
    operation string,
    src Struct,
    srcArgument Argument,
    srcField Field,
    dest Struct,
    destArgument Argument,
) (string, error) {
    return fet.rewriteString2(
        sig, FieldExpressionTypeValue, operation,
        "src",  src,  srcArgument,  srcField,
        "dest", dest, destArgument,
    )
}
*/

/*
// rewriteStringAB performs the special '$'-token replacement in
// a field expression, function signature or comment, as described by
// [FieldExpression], for a 2-target boolean expression.
//
// The standalone token "$", which may appear in a function signature or
// function comment, is left unchanged.
func (fet FieldExpressionType) rewriteStringAB(
    sig string,
    src Struct,
    srcArgument Argument,
    srcField Field,
    dest Struct,
    destArgument Argument,
) (string, error) {
    return fet.rewriteString2(
        sig, FieldExpressionTypeBool, operation,
        "a",  src,  srcArgument,  srcField,
        "b", dest, destArgument,
    )
}
*/

// rewriteStringSelf performs the special '$'-token replacement in
// a field expression, function signature or comment, as described by
// [FieldExpression], for a single-target expression.
//
// The standalone token "$", which may appear in a function signature or
// function comment, is set to the name of the function.
func (fet FieldExpressionType) rewriteString1(
    sig string,
    operation string,
    self Struct,
    arg Argument,
    field Field,
) (string, error) {
    if (fet.Targets != 1) {
        panic(fmt.Errorf(
            "invalid FieldExpressionType.rewriteString1 call on a target of "+
            "type %q with %d target(s) (expected 1 target)",
            fet.Type, fet.Targets,
        ))
    }
    tr := internal.TokenReplacer{
        Single: func() (string, bool) {
            return operation, len(operation) > 0
        },
        ByName: func(name string) (string, bool) {
            if name == "self" {
                return arg.Name, true
            } else if name == "this" {
                return arg.Name + "." + field.Name, true
            } else {
                return "", false
            }
        },
        FieldByName: func(structName string, fieldName string) (string, bool) {
            if (structName == "self") && (field.Type != "") {
                return arg.Name + "." + fieldName, true
                // TODO could check this is a valid field access
                //   but not checking allows mis-parsed method calls to
                //   still succeed
            } else {
                return "", false
            }
        },
        Modifier: func(kw string, target string) (string, bool) {
            if kw == "" {
                // "struct.$" for current field
                if (target == arg.Name) && (field.Type != "") {
                    return target + "." + field.Name, true
                }
            } else if kw == "type" {
                // "struct.$type" or
                if s, f, ok := strings.Cut(target, "."); ok {
                    // "struct.field.$type"
                    if s == arg.Name {
                        f, ok := self.namedField(f)
                        return f.Type, ok
                    }
                } else {
                    // "struct.$type"
                    if target == arg.Name {
                        return arg.Type, true
                    }
                }
                return "", false
            } else if kw == "title" {
                if len(target) > 0 {
                    // TODO unicode
                    return strings.ToUpper(string(target[0])) + target[1:], true
                }
                return "", true
            } else if kw == "untitle" {
                if len(target) > 0 {
                    // TODO unicode
                    return strings.ToLower(string(target[0])) + target[1:], true
                }
                return "", true
            }
            return "", false
        },
    }
    tr.SetDefaults()
    return tr.Replace(sig)
}

// Signature returns the Go type signature of a struct as a string, including
// any generic type constraints, omitting the "type" and "struct" keywords.
//
// For example, returns a result like "Orange" or "Orange[X, Y any]".
func (s Struct) Signature() string {
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

// String returns a Go source code representation of the given struct.
//
// For example, returns a result like:
//
//     // Foo is a thing that bars.
//     type Foo struct {
//         Field Type `tag:"value"` // Comment
//     }
//
func (s Struct) String() string {
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

func (s Struct) matchFieldExpressionType(targets int, operation string) *FieldExpressionType {
    // find matching FieldExpressionType
    var fet *FieldExpressionType
    for _, f := range s.Fields {
        fe := f.GetCustomExpression(operation)
        if fe == nil { continue }
        fet = fe.Type
        break
    }
    if fet == nil {
        return nil
    }
    if fet.Targets != targets {
        return nil
    }
    return fet
}

func (s Struct) CustomUnaryFunction(operation string, signature string) (Function, error) {
    fet := s.matchFieldExpressionType(1, operation)
    if fet == nil {
        return Function{}, fmt.Errorf("no matching unary FieldExpressionType for operation %q", operation)
    }
    return fet.formatStructUnaryFunction(operation, signature, s)
}

// CustomBinaryFunction TODO docs
//
// The struct specified as the method receiver is treated as argument "$a" or
// "$dest" for $-token replacement. The argument specified as "other" is
// treated as argument "$b" or "$src" for $-token replacement.
func (s Struct) CustomBinaryFunction(operation string, signature string, other Struct) (Function, error) {
    fet := s.matchFieldExpressionType(2, operation)
    if fet == nil {
        return Function{}, fmt.Errorf("no matching binary FieldExpressionType for operation %q", operation)
    }
    return fet.formatStructBinaryFunction(operation, signature, s, other)
}

// formatStructUnaryFunction generates Go source code for a function with the given
// signature, performing some operation defined by a FieldExpressionType
// that has a Target of 1.
//
// Omit the leading "func" keyword from the signature.
//
// If the FieldExpressionType has a Type of FieldExpressionTypeValue, then the
// function signature must have a (named or unnamed) return value with a type
// matching the provided struct's type, ignoring generic type constraints, or a
// pointer of that type, or, failing that, at least one method receiver or
// input argument matching the provided struct's type, ignoring generic type
// constraints, which must be a pointer, in which case the first such matching
// input is used.
//
// Otherwise, the function signature must specify at least one method receiver
// or input argument matching the provided struct's type, ignoring generic type
// constraints, or pointer of that type, in which case the first such matching
// input is used.
//
// A function signature may have an additional return value of the error type
// that is used to catch any panics inside the generated function.
func (fet *FieldExpressionType) formatStructUnaryFunction(
    operation string,
    signature string,
    self Struct,
) (Function, error) {
    esc := func(err error) (Function, error) {
        return Function{}, fmt.Errorf(
            "error generating morph.Struct function for struct %q: %w",
            self.Name, err,
        )
    }

    rwsignature, err := fet.rewriteString1(signature, operation, self, Argument{Type: self.Name}, Field{})
    if err != nil {
        return esc(fmt.Errorf("error rewriting function signature %q: %w", signature, err))
    }
    signature = rwsignature

    fs, err := parseFunctionSignatureFromString(signature)
    if err != nil {
        return esc(fmt.Errorf("error parsing function signature %q: %w", signature, err))
    }
    //returnsError := fs.returnsError()

    var arg Argument
    destIsReturnValue := false
    // var expectedReturnArgumentCount TODO
    if fet.Type == FieldExpressionTypeValue {
        if output, isReturnValue, ok := fs.matchingOutput(self.Name); ok {
            arg = output
            destIsReturnValue = isReturnValue
        } else {
            return esc(fmt.Errorf("missing output value argument in signature: %q", fs.String()))
        }
    } else {
        if input, ok := internal.First(
            internal.Filter(argumentTypeFilterer(self.Name), fs.Inputs()),
        ); ok {
            arg = input
        } else {
            return esc(fmt.Errorf("missing input value argument in signature: %q", fs.String()))
        }
    }

    if arg.Name == "" {
        arg.Name = "_unnamed_self"
    }

    feAccessor := fet.defaultAccessor()

    fields := internal.Map(func(f Field) Field {
        f = f.Copy()
        fe := feAccessor(f)

        pattern := ""
        if fe != nil {
            if fe.getType() != fet {
                if fe.getType() != nil {
                    panic(fmt.Errorf("mismatching field expressions types on fields"))
                }
            }
            pattern = fe.Pattern
        }

        if pattern == "" { pattern = fet.Default }
        destArg := arg
        if fet.Type == FieldExpressionTypeValue {
            destArg.Name = "_out"
        }

        rewritten, err := fet.rewriteString1(pattern, operation, self, destArg, f)
        if err != nil {
            panic(fmt.Errorf("cannot rewrite field expression pattern %q: %w", pattern, err))
        }
        pattern = rewritten

        rewritten, err = fet.rewriteString1(fet.FieldComment, operation, self, arg, f)
        if err != nil {
            panic(fmt.Errorf("cannot rewrite field expression type field comment pattern %q: %w", fet.FieldComment, err))
        }
        f.Comment = rewritten

        f.SetCustomExpression(FieldExpression{
            Type:    fet,
            Pattern: pattern,
        })
        return f
    }, self.Fields)

    var body string
    if fet.Type == FieldExpressionTypeVoid {
        //body = formatStructVoidUnaryFunctionBody(arg, fields)
    } else if fet.Type == FieldExpressionTypeBool {
        body = fet.formatStructBooleanFunctionBody(fields)
    } else if fet.Type == FieldExpressionTypeValue {
        body = fet.formatStructValueFunctionBody(arg, destIsReturnValue, fields)
    }

    fs.Comment, err = fet.rewriteString1(fet.Comment, operation, self, arg, Field{})
    if err != nil {
        panic(fmt.Errorf("cannot rewrite field expression type comment pattern %q: %w", fet.Comment, err))
    }
    return Function{
        Signature: fs,
        Body:      body,
    }, nil
}

func (fet *FieldExpressionType) formatStructBinaryFunction(
    operation string,
    signature string,
    aOrDest Struct,
    bOrSrc Struct,
) (Function, error) {
    esc := func(err error) (Function, error) {
        return Function{}, fmt.Errorf(
            "error generating morph.Struct function for structs %q and %q: %w",
            aOrDest.Name, bOrSrc.Name, err,
        )
    }

    var aOrDestToken string // e.g. "a" or "src"; corresponds to "$a" or "$dest".
    var bOrSrcToken string // e.g. "b" or "dest"; corresponds to "$b" or "$src".
    if fet.Type == FieldExpressionTypeValue {
        aOrDestToken = "dest"
        bOrSrcToken  = "src"
    } else {
        aOrDestToken = "a"
        bOrSrcToken  = "b"
    }

    rwsignature, err := fet.rewriteString2(
        signature,
        operation,
        aOrDestToken, aOrDest, Argument{Type: aOrDest.Name}, Field{},
        bOrSrcToken,  bOrSrc,  Argument{Type: bOrSrc.Name},
    )
    if err != nil {
        return esc(fmt.Errorf("error rewriting function signature %q: %w", signature, err))
    }
    signature = rwsignature

    fs, err := parseFunctionSignatureFromString(signature)
    if err != nil {
        return esc(fmt.Errorf("error parsing function signature %q: %w", signature, err))
    }
    //returnsError := fs.returnsError()

    var arg1, arg2 Argument
    destIsReturnValue := false

    // var expectedReturnArgumentCount TODO
    if fet.Type == FieldExpressionTypeValue {
        if output, isReturnValue, ok := fs.matchingOutput(aOrDest.Name); ok {
            arg1 = output
            destIsReturnValue = isReturnValue
        } else {
            return esc(fmt.Errorf("missing output value argument in signature: %q", fs.String()))
        }
    } else {
        if input, ok := internal.First(
            internal.Filter(argumentTypeFilterer(aOrDest.Name), fs.Inputs()),
        ); ok {
            arg1 = input
        } else {
            return esc(fmt.Errorf("missing input value argument in signature: %q", fs.String()))
        }
    }

    if arg1.Name == "" {
        arg1.Name = "_unnamed_" + aOrDestToken
    }

    atf := argumentTypeFilterer(bOrSrc.Name)
    filter := func(f Argument) bool {
        return f.Name != arg1.Name && atf(f)
    }
    if input, ok := internal.First(
        internal.Filter(filter, fs.Inputs()),
    ); ok {
        arg2 = input
    } else {
        return esc(fmt.Errorf("missing input value argument in signature: %q", fs.String()))
    }

    feAccessor := fet.defaultAccessor()

    fields := internal.Map(func(f Field) Field {
        f = f.Copy()
        fe := feAccessor(f)

        pattern := ""
        if fe != nil {
            if fe.getType() != fet {
                if fe.getType() != nil {
                    panic(fmt.Errorf("mismatching field expressions types on fields"))
                }
            }
            pattern = fe.Pattern
        }

        if pattern == "" { pattern = fet.Default }

        destArg := arg1
        if fet.Type == FieldExpressionTypeValue {
            destArg.Name = "_out"
        }

        rewritten, err := fet.rewriteString2(pattern, operation, aOrDestToken, aOrDest, destArg, f, bOrSrcToken, bOrSrc, arg2)
        if err != nil {
            panic(fmt.Errorf("cannot rewrite field expression pattern %q: %w", pattern, err))
        }
        pattern = rewritten

        rewritten, err = fet.rewriteString2(fet.FieldComment, operation, aOrDestToken, aOrDest, arg1, f, bOrSrcToken, bOrSrc, arg2)
        if err != nil {
            panic(fmt.Errorf("cannot rewrite field expression type field comment pattern %q: %w", fet.FieldComment, err))
        }
        f.Comment = rewritten

        f.SetCustomExpression(FieldExpression{
            Type:    fet,
            Pattern: pattern,
        })
        return f
    }, aOrDest.Fields)

    var body string
    if fet.Type == FieldExpressionTypeVoid {
        //body = formatStructVoidUnaryFunctionBody(arg, fields)
    } else if fet.Type == FieldExpressionTypeBool {
        body = fet.formatStructBooleanFunctionBody(fields)
    } else if fet.Type == FieldExpressionTypeValue {
        body = fet.formatStructValueFunctionBody(arg1, destIsReturnValue, fields)
    }

    fs.Comment, err = fet.rewriteString2(fet.Comment, operation, aOrDestToken, aOrDest, arg1, Field{}, bOrSrcToken, bOrSrc, arg2)
    if err != nil {
        panic(fmt.Errorf("cannot rewrite field expression type comment pattern %q: %w", fet.Comment, err))
    }
    return Function{
        Signature: fs,
        Body:      body,
    }, nil
}

func formatComment(indent string, comment string) string {
    var sb strings.Builder
    for _, line := range strings.Split(comment, "\n") {
        sb.WriteString(fmt.Sprintf("%s// %s\n", indent, line))
    }
    return sb.String()
}

func (fet *FieldExpressionType) formatStructBooleanFunctionBody(
    fields []Field,
) string {
    var sb bytes.Buffer

    feAccessor := fet.defaultAccessor()

    for i, f := range fields {
        if i > 0 { sb.WriteString("\n") }

        sb.WriteString(formatComment("\t", f.Comment))

        fe := feAccessor(f)
        if fe == nil {
            panic("accessor returned nil field expression")
        }

        pattern := fe.Pattern
        if pattern == "skip" {
            sb.WriteString("\t//skipped\n")
            continue
        }
        sb.WriteString(fmt.Sprintf("\t_cmp%d := bool(%s)\n", i, pattern))
        if fet.Collect == "||" {
            sb.WriteString(fmt.Sprintf("\tif _cmp%d { return true }\n", i))
        } else if fet.Collect == "&&" {
            sb.WriteString(fmt.Sprintf("\tif !_cmp%d { return false }\n", i))
        } else {
            panic("invalid field expression Collect value")
        }
    }

    if len(fields) > 0 { sb.WriteString("\n") }
    if fet.Collect == "||" {
        sb.WriteString("\treturn false")
    } else if fet.Collect == "&&" {
        sb.WriteString("\treturn true")
    }

    return sb.String()
}

func (fet *FieldExpressionType) formatStructValueFunctionBody(
    dest Argument,
    destIsReturnValue bool,
    fields []Field,
) string {
    var sb bytes.Buffer

    sb.WriteString(fmt.Sprintf("\t_out := %s{}\n\n", strings.TrimPrefix(dest.Type, "*")))

    feAccessor := fet.defaultAccessor()

    for i, f := range fields {
        if i > 0 { sb.WriteString("\n") }

        sb.WriteString(formatComment("\t", f.Comment))

        fe := feAccessor(f)
        if fe == nil {
            panic("accessor returned nil field expression")
        }

        pattern := fe.Pattern
        if pattern == "skip" {
            sb.WriteString("\t//skipped\n")
            continue
        }
        sb.WriteString(fmt.Sprintf("\t%s\n", pattern))
    }

    if len(fields) > 0 { sb.WriteString("\n") }

    if destIsReturnValue {
        if strings.HasPrefix(dest.Type, "*") {
            sb.WriteString("\treturn &_out")
        } else {
            sb.WriteString("\treturn _out")
        }
    } else {
        sb.WriteString("\t*dest = _out")
    }

    return sb.String()
}


// OLD:

// formatStructConverter formats the function body created by the
// [Struct.Converter] method.
func __formatStructConverter(returnType string, assignments []Field) string {
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
        if asgn.Converter == "" {
            // TODO just assign with =
            sb.WriteString(fmt.Sprintf("\t\t// %s is the zero value.\n", asgn.Name))
        } else if asgn.Converter == "nil" {
            sb.WriteString(fmt.Sprintf("\t\t// %s is the zero value.\n", asgn.Name))
        } else {
            sb.WriteString(fmt.Sprintf("\t\t%s: %s,\n", asgn.Name, asgn.Converter))
        }
    }
    sb.WriteString("\t}")
    return sb.String()
}

// formatStructComparer formats the function body created by the
// [Struct.Comparer] method.
func __formatStructComparer(arg1Name, arg2Name string, fs []Field) string {
    // source code representation
    var sb bytes.Buffer

    for i, f := range fs {
        sb.WriteString(fmt.Sprintf("\t// %s.%s == %s.%s\n",
            arg1Name, f.Name, arg2Name, f.Name))
        if f.Comparer == "" {
            sb.WriteString(fmt.Sprintf("\t_cmp%d := (%s.%s == %s.%s)\n\n",
                i, arg1Name, f.Name, arg2Name, f.Name))
        } else {
            sb.WriteString(fmt.Sprintf("\t_cmp%d := bool(%s)\n\n", i, f.Comparer))
        }
    }

    if len(fs) == 0 {
        sb.WriteString("return true\n")
    } else {
        sb.WriteString("return (")
        for i := 0; i < len(fs); i++ {
            if i > 0 { sb.WriteString(" && ") }
            sb.WriteString(fmt.Sprintf("_cmp%d", i))
        }
        sb.WriteString(")")
    }

    return sb.String()
}

// formatStructCopier formats the function body created by the
// [Struct.Copier] method.
func __formatStructCopier(inputName string, outputName string, returnType string, fs []Field) string {
    // source code representation
    var sb bytes.Buffer

    // Remove pointer from temporary output type.
    outType := returnType
    if strings.HasPrefix(returnType, "*") {
        outType = outType[1:]
    }
    sb.WriteString(fmt.Sprintf("\tvar %s %s\n\n", outputName, returnType))

    for _, f := range fs {
        sb.WriteString(fmt.Sprintf("\t// %s.%s = %s.%s\n",
            outputName, f.Name, inputName, f.Name))
        if f.Copier == "" {
            sb.WriteString(fmt.Sprintf("\t%s.%s = %s.%s\n\n",
                outputName, f.Name, inputName, f.Name))
        } else {
            sb.WriteString(fmt.Sprintf("\t%s\n\n", f.Copier))
        }
    }

    // Restore pointer
    sb.WriteString("\treturn ")
    if strings.HasPrefix(returnType, "*") {
        sb.WriteRune('&')
    }
    sb.WriteString(outputName)

    return sb.String()
}

// formatStructOrderer formats the function body created by the
// [Struct.Orderer] method.
func __formatStructOrderer(arg1Name, arg2Name string, fs []Field) string {
    // source code representation
    var sb bytes.Buffer

    for i, f := range fs {
        sb.WriteString(fmt.Sprintf("\t// %s.%s < %s.%s\n",
            arg1Name, f.Name, arg2Name, f.Name))
        if f.Orderer == "" {
            sb.WriteString(fmt.Sprintf("\t_cmp%d := (%s.%s < %s.%s)\n",
                i, arg1Name, f.Name, arg2Name, f.Name))
        } else {
            sb.WriteString(fmt.Sprintf("\t_cmp%d := bool(%s)\n", i, f.Orderer))
        }
        sb.WriteString("if _cmp { return true }\n\n")
    }

    sb.WriteString("return false\n")
    return sb.String()
}
