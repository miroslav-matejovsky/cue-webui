# Examples

Each example is a CUE schema file. Run any of them by passing the path to `cmd`:

```bash
go run ./cmd examples/basic/schema.cue
go run ./cmd examples/nested-tabs/schema.cue
go run ./cmd examples/platform-stack/schema.cue
```

All examples start a server on `http://localhost:8080`. Use `-addr` to change the address:

```bash
go run ./cmd -addr 0.0.0.0:9090 examples/basic/schema.cue
```

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
