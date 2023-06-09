package morph

import (
    "errors"
    "fmt"
    "strconv"
    "strings"

    "github.com/tawesoft/morph/internal"
)

// String formats a function or method as Go source code.
//
// For example, gives a result like:
//
//     // Foo bars a baz.
//     func Foo(baz Baz) Bar {
//         /* function body */
//     }
//
func (fn Function) String() string {
    var sb strings.Builder
    comment := fn.Signature.Comment
    comment = strings.ReplaceAll(comment, "$", fn.Signature.Name) // TODO properly
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
    out, err := internal.FormatSource(source)
    if err != nil {
        return fmt.Sprintf(
            "// error formatting function: %v\n// %s\n",
            err,
            strings.Join(strings.Split(source, "\n"), "\n//"),
        )
    }
    return out
}

// String formats the function signature as Go source code, omitting the
// leading "func" keyword.
func (fs FunctionSignature) String() string {
    var sb strings.Builder
    if fs.Receiver.Type != "" {
        receiver := fmt.Sprintf("(%s %s) ", fs.Receiver.Name, fs.Receiver.Type)
        sb.WriteString(receiver)
    }
    sb.WriteString(fs.Name)
    if len(fs.Type) > 0 {
        sb.WriteRune('[')
        for i, arg := range fs.Type {
            sb.WriteString(arg.Name)
            sb.WriteRune(' ')
            sb.WriteString(arg.Type)
            if (i < len(fs.Type) - 1) {
                sb.WriteRune(',')
            }
        }
        sb.WriteRune(']')
    }
    fs.writeArgs(&sb)
    fs.writeReturns(&sb)
    return sb.String()
}

// Value formats the function signature as Go source code as a value, without
// the leading func keyword, and its name omitted.
//
// Methods are rewritten as functions with their receiver inserted at the
// start of the function's arguments.
//
// Generic functions cannot be written this way.
func (fs FunctionSignature) Value() (string, error) {
    var sb strings.Builder
    if len(fs.Type) > 0 {
        return "", fmt.Errorf(
            "cannot format function %s: cannot format a generic function as a value",
            fs.Name,
            // TODO capture proper error with full function signature
        )
    }
    if fs.Receiver.Type != "" {
        // move reciever to first arg
        args := []Argument{fs.Receiver}
        args = append(args, fs.Arguments...)
        fs = fs.Copy()
        fs.Receiver = Argument{}
        fs.Arguments = args
    }
    fs.writeArgs(&sb)
    fs.writeReturns(&sb)
    return sb.String(), nil
}

func (fs FunctionSignature) writeArgs(sb *strings.Builder) {
    sb.WriteRune('(')
    for _, arg := range fs.Arguments {
        sb.WriteString(arg.Name)
        sb.WriteRune(' ')
        sb.WriteString(arg.Type)
        sb.WriteRune(',')
    }
    sb.WriteRune(')')
}

func (fs FunctionSignature) writeReturns(sb *strings.Builder) {
    if len(fs.Returns) > 0 {
        sb.WriteString(" (")
        for _, arg := range fs.Returns {
            sb.WriteString(arg.Name)
            sb.WriteRune(' ')
            sb.WriteString(arg.Type)
            sb.WriteRune(',')
        }
        sb.WriteRune(')')
    }
}

type FunctionError struct {
    Message string
    Signature FunctionSignature
    Reason error
}

func (e FunctionError) Error() string {
    return fmt.Sprintf(
        "Function %s: error: %s: %s",
        e.Signature.Name,
        e.Message,
        e.Reason,
    )
}

var errorWrappedFunctionImplementsItself = errors.New("wrapped function implements itself")

// Function returns the result of converting a wrapped function into a
// concrete implementation representing Go source code.
func (w WrappedFunction) Function() (Function, error) {
    var sb strings.Builder
    esc := func(reason error) (Function, error) {
        return Function{}, FunctionError{
            Message:   "cannot create function from wrapped function",
            Signature: w.Signature.Copy(),
            Reason:    reason,
        }
    }

    if w.Wraps == nil {
        return esc(errorWrappedFunctionImplementsItself)
    }

    reversed := make([]*WrappedFunction, 0)
    for current := &w; current.Wraps != nil; current = current.Wraps {
        reversed = append(reversed, current)
    }

    for i, current := range reversed[1:] {
        sb.WriteString("// from ")
        sb.WriteString(current.Signature.Name)
        sb.WriteString(fmt.Sprintf("\n\t_f%d := func ", i))
        sig, err := current.Signature.Value()
        if err != nil { return esc(err) }
        sb.WriteString(sig)
        sb.WriteString(" {\n")
        if i == 0 {
            writeWrappedFunctionBody(current, &sb, "\t\t", current.Wraps.Signature.Name)
        } else {
            writeWrappedFunctionBody(current, &sb, "\t\t", fmt.Sprintf("_f%d", i-1))
        }
        sb.WriteString("\t}\n\n")
    }

    var name string
    if len(reversed) > 1 {
        name = "_f0"
    } else {
        name = w.Wraps.Signature.Name
    }

    err := writeWrappedFunctionBody(&w, &sb, "\t", name)
    if err != nil {
        return esc(fmt.Errorf("error generating function body: %w", err))
    }

    return Function{
        Signature: w.Signature,
        Body:      sb.String(),
    }, nil
}

type captureResult struct {
    Name string
    Types []string
    Value string
}

func (cr *captureResult) Capture(Type string) {
    cr.Types = append(cr.Types, Type)
}

func tokenReplacerForArgs(referenced internal.Set[int], inputs []Argument) internal.TokenReplacer {
    return internal.TokenReplacer{
        ByIndex: func(i int) (string, bool) {
            if (i < 0) || (i >= len(inputs)) { return "", false }
            arg := inputs[i]
            referenced.Add(i)
            return arg.Name, true
        },
        ByName: func(name string) (string, bool) {
            for i, arg := range inputs {
                if arg.Name == name {
                    referenced.Add(i)
                    return arg.Name, true
                }
            }
            return "", false
        },
        TupleByIndex: func(int, int) (string, bool) { return "", false },
        TupleByName:  func(string, int) (string, bool) { return "", false },
    }
}

func tokenReplacerForCaptures(crprefix string, inputs []captureResult) internal.TokenReplacer {
    return internal.TokenReplacer{
        ByIndex: func(i int) (string, bool) {
            if (i < 0) || (i >= len(inputs)) { return "", false }
            arg := inputs[i]
            return crprefix+strconv.Itoa(i), len(arg.Types) == 1
        },
        ByName: func(name string) (string, bool) {
            for i, arg := range inputs {
                if arg.Name == name {
                    return crprefix+strconv.Itoa(i), len(arg.Types) == 1
                }
            }
            return "", false
        },
        TupleByIndex: func(i int, j int) (string, bool) {
            if (i < 0) || (i >= len(inputs)) { return "", false }
            arg := inputs[i]
            if (j < 0) || (j >= len(arg.Types)) { return "", false }
            return crprefix+strconv.Itoa(i)+"_"+strconv.Itoa(j), len(arg.Types) > 1
        },
        TupleByName: func(name string, j int) (string, bool) {
            for i, arg := range inputs {
                if arg.Name == name {
                    return crprefix+strconv.Itoa(i)+"_"+strconv.Itoa(j), len(arg.Types) > 1
                }
            }
            return "", false
        },
    }
}

func (w ArgRewriter) capture(tr internal.TokenReplacer) ([]captureResult, error) {
    var results []captureResult

    for _, capture := range w.Capture {
        value, err := tr.Replace(capture.Value)
        if err != nil {
            return nil, fmt.Errorf("error replacing capture argument token: %w", err)
        }
        result := captureResult{Name: capture.Name, Value: value}

        types, ok := internal.SplitTypeTuple(capture.Type)
        if !ok { return nil, fmt.Errorf("error parsing type tuple %q", types) } // TODO error type

        for _, Type := range types {
            (&result).Capture(Type)
        }

        results = append(results, result)
    }

    return results, nil
}

// writeCaptureLHS is used to generate source code for the Left Hand Side of a
// capture expression, which may capture zero, one, or a tuple of results from
// the right hand side value.
//
// This writes `prefix` for n = 1, or `prefix_0, prefix_1, ... prefix_n` for
// n > 1.
func (captureResult) writeCaptureLHS(sb *strings.Builder, prefix string, i, n int) {
    if n == 1 {
        sb.WriteString(prefix)
        sb.WriteString(strconv.Itoa(i))
    } else {
        for j := 0; j < n; j++ {
            if j > 0 {
                sb.WriteString(", ")
            }
            sb.WriteString(prefix)
            sb.WriteString(strconv.Itoa(i))
            sb.WriteRune('_')
            sb.WriteString(strconv.Itoa(j))
        }
    }
}

// writeCaptureRHSComment is used to generate source code for the trailing
// comment on the Right Hand Side of a capture expression, which may
// capture zero, one, or a tuple of results from the right hand side value.
//
// An example comment is "// accessible as $0.N or $foo.N".
func (cr captureResult) writeCaptureRHSComment(sb *strings.Builder, i int, n int) {
    if n > 0 {
        sb.WriteString(" // accessible as ")
        sb.WriteString("$")
        sb.WriteString(strconv.Itoa(i))
        if n > 1 { sb.WriteString(".N") }
        if cr.Name != "" {
            sb.WriteString(" or $")
            sb.WriteString(cr.Name)
            if n > 1 { sb.WriteString(".N") }
        }
    }
}

func (cr captureResult) writeCapture(sb *strings.Builder, prefix string, i int) {
    n := len(cr.Types) // no. of variables on left hand side
    cr.writeCaptureLHS(sb, prefix, i, n)
    if n > 0 {
        sb.WriteString(" := ")
    }
    sb.WriteString(cr.Value)
    cr.writeCaptureRHSComment(sb, i, n)
    sb.WriteString("\n")
}

func captureArgs(args []Argument) []captureResult {
    var results []captureResult
    for _, arg := range args {
        result := captureResult{Name: arg.Name}
        (&result).Capture(arg.Type)
        results = append(results, result)
    }
    return results
}

// writeWrappedFunctionBody formats the calling of a wrapped function's
// wrapped inner function, with the name of the inner function call rewritten
// to localInnerFuncName.
func writeWrappedFunctionBody(
    w *WrappedFunction,
    sb *strings.Builder,
    indent string,
    localInnerFuncName string,
) error {
    referenced := internal.NewSet[int]()
    tr := tokenReplacerForArgs(referenced, w.Signature.Arguments)
    inputs, err := w.Inputs.capture(tr)
    if err != nil {
        return fmt.Errorf("error formatting captures for input arguments: %w", err)
    }
    for i, arg := range w.Signature.Arguments {
        if !referenced.Contains(i) {
            return fmt.Errorf("input argument %q not referenced", arg.Name)
        }
    }

    // rewrite inputs (if any) as `_inN := ...` or
    // `_inN_0, _inN_1, ..., _inN_M := ...` where RHS returns a tuple.
    for i, capture := range inputs {
        capture.writeCapture(sb, "_in", i)
    }
    if len(inputs) > 0 { sb.WriteString("\n") }

    // capture outputs (if any) as `_r0, _r1 ... rN := ...`
    returns := w.Wraps.Signature.Returns
    capturedReturns := captureArgs(w.Wraps.Signature.Returns)
    sb.WriteString(indent)
    for i := 0; i < len(returns); i++ {
        if i > 0 { sb.WriteString(", ") }
        sb.WriteString(fmt.Sprintf("_r%d", i))
    }
    if len(returns) > 0 {
        sb.WriteString(" := ")
    }

    // call of inner function localInnerFuncName(_in0, _in1, ... _inN)
    tr = tokenReplacerForCaptures("_in", inputs)
    sb.WriteString(fmt.Sprintf("%s(", localInnerFuncName))
    value, err := tr.Replace(w.Inputs.Formatter)
    if err != nil {
        return fmt.Errorf("error formatting captures for local function call %w", err)
    }

    sb.WriteString(value)
    sb.WriteString(")")

    // named return values
    if len(returns) > 0 {
        sb.WriteString(fmt.Sprintf("%s// results accessible as ", indent))
    }
    for i, r := range returns {
        if i > 0 { sb.WriteString(", ") }
        sb.WriteRune('$')
        if r.Name == "" {
            sb.WriteString(strconv.Itoa(i))
        } else {
            sb.WriteString(r.Name)
        }
    }
    if len(returns) > 0 {
        sb.WriteRune('\n')
    }
    sb.WriteRune('\n')

    tr = tokenReplacerForCaptures("_r", capturedReturns)
    outputs, err := w.Outputs.capture(tr)
    if err != nil {
        return fmt.Errorf("error formatting captures for outputs from %+v: %w", returns, err)
    }

    // rewrite outputs (if any) as `_outN := ...` or
    // `_outN_0, _outN_1, ..., _outN_M := ...` where RHS returns a tuple.
    for i, capture := range outputs {
        capture.writeCapture(sb, "_out", i)
    }
    sb.WriteString("\n")

    // return values
    sb.WriteString(fmt.Sprintf("%sreturn", indent))
    if len(w.Outputs.Formatter) != 0 {
        tr = tokenReplacerForCaptures("_out", outputs)
        value, err = tr.Replace(w.Outputs.Formatter)
        if err != nil {
            return fmt.Errorf("error formatting captures for return: %w", err)
        }
        sb.WriteRune(' ')
        sb.WriteString(value)
    }
    // TODO discard _ types
    return nil
}

// String returns the result of [WrappedFunction.Format].
//
// In the event of error, a suitable error message is formatted as a Go comment
// literal, instead.
func (w WrappedFunction) String() string {
    // TODO rewite error instead of panicking
    return internal.Must(w.Format())
}

func (w WrappedFunction) Format() (string, error) {
    f, err := w.Function()
    if err != nil { return "", err }
    return f.String(), nil
}

// bind implements [FunctionSignature.Bind] and [Function.Bind].
func ___bind(fs FunctionSignature, name string, xargs []Field, inline *Function) (Function, error) {
    // Note, implementation is suboptimal (n^2) but shouldn't matter for
    // small number of args.

    /*
    esc := func(err error) (Function, error) { // TODO proper error type
        return Function{}, fmt.Errorf("FunctionSignature.Bind error: %w", err)
    }
    */

    match := func(f Argument, name string, Type string) bool {
        if (f.Name != name) { return false }
        if (Type != "") && (f.Type != Type) { return false }
        return true
    }
    matchAny := func(a Argument) bool {
        for _, arg := range xargs {
            if match(a, arg.Name, arg.Type) { return true }
        }
        return false
    }
    matchAnyInverse := func(a Argument) bool {
        return !matchAny(a)
    }

    inner := fs.Copy()
    outer := fs.Copy()
    outer.Name = name
    outer.Comment = ""

    if len(xargs) > 0 {
        var cb strings.Builder
        cb.WriteString("$ returns a function that implements [")
        cb.WriteString(fs.Name)
        cb.WriteString("]\nwith the argument")
        if len(xargs) > 1 { cb.WriteString("s") }
        cb.WriteString(" ")
        for i, arg := range xargs {
            if i > 0 {
                if i == len(xargs) - 1 {
                    cb.WriteString(" and ")
                } else {
                    cb.WriteString(", ")
                }
            }
            cb.WriteString(arg.Name)
        }
        cb.WriteString(" already applied.")
        outer.Comment = cb.String()
    }

    inner.Arguments = internal.Filter(matchAnyInverse, inner.Arguments)
    outer.Arguments = internal.Filter(matchAny,        outer.Arguments)

    outer.Returns = []Argument{{
        Type: "func" + internal.Must(inner.Value()),
    }}

    var sb strings.Builder
    sb.WriteString("\treturn func")
    sb.WriteString(internal.Must(inner.Value()))
    sb.WriteString(" {\n")
    sb.WriteString("\t\treturn ")

    if inline == nil {
        sb.WriteString(fs.Name)
    } else {
        sb.WriteString("func ")
        sb.WriteString(internal.Must(inline.Signature.Value()))
        sb.WriteString(" {")
        sb.WriteString(inline.Body)
        sb.WriteString("}")
    }

    sb.WriteString("(")
    for _, arg := range fs.Arguments {
        found := false
        for _, specArg := range xargs {
            if match(arg, specArg.Name, specArg.Type) {
                sb.WriteString(specArg.Name)
                found = true
                break
            }
        }
        if !found {
            sb.WriteString(arg.Name)
        }
        sb.WriteString(", ")
    }
    sb.WriteString(")\t}")

    return Function{
        Signature: outer,
        Body:      sb.String(),
    }, nil
}
