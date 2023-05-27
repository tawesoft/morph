package funcwrappers

import (
    "fmt"
    "strconv"
    "strings"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/internal"
)

type FieldNotFound struct {
    Name string
}
func (err FieldNotFound) Error() string {
    return fmt.Sprintf("field %q not found", err.Name)
}

// SetArg returns a function that constructs a [morph.FunctionWrapper] for the
// provided Function that...
//
// The wrapper sets the named argument to a given value at
// call time. Name can either be the name of a field in f.Arguments, or a
// number representing the index of a field in f.Arguments.
//
// For example, for a [morph.Function] f that represents the Go function
// `Divide(a float64, b float64) float64` (returns a divided by b), then
// SetArg("b", "2") returns a FunctionWrapper that can construct the
// function `func(a float64) float64` (returns a divided by two).
func SetArg(name string, value string) morph.FunctionWrapper {
    return func(f morph.WrappedFunction) (morph.WrappedFunction, error) {
        target := -1
        if len(name) == 0 {
        } else if n, err := strconv.Atoi(name); err == nil {
            if (n >= 0) && (n <= len(f.Signature.Arguments) - 1) {
                target = n
            }
        } else {
            for i, arg := range f.Signature.Arguments {
                if arg.Name == name {
                    target = i
                    break
                }
            }
        }
        if target < 0 {
            return morph.WrappedFunction{}, FieldNotFound{Name: name}
        }

        fs := f.Signature.Copy()

        var inputs, outputs strings.Builder
        for i, arg := range fs.Arguments {
            if inputs.Len() > 0 {
                inputs.WriteString(", ")
            }
            inputs.WriteRune('$')
            if i == target {
                inputs.WriteString(value)
            } else {
                inputs.WriteString(arg.Name)
            }
        }

        // TODO refactor this common one
        for i := 0; i < len(fs.Returns); i++ {
            if outputs.Len() > 0 {
                outputs.WriteString(", ")
            }
            outputs.WriteRune('$')
            outputs.WriteString(strconv.Itoa(i))
        }

        fs.Name = "__SetArg__" + fs.Name
        fs.Arguments = internal.RemoveElementByIndex(target, fs.Arguments)
        fs.Comment = fmt.Sprintf(
            "$ returns the result of [%s] called with the arguments (%s).",
            f.Signature.Name,
            inputs.String(),
        )

        input := morph.ArgRewriter{
            Capture:   nil,
            Formatter: inputs.String(),
        }
        output := morph.ArgRewriter{
            Capture:   nil,
            Formatter: outputs.String(),
        }

        return morph.WrappedFunction{
            Signature: fs,
            Inputs:    input,
            Outputs:   output,
            Wraps:     &f,
        }, nil
    }
}

// SimpleRewriteResults constructs a [morph.FunctionWrapper] that rewrites a
// function's results.
//
// Mapper is a code expression that lists the new results, for example
// "$0 * 2, $1 == nil", constructed from the original results of types
// float64 and error.
//
// The types, for example "float64, bool", represent the new return types after
// applying the mapper.
//
// This function has a limited syntax; in particular it cannot support
// rewriting a result tuple into multiple fields. Use a [ResultRewriter] for
// more advanced use.
func SimpleRewriteResults(mapper string, types string) morph.FunctionWrapper {
    return func(f morph.WrappedFunction) (morph.WrappedFunction, error) {
        fs := f.Signature.Copy()

        var inputs, outputs strings.Builder

        // TODO refactor this common one
        for i := 0; i < len(fs.Arguments); i++ {
            if inputs.Len() > 0 {
                inputs.WriteString(", ")
            }
            inputs.WriteString(fs.Arguments[i].Name)
        }

        // TODO use proper parser here
        fs.Returns = internal.Map(func (x string) morph.Field {
            return morph.Field{Type: strings.TrimSpace(x)}
        }, strings.Split(types, ","))

        // TODO refactor this common one
        for i := 0; i < len(fs.Returns); i++ {
            if outputs.Len() > 0 {
                outputs.WriteString(", ")
            }
            outputs.WriteString(fmt.Sprintf("$%d", i))
        }

        fs.Name = "__RewriteResults__" + fs.Name
        fs.Comment = fmt.Sprintf(
            "$ returns the result of [%s] with the result rewritten as (%s).",
            f.Signature.Name,
            mapper,
        )

        input := morph.ArgRewriter{
            Capture:   nil,
            Formatter: inputs.String(),
        }
        output := morph.ArgRewriter{
            Capture:   nil,
            Formatter: outputs.String(),
        }

        return morph.WrappedFunction{
            Signature: fs,
            Inputs:    input,
            Outputs:   output,
            Wraps:     &f,
        }, nil
    }
}
