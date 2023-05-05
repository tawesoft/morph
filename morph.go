// Package morph generates Go code to map values between related struct types.
//
// - without runtime reflection.
//
// - without stuffing a new domain-specific language into struct field tags.
//
// - with a simple, fully programmable mapping described in native Go code.
//
// - where you can map to existing types, or use Morph to automatically
//   generate new types.
//
// Developed by [Tawesoft Ltd].
//
// [Tawesoft Ltd]: https://www.tawesoft.co.uk/
//
// # Security Model
//
// WARNING: It is assumed that all inputs are trusted. DO NOT accept arbitrary
// input from untrusted sources in any circumstances.
package morph

func must[T any](result T, err error) T {
    if err == nil { return result }
    panic(err)
}

/*
func generateFunc(signature Signature, returnType string, assignments []assignment) []byte {
    var sb bytes.Buffer
    fmt.Fprintf(&sb, "func %s {\n\treturn %s{\n",
        signature.s.Source,
        returnType,
    )

    for _, asgn := range assignments {
        fmt.Fprintf(&sb, "\t\t%s: %s,\n", asgn.Name, asgn.Value)
    }

    sb.WriteString("\t}\n}\n")
    return sb.Bytes()
}
*/
