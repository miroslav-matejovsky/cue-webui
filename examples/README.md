# Examples

Each example is a standalone Go package with an embedded CUE schema. Run any example from the repository root:

```bash
go run ./examples/basic
go run ./examples/nested-tabs
go run ./examples/platform-stack
```

All examples start a server on `http://localhost:8080`.

## Included examples

### `basic`

The original demo moved out of the module root. It shows the core flow:

- scalar fields
- select widgets from CUE disjunctions
- number bounds mapped to HTML validation
- top-level grouped sections

### `nested-tabs`

Demonstrates deeper nesting with `UI_Navigation: tabs` at multiple levels. Use this example to verify the CSS-only tabbed UI for complex configuration trees.

### `platform-stack`

Demonstrates a denser real-world schema with:

- radio widgets
- textarea overrides
- regex validation
- defaults
- readonly and hidden fields
- multi-column layout hints
