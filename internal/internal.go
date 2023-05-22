package internal

import (
    "strings"

    "github.com/tawesoft/morph/tag"
)

func Map[X any](fn func(x X) X, xs []X) []X {
    if xs == nil { return nil }
    result := make([]X, len(xs))
    for i := 0; i < len(xs); i++ {
        result[i] = fn(xs[i])
    }
    return xs
}

func AppendComments(comment string, comments ... string) string {
    var elements []string
    if len(comment) == 0 {
        elements = append([]string{}, comments...)
    } else {
        elements = append([]string{comment}, comments...)
    }
    return strings.Join(elements, "\n")
}

func AppendTags(t string, tags ... string) string {
    var elements []string
    if len(t) == 0 {
        elements = []string{}
    } else {
        elements = []string{t}
    }
    for _, tt := range tags {
        key, _, _, ok := tag.NextPair(tt)
        if !ok { continue }
        _, exists := tag.Lookup(t, key)
        if exists { continue }
        elements = append(elements, tt)
    }
    return strings.Join(elements, " ")
}
