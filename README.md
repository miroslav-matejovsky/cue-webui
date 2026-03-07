# CUE WebUI

Automatically generates an HTML form UI from a [CUE](https://cuelang.org/) schema. Define your configuration as CUE definitions with optional UI hints in doc comments, and get a fully functional web form with validation, sections, and multiple widget types.

## Features

- **Schema-driven forms** — CUE definitions are introspected at startup; fields, types, and constraints become form inputs automatically.
- **Native CUE validation** — Disjunctions (`"a" | "b"`) render as `<select>` dropdowns, bound constraints (`>=1 & <=65535`) become HTML `min`/`max` attributes, and `=~` regex constraints become `pattern` attributes.
- **UI hints via doc comments** — Control labels, help text, widget types, layout, visibility, ordering, and more with `// UI_*` directives.
- **Nested sections** — Struct fields become collapsible fieldsets with configurable grid columns.
- **Default values** — CUE defaults (`*"value"`) pre-populate form fields.
- **No JavaScript required** — Pure server-rendered HTML with embedded CSS.

## Quick Start

```bash
go run main.go
```

Open [http://localhost:8080](http://localhost:8080) to see the generated form.

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

| CUE Constraint | Form Effect |
|---|---|
| `"a" \| "b" \| "c"` | `<select>` dropdown with options |
| `>=1 & <=65535` | `min` and `max` on `<input type="number">` |
| `=~"^[a-z]+$"` | `pattern` attribute on `<input>` |
| `*"default"` | Pre-populated field value |
| `bool` | Checkbox widget |
| `int`, `float` | `<input type="number">` |

## UI Hints

Place `// UI_Key: value` directives in CUE doc comments to customize form rendering:

| Hint | Description |
|---|---|
| `UI_Label` | Custom display label (default: title-cased field name) |
| `UI_Help` | Help text shown below the input |
| `UI_Widget` | Widget override: `input`, `select`, `textarea`, `radio`, `checkbox` |
| `UI_Hidden` | Hide field from UI (`true`/`false`) |
| `UI_Readonly` | Make field read-only (`true`/`false`) |
| `UI_Order` | Display order within section (integer, lower first) |
| `UI_Columns` | Grid columns for a section (default: 2) |
| `UI_Colspan` | Number of grid columns a field spans |

## Project Structure

```
main.go              # Entry point — compiles schema, starts HTTP server
schema.cue           # CUE schema defining the configuration
webui/
  hints.go           # UI hint parsing and CUE constraint extraction
  form.go            # Form/section/field builder from CUE values
  server.go          # HTTP handler (form page, CSS, submit endpoint)
  templates/
    form.html        # Go HTML template (form + result views)
  static/
    style.css        # Embedded stylesheet
```

## Running Tests

```bash
go test ./webui/ -v
```
