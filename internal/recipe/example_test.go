package recipe_test

import (
	"fmt"
	"log"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/recipe"
)

// ExampleLoad loads a builtin recipe from the embedded filesystem.
func ExampleLoad() {
	rec, err := recipe.Load(keel.BuiltinFS, "go-service")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(rec.Name, rec.Language)
	// Output: go-service go
}
