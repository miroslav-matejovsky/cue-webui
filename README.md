# CUE WebUI

Automatically generates an HTML form UI from a [CUE](https://cuelang.org/) schema. Define your configuration as CUE definitions with optional UI hints in doc comments, and get a fully functional web form with validation, sections, and multiple widget types.

The module root is library-focused. Example CUE schemas live under `examples/` and can be served with `go run ./cmd <schema.cue>`.

## Features

- **Schema-driven forms** — CUE definitions are introspected at startup; fields, types, and constraints become form inputs automatically.
- **Native CUE validation** — Disjunctions (`"a" | "b"`) render as `<select>` dropdowns, bound constraints (`>=1 & <=65535`) become HTML `min`/`max` attributes, and `=~` regex constraints become `pattern` attributes.
- **UI hints via doc comments** — Control labels, help text, widget types, layout, visibility, ordering, and more with `// UI_*` directives.
- **Nested sections and tabs** — Struct fields become recursive sections, and deeply nested groups can switch to CSS-only tabs.
- **Default values** — CUE defaults (`*"value"`) pre-populate form fields.
- **No JavaScript required** — Pure server-rendered HTML with embedded CSS.

## Quick Start

Point the `cmd` binary at any CUE schema file to get an instant web form:

```bash
go run ./cmd <schema.cue> <config.json>
```

An optional `-addr` flag sets the listen address (default `localhost:8080`):

```bash
go run ./cmd -addr 0.0.0.0:9090 myschema.cue config.json
```

Open [http://localhost:8080](http://localhost:8080) to see the generated form.

Try the bundled example schemas:

```bash
go run ./cmd examples/basic/schema.cue config.json
go run ./cmd examples/nested-tabs/schema.cue config.json
go run ./cmd examples/platform-stack/schema.cue config.json
```

See [examples/README.md](examples/README.md) for the catalog.

## Library Usage

If you want to embed your own schema in an application, the flow is:

1. Compile the CUE schema with `cuecontext.New().CompileString(...)`.
2. Convert it to `webform.FormData` with `webform.BuildFormData(...)`.
3. Serve the generated handler from `webui.NewHandler(formData, cueSchema, configPath)`.

## Schema Example

```cue
#Connection: {
    // UI_Label: Server Address
    // UI_Help: Hostname or IP address to listen on
    address: string

    // UI_Help: TCP port number (1-65535)
    port: int & >=1 & <=65535

    // UI_Help: Network protocol to use
    protocol: "http" | "https" | "tcp" | "udp"
}

// UI_Label: Server Configuration
#Configuration: {
    // UI_Help: Network and protocol settings
    // UI_Columns: 3
    connection: #Connection
}
```

Native CUE features used for form behavior:

| CUE Constraint      | Form Effect                                |
| ------------------- | ------------------------------------------ |
| `"a" \| "b" \| "c"` | `<select>` dropdown with options           |
| `>=1 & <=65535`     | `min` and `max` on `<input type="number">` |
| `=~"^[a-z]+$"`      | `pattern` attribute on `<input>`           |
| `*"default"`        | Pre-populated field value                  |
| `bool`              | Checkbox widget                            |
| `int`, `float`      | `<input type="number">`                    |

## UI Hints

Place `// UI_Key: value` directives in CUE doc comments to customize form rendering:

| Hint            | Description                                                          |
| --------------- | -------------------------------------------------------------------- |
| `UI_Label`      | Custom display label (default: title-cased field name)               |
| `UI_Help`       | Help text shown below the input                                      |
| `UI_Widget`     | Widget override: `input`, `select`, `textarea`, `radio`, `checkbox`  |
| `UI_Hidden`     | Hide field from UI (`true`/`false`)                                  |
| `UI_Readonly`   | Make field read-only (`true`/`false`)                                |
| `UI_Order`      | Display order within section (integer, lower first)                  |
| `UI_Columns`    | Grid columns for a section (default: 2)                              |
| `UI_Colspan`    | Number of grid columns a field spans                                 |
| `UI_Navigation` | Child section layout mode. Set to `tabs` for CSS-only tab navigation |

Use `UI_Navigation: tabs` on any struct or definition that contains sub-sections when you want deeper configuration trees to render as tabs instead of a long stack of nested fieldsets.

```cue
#TLS: {
  certFile: string
  keyFile:  string
}

#Auth: {
  mode: "none" | "mtls"
  tls:  #TLS
}

#Configuration: {
  // UI_Navigation: tabs
  server: {
    host: string
  }
  auth: #Auth
}
```

## Project Structure

```text
cmd/
  main.go                    # Standalone binary entry point
examples/                    # Example CUE schemas
  basic/                     # Original starter example
  nested-tabs/               # Deeply nested tabbed configuration example
  platform-stack/            # Denser real-world deployment schema example
internal/
  app/
    main.go                  # CLI flag parsing and server startup
  config/
    config.go                # Load/save flat key-value maps to/from JSON; CUE validation
  webui/
    server.go                # HTTP handler (form page, CSS, submit endpoint)
    values.go                # Stored value hydration and submission merging helpers
    templates/
      form.html              # Go HTML template (form + result views)
    static/
      style.css              # Embedded stylesheet
    webform/
      form.go                # Form/section/field builder from CUE values
      hints.go               # UI hint parsing and CUE constraint extraction
```

## Examples

`examples/basic/schema.cue` is the original simple demo. Run it with `go run ./cmd examples/basic/schema.cue config.json`.

`examples/nested-tabs/schema.cue` shows deeply nested configuration with repeated `UI_Navigation: tabs` hints.

`examples/platform-stack/schema.cue` shows a larger operations-style schema with regex validation, defaults, readonly fields, hidden fields, radios, textarea overrides, and multi-column sections.

## Running Tests

```bash
go test ./...
```
