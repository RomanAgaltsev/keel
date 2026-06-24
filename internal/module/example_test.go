package module_test

import (
	"fmt"
	"log"

	"github.com/RomanAgaltsev/keel"
	"github.com/RomanAgaltsev/keel/internal/module"
)

// ExampleNewComposite builds a composite over the builtin modules (no externals)
// and loads one module's manifest.
func ExampleNewComposite() {
	comp, err := module.NewComposite(keel.BuiltinFS, nil)
	if err != nil {
		log.Fatal(err)
	}
	m, err := comp.Load("base-layout")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(m.Name)
	// Output: base-layout
}
