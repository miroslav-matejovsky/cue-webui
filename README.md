# CUE WebUI

A developer tool for experimenting with [CUE](https://cuelang.org/) schemas for application configuration. Point it at a `.cue` schema file and get an instant web form that lets you explore the schema interactively, edit values with live reload, and export valid JSON configs — without writing any UI code.

Can also serve as inspiration or a starting point for your own configuration server built on top of CUE.

Example CUE schemas live under `examples/` and can be served with `go run ./cmd -live <schema.cue> <config.json>`.

## Features

- **Instant web form from a CUE schema** — point the tool at any `.cue` file and get a fully functional form immediately; no UI code required.
- **Live reload** — schema and config file changes are detected automatically and the browser refreshes via SSE; tight feedback loop for schema iteration.
- **JSON config export** — form submissions are validated against the CUE schema and written to a JSON file on disk; the output is always schema-valid.
- **Native CUE validation** — disjunctions (`"a" | "b"`) render as `<select>` dropdowns, bound constraints (`>=1 & <=65535`) become HTML `min`/`max` attributes, and `=~` regex constraints become `pattern` attributes.
- **JSON Schema export** — a `/schema.json` endpoint exposes the loaded CUE schema as JSON Schema (Draft 2020-12).
- **UI hints via doc comments** — control labels, help text, widget types, layout, visibility, ordering, and more with `// UI_*` directives.
- **Nested sections and tabs** — struct fields become recursive sections; deeply nested groups can switch to CSS-only tabs.
- **Default values** — CUE defaults (`*"value"`) pre-populate form fields.
- **No JavaScript required** — pure server-rendered HTML with embedded CSS (live reload injects a minimal SSE snippet only when the `-live` flag is set).

## Download

Pre-built binaries for Linux, macOS, and Windows are published with every [GitHub release](../../releases/latest):

| Platform     | Architecture | File                              |
| ------------ | ------------ | --------------------------------- |
| Linux        | x86-64       | `cue-webui-linux-amd64`           |
| Linux        | ARM64        | `cue-webui-linux-arm64`           |
| macOS        | x86-64       | `cue-webui-darwin-amd64`          |
| macOS (M1+)  | ARM64        | `cue-webui-darwin-arm64`          |
| Windows      | x86-64       | `cue-webui-windows-amd64.exe`     |

Download the binary for your platform, make it executable (Linux/macOS: `chmod +x`), and run it directly — no Go toolchain required.

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

### Live Reload

Pass the `-live` flag to enable live reload. The server watches the schema and config files for changes and automatically refreshes the browser via a server-sent events (SSE) endpoint:

```bash
go run ./cmd -live myschema.cue config.json
```

When live reload is active, a small EventSource script is injected into the page. No external tooling or browser extension is needed.

## Library Usage

If you want to embed your own schema in an application, the flow is:

1. Compile the CUE schema with `cuecontext.New().CompileString(...)`.
2. Convert it to `webform.FormData` with `webform.BuildFormData(...)`.
3. Serve the generated handler from `webui.NewHandler(formData, cueSchema, configPath)`.

The handler exposes the following HTTP endpoints:

| Endpoint            | Method | Description                                                   |
| ------------------- | ------ | ------------------------------------------------------------- |
| `/`                 | GET    | HTML form pre-populated with values from `configPath`         |
| `/static/style.css` | GET    | Embedded CSS stylesheet                                       |
| `/schema.json`      | GET    | JSON Schema (Draft 2020-12) generated from the CUE schema     |
| `/submit`           | POST   | Validates form values, writes JSON to `configPath`, redirects |

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
    server.go                # HTTP handler (form page, CSS, schema.json, submit endpoint)
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

```bash
go run ./cmd examples/basic/schema.cue examples/basic/config.json
go run ./cmd examples/nested-tabs/schema.cue examples/nested-tabs/config.json
go run ./cmd examples/platform-stack/schema.cue examples/platform-stack/config.json
```

- **basic** — original simple demo.
- **nested-tabs** — deeply nested configuration with repeated `UI_Navigation: tabs` hints.
- **platform-stack** — larger operations-style schema with regex validation, defaults, readonly fields, hidden fields, radios, textarea overrides, and multi-column sections.

## Running Tests

```bash
go test ./...
```
