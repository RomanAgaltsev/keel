package keel

import "embed"

// BuiltinFS holds keel's embedded built-in modules and recipes.
//
//go:embed modules recipes
var BuiltinFS embed.FS
