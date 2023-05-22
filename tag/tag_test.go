package tag_test

import (
    "testing"

    "github.com/tawesoft/morph/tag"
)

func TestNextPair(t *testing.T) {
    type result struct {
        key, value, rest string
        ok bool
    }
    tests := []struct {
        input string
        outputs []result
    }{
        {
            input: `tag1:"foo" tag2:"bar" tag3:"foo bar baz"`,
            outputs: []result{
                {"tag1", "foo", `tag2:"bar" tag3:"foo bar baz"`, true},
                {"tag2", "bar", `tag3:"foo bar baz"`, true},
                {"tag3", "foo bar baz", ``, true},
                {"", "", ``, false},
            },
        },
    }

    for i, test := range tests {
        var key, value, rest string
        var ok bool
        rest = test.input
        for j, output := range test.outputs {
            key, value, rest, ok = tag.NextPair(rest)
            got := result{key, value, rest, ok}
            if got != output {
                t.Logf("got: %+v", got)
                t.Logf("expected: %+v", output)
                t.Errorf("test %d output %d: unexpected result", i, j)
            }
        }
    }
}

func TestLookup(t *testing.T) {
    tests := []struct {
        input string
        lookup string
        expected string
        missing bool
    }{
        {
            input: `tag1:"foo" tag2:"bar" tag3:"foo bar baz"`,
            lookup: "tag2",
            expected: "bar",
        },
        {
            input: `tag1:"foo" tag2:"bar" tag3:"foo bar baz"`,
            lookup: "tag3",
            expected: "foo bar baz",
        },
        {
            input: `tag1:"foo" tag2:"bar" tag3:"foo bar baz"`,
            lookup: "tag4",
            missing: true,
        },
    }

    for i, test := range tests {
        value, exists := tag.Lookup(test.input, test.lookup)
        if (!exists) && (test.missing) {
            continue
        } else if (!exists) {
            t.Errorf("test %d: value unexpectedly missing", i)
        } else if value != test.expected {
            t.Logf("got: %q", value)
            t.Logf("expected: %q", test.expected)
            t.Errorf("test %d: unexpected result", i)
        }
    }
}
