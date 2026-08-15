package render_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/RomanAgaltsev/keel/v2"
	"github.com/RomanAgaltsev/keel/v2/internal/answers"
	"github.com/RomanAgaltsev/keel/v2/internal/module"
	"github.com/RomanAgaltsev/keel/v2/internal/render"
)

func licenseAnswers(license string) answers.Answers {
	return answers.Answers{
		"repo_name": "x", "description": "d", "module_path": "github.com/x/x",
		"author_name": "Ada Lovelace", "year": 2026, "license": license,
	}
}

func TestLicenseMIT(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"license"}, licenseAnswers("MIT"))
	require.NoError(t, err)
	require.Contains(t, plan.Files, "LICENSE")
	require.Contains(t, plan.Files["LICENSE"], "MIT License")
	require.Contains(t, plan.Files["LICENSE"], "Copyright (c) 2026 Ada Lovelace")
}

func TestLicenseApache(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"license"}, licenseAnswers("Apache-2.0"))
	require.NoError(t, err)
	require.Contains(t, plan.Files["LICENSE"], "Apache License")
	require.Contains(t, plan.Files["LICENSE"], "Version 2.0, January 2004")
}

func TestLicenseBSD3(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"license"}, licenseAnswers("BSD-3-Clause"))
	require.NoError(t, err)
	require.Contains(t, plan.Files["LICENSE"], "Redistribution and use in source and binary forms")
}

func TestLicenseNoneEmitsNothing(t *testing.T) {
	l := module.NewFSLoader(keel.BuiltinFS)
	plan, err := render.BuildRecipe(l, []string{"license"}, licenseAnswers("none"))
	require.NoError(t, err)
	require.NotContains(t, plan.Files, "LICENSE")
}
