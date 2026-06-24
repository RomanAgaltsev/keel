# Example external module: `editorconfig`

A minimal external module demonstrating the authoring surface:

- a `bool` question (`enable_editorconfig`) gating the file via a `when` condition,
- an `int` question (`indent_size`) used inside the template,
- a `.tmpl` file rendered with `text/template` (the `.tmpl` suffix is stripped, so
  `templates/.editorconfig.tmpl` becomes `.editorconfig`).

It is consumed by `../custom-recipe/recipe.yaml` through a relative `dir:` source.
External modules keep `module.yaml` and `templates/` at the module's root.
