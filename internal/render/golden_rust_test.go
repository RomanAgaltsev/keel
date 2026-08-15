package render_test

import (
	"testing"
)

func TestRustServiceGolden(t *testing.T) {
	assertGolden(t, planForRecipe(t, "rust-service"), "rust-service")
}
