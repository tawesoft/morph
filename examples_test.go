package morph_test

import (
    "fmt"
    "time"
    "unsafe"

    "github.com/tawesoft/morph"
    "github.com/tawesoft/morph/fieldmappers"
    "github.com/tawesoft/morph/fieldmappers/fieldops"
    "github.com/tawesoft/morph/structmappers"
)

func must[X any](value X, err error) X {
    if err != nil { panic(err) }
    return value
}

func ExampleParseStruct_fromString() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
    Price     price.Price
}
`

    apple := must(morph.ParseStruct("test.go", source, "Apple"))
    fmt.Println(apple)

    // output:
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
}

type Animal struct {
    Name string
    Born time.Time
}

func ExampleParseStruct_fromFile() {
    animal := must(morph.ParseStruct("examples_test.go", nil, "Animal"))
    fmt.Println(animal)

    // output:
    // type Animal struct {
    //	Name string
    //	Born time.Time
    // }
}

/*
func Example_applesToOranges() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
    Price     price.Price
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        int64ToWeight := func(input morph.Field, emit func(output morph.Field)) {
            output := input
            output.Type = "weight.Weight"
            output.Value = "weight.FromGrams($.$)"
            emit(output)
        }

        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Value = "$.$.Grams()" // rewrite the value
            output.Comment = "grams from weight.Weight"
            output.Reverse = fieldmappers.Compose(int64ToWeight, output.Reverse)
            emit(output)
        } else {
            emit(input)
        }
    }

    orange := apple.Map(
        structmappers.Rename("Orange"),
    ).MapFields(
        fieldmappers.TimeToInt64,
        WeightToInt64,
        // PriceToInt64 -- left as exercise for the reader
    )

    fmt.Println(apple)
    fmt.Println(orange)

    const converterFormat = "($from $From) To$To() $To"
    appleToOrange := must(orange.Converter(converterFormat))

    appleFromOrange := orange.Map(structmappers.Rename("Apple")).MapFields(fieldmappers.Reverse)
    orangeToApple := must(appleFromOrange.Converter(converterFormat))

    fmt.Println(appleToOrange)
    fmt.Println(orangeToApple)

    // output:
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    //	Price     price.Price
    // }
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64 // grams from weight.Weight
    //	Price     price.Price
    // }
    // // ToOrange converts [Apple] to [Orange].
    // func (apple Apple) ToOrange() Orange {
    //	return Orange{
    //		Picked:    apple.Picked.UTC().Unix(),
    //		LastEaten: apple.LastEaten.UTC().Unix(),
    //		Weight:    apple.Weight.Grams(),
    //		// Price is the zero value.
    //	}
    // }
    // // ToApple converts [Orange] to [Apple].
    // func (orange Orange) ToApple() Apple {
    //	return Apple{
    //		Picked:    time.Unix(orange.Picked, 0).UTC(),
    //		LastEaten: time.Unix(orange.LastEaten, 0).UTC(),
    //		Weight:    weight.FromGrams(orange.Weight),
    //		// Price is the zero value.
    //	}
    // }
}
*/

func ExampleFieldMapper() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            emit(output)
        } else {
            emit(input)
        }
    }

    orange := apple.Map(
        structmappers.Rename("Orange"),
    ).MapFields(
        fieldmappers.TimeToInt64,
        WeightToInt64,
    )
    fmt.Println(orange)

    // Output:
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64
    // }
}

func ExampleStruct_Converter() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Value = "$.$.Weight()"
            emit(output)
        } else {
            emit(input)
        }
    }

    orange := apple.Map(
        structmappers.Rename("Orange"),
    ).MapFields(
        fieldmappers.TimeToInt64,
        WeightToInt64,
    )

    functionSignature := "$FromTo$To($from $From) $To"
    fmt.Println(must(orange.Converter(functionSignature)))

    // Output:
    // // AppleToOrange converts [Apple] to [Orange].
    // func AppleToOrange(apple Apple) Orange {
    //	return Orange{
    //		Picked:    apple.Picked.UTC().Unix(),
    //		LastEaten: apple.LastEaten.UTC().Unix(),
    //		Weight:    apple.Weight.Weight(),
    //	}
    // }
}


func ExampleStruct_Converter_reverse() {
    source := `
package example

type Apple struct {
    Picked    time.Time
    LastEaten time.Time
    Weight    weight.Weight
}
`

    apple := must(morph.ParseStruct("test.go", source, ""))

    WeightToInt64 := func(input morph.Field, emit func(output morph.Field)) {
        reverse := func(input morph.Field, emit func(output morph.Field)) {
            output := input
            output.Type = "weight.Weight"
            output.Value = "weight.FromGrams($.$)"
            output.Comment = ""
            emit(output)
        }

        if input.Type == "weight.Weight" {
            output := input // copy
            output.Type = "int64" // rewrite the type
            output.Value = "$.$.Grams()" // rewrite the value
            output.Reverse = fieldmappers.Compose(reverse, output.Reverse)
            output.Comment = "weight in grams"
            emit(output)
        } else {
            emit(input)
        }
    }

    orange := apple.Map(
        structmappers.Rename("Orange"),
    ).MapFields(
        fieldmappers.TimeToInt64,
        WeightToInt64,
    )

    const functionSignature = "$FromTo$To($from $From) $To"

    fmt.Println(orange)
    fmt.Println(must(orange.Converter(functionSignature)))

    appleAgain := orange.Map(structmappers.Reverse)

    fmt.Println(appleAgain)
    fmt.Println(must(appleAgain.Converter(functionSignature)))

    // Output:
    // type Orange struct {
    //	Picked    int64 // time in seconds since Unix epoch
    //	LastEaten int64 // time in seconds since Unix epoch
    //	Weight    int64 // weight in grams
    // }
    // // AppleToOrange converts [Apple] to [Orange].
    // func AppleToOrange(apple Apple) Orange {
    //	return Orange{
    //		Picked:    apple.Picked.UTC().Unix(),
    //		LastEaten: apple.LastEaten.UTC().Unix(),
    //		Weight:    apple.Weight.Grams(),
    //	}
    // }
    // type Apple struct {
    //	Picked    time.Time
    //	LastEaten time.Time
    //	Weight    weight.Weight
    // }
    // // OrangeToApple converts [Orange] to [Apple].
    // func OrangeToApple(orange Orange) Apple {
    //	return Apple{
    //		Picked:    time.Unix(orange.Picked, 0).UTC(),
    //		LastEaten: time.Unix(orange.LastEaten, 0).UTC(),
    //		Weight:    weight.FromGrams(orange.Weight),
    //	}
    // }
}

func ExampleStruct_Comparer() {
    source := `
package example

type Thing struct {
    ID string
    Children []Thing
    Ignored string
    Created time.Time
    Foo int
}
`

    // parse the struct from source code
    thing := must(morph.ParseStruct("test.go", source, ""))

    // set default comparison expressions for time.Time fields
    // i.e. use [time.Equals]
    thing = thing.MapFields(fieldops.Time)

    // manually set some more for the sake of example.
    thing.Fields[0].Comparer = "strings.EqualFold($a.$, $b.$)"
    thing.Fields[1].Comparer = "slices.EqualFunc($a.$, $b.$, ThingEquals)"
    thing.Fields[2].Comparer = "true" // Ignored always compare true
    // thing.Fields[4] not specified, so Foo uses default comparison of ==

    // derive a new struct, and observe it converts the fields of type
    // [time.Time] to fields of type int64 (a Unix time stamp) updates the
    // comparison expression appropriately (to compare integers).
    derivedThing := thing.Map(
        structmappers.Rename("Derived"),
    ).MapFields(
        fieldmappers.TimeToInt64,
    )

    // function signature
    sig := `$Equals(thing1 $, thing2 $)`

    fmt.Println(must(thing.Comparer(sig)))
    fmt.Println(must(derivedThing.Comparer(sig)))

    // Output:
    // // ThingEquals returns true if two [Thing] values are equal.
    // func ThingEquals(thing1 Thing, thing2 Thing) bool {
    //	// thing1.ID == thing2.ID
    //	_cmp0 := bool(strings.EqualFold(thing1.ID, thing2.ID))
    //
    //	// thing1.Children == thing2.Children
    //	_cmp1 := bool(slices.EqualFunc(thing1.Children, thing2.Children, ThingEquals))
    //
    //	// thing1.Ignored == thing2.Ignored
    //	_cmp2 := bool(true)
    //
    //	// thing1.Created == thing2.Created
    //	_cmp3 := bool(thing1.Created.Equals(thing2.Created))
    //
    //	// thing1.Foo == thing2.Foo
    //	_cmp4 := (thing1.Foo == thing2.Foo)
    //
    //	return (_cmp0 && _cmp1 && _cmp2 && _cmp3 && _cmp4)
    // }
    // // DerivedEquals returns true if two [Derived] values are equal.
    // func DerivedEquals(thing1 Derived, thing2 Derived) bool {
    //	// thing1.ID == thing2.ID
    //	_cmp0 := bool(strings.EqualFold(thing1.ID, thing2.ID))
    //
    //	// thing1.Children == thing2.Children
    //	_cmp1 := bool(slices.EqualFunc(thing1.Children, thing2.Children, ThingEquals))
    //
    //	// thing1.Ignored == thing2.Ignored
    //	_cmp2 := bool(true)
    //
    //	// thing1.Created == thing2.Created
    //	_cmp3 := (thing1.Created == thing2.Created)
    //
    //	// thing1.Foo == thing2.Foo
    //	_cmp4 := (thing1.Foo == thing2.Foo)
    //
    //	return (_cmp0 && _cmp1 && _cmp2 && _cmp3 && _cmp4)
    // }
}

func ExampleStruct_Copier_deep() {
    source := `
package example

type Tree[X comparable] struct {
    Value X
    Children []Tree[X]
}
`

    // parse the struct from source code
    tree := must(morph.ParseStruct("test.go", source, "Tree"))

    // manually set for the sake of example.
    tree.Fields[1].Copier = "$dest.$ = append(Tree[X]{}, Map(Tree.Copy, $src.$))"
    // tree.Fields[0] not specified, so Value uses default copier of =

    // function signature
    sig := `$Copy(input $) (output $)`

    fmt.Println(must(tree.Copier(sig)))

    // Output:
    // // TreeCopy returns a copy of the [Tree] input.
    // func TreeCopy(input Tree) (output Tree) {
    //	var _out Tree
    //
    //	// _out.Value = input.Value
    //	_out.Value = input.Value
    //
    //	// _out.Children = input.Children
    //	_out.Children = append(Tree[X]{}, Map(Tree.Copy, input.Children))
    //
    //	return _out
    // }
}

func ExampleStruct_Orderer_deep() {
    source := `
package example

type Tree[X constraints.Ordererd] struct {
    Value X
    Children []Tree[X]
}
`

    // parse the struct from source code
    tree := must(morph.ParseStruct("test.go", source, "Tree"))

    // manually set for the sake of example.
    tree.Fields[1].Orderer = "LessThanFunc($a.$, $b.$, TreeLessThan)"
    // tree.Fields[0] not specified, so Value uses default orderer of <

    // function signature
    sig := `$LessThan(first, second $)`

    fmt.Println(must(tree.Orderer(sig)))

    // Output:
    // // TreeLessThan returns true if the first [Tree] is less than the second.
    // func TreeLessThan(first Tree, second Tree) bool {
    //	// first.Value < second.Value
    //	_cmp0 := (first.Value < second.Value)
    //	if _cmp {
    //		return true
    //	}
    //
    //	// first.Children < second.Children
    //	_cmp1 := bool(LessThanFunc(first.Children, second.Children, TreeLessThan))
    //	if _cmp {
    //		return true
    //	}
    //
    //	return false
    //
    // }
}

type List[X comparable] struct {
    Value X
    Next *List[X]
}

type visit struct {
    a, b unsafe.Pointer
}

type visitor map[visit]struct{}

func (v visitor) seen(a, b unsafe.Pointer) bool {
    if uintptr(a) > uintptr(b) {
        a, b = b, a
    }
    _, exists := v[visit{a, b}]
    return exists
}

func (v visitor) mark(a, b unsafe.Pointer) {
    if uintptr(a) > uintptr(b) {
        a, b = b, a
    }
    v[visit{a, b}] = struct{}{}
}

func compare[T any](a *T, b *T, v visitor, cmp func(*T, *T, visitor) bool) bool {
    if a == nil && b == nil { return true }
    if (a == nil || b == nil) && (a != nil || b != nil) { return false }
    if v.seen(unsafe.Pointer(a), unsafe.Pointer(b)) { return true }
    v.mark(unsafe.Pointer(a), unsafe.Pointer(b))
    return cmp(a, b, v)
}

func ListsEqual[X comparable](as *List[X], bs *List[X]) bool {
    // ListsEqual is a wrapper for listsEqual,
    // which is generated using morph.
    return listsEqual(as, bs, visitor(make(map[visit]struct{})))
}

func listsEqual[X comparable](x *List[X], y *List[X], v visitor) bool {
    // this was copied from the output generated by morph

    // x.Value == y.Value
    _cmp0 := (x.Value == y.Value)

    // x.Next == y.Next
    _cmp1 := bool(compare(x.Next, y.Next, v, listsEqual[X]))

    return (_cmp0 && _cmp1)
}

func ExampleStruct_Comparer_listWithCycles() {
    // this example is explained in a tutorial

    // see above for required machinery

    source := `
package example

type List[X comparable] struct {
    Value X
    Next *List[X]
}
`
    list := must(morph.ParseStruct("test.go", source, "List"))

    sig := `listsEqual[X comparable](x *$[X], y *$[X], v visitor)`
    list.Fields[1].Comparer = "compare($a.$, $b.$, v, listsEqual[X])"

    fmt.Println(must(list.Comparer(sig)))

    {
        assert := func(x bool) {
            if !x { panic("assertion failed") }
        }

        var a1, b1, c1 List[string]
        var a2, b2, c2 List[string]
        a1.Value = "a"; a2.Value = "a"
        b1.Value = "b"; b2.Value = "b"
        c1.Value = "c"; c2.Value = "c"
        a1.Next = &b1;  a2.Next = &b2;
        b1.Next = &c2;  b2.Next = &c2;
        assert(ListsEqual(&a1, &a2))
        assert(ListsEqual(&a1, &a1))
        assert(!ListsEqual(&a1, &b1))
        assert(!ListsEqual(&a1, &b2))
        c1.Next = &a1; c2.Next = &a2;
        assert(ListsEqual(&a1, &a2))
        assert(ListsEqual(&a1, &a1))
        assert(!ListsEqual(&a1, &b1))
        assert(!ListsEqual(&a1, &b2))
    }

    // Output:
    // // listsEqual returns true if two [List] values are equal.
    // func listsEqual[X comparable](x *List[X], y *List[X], v visitor) bool {
    //	// x.Value == y.Value
    //	_cmp0 := (x.Value == y.Value)
    //
    //	// x.Next == y.Next
    //	_cmp1 := bool(compare(x.Next, y.Next, v, listsEqual[X]))
    //
    //	return (_cmp0 && _cmp1)
    // }
}
