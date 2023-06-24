package internal

import (
    "fmt"
    "testing"
)

// This test merely asserts that we can compile and run some generated code
// to check that it gives an expected result.
func Test_GeneratorTests(t *testing.T) {
    generated := `
package main

import (
    "fmt"
)

func main() {
    fmt.Print("example")
}
`
    TestCompileAndRun(t, generated, func(stdout string) (err error) {
        expected := "example"
        if stdout != expected {
            err = fmt.Errorf("expected: %s", expected)
        }
        return err
    })
}
