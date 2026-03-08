package webui

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/miroslav-matejovsky/cue-webui/internal/webui/webform"
	"github.com/stretchr/testify/require"
)

func TestFormTemplate_NotEmpty(t *testing.T) {
	tmpl := FormTemplate()
	require.NotEmpty(t, tmpl, "FormTemplate() returned empty string")
	require.Contains(t, tmpl, `{{define "form"}}`, "template missing form definition")
	require.Contains(t, tmpl, `{{define "result"}}`, "template missing result definition")
	require.Contains(t, tmpl, `{{define "section"}}`, "template missing section definition")
}

func TestStyleCSS_NotEmpty(t *testing.T) {
	css := StyleCSS()
	require.NotEmpty(t, css, "StyleCSS() returned empty string")
	require.Contains(t, css, ".container", "CSS missing .container rule")
}

func TestParseFormTemplate(t *testing.T) {
	tmpl, err := ParseFormTemplate()
	require.NoError(t, err)
	require.NotNil(t, tmpl)
}

// sampleCUESchema returns a compiled CUE value for a simple schema with server.host (string) and server.port (int).
func sampleCUESchema() cue.Value {
	ctx := cuecontext.New()
	return ctx.CompileString(`{ server: { host: string, port: int & >=1 & <=65535 } }`)
}

// definitionCUESchema returns a CUE schema that uses definitions, as real schemas loaded
// from .cue files do. The root definition is #Configuration which references #Connection.
func definitionCUESchema() cue.Value {
	ctx := cuecontext.New()
	return ctx.CompileString("#Connection: { host: string, port: int & >=1 & <=65535 }\n#Configuration: { connection: #Connection }\n")
}

// permissiveCUESchema returns a CUE value that accepts any structure (for tests that don't need validation).
func permissiveCUESchema() cue.Value {
	ctx := cuecontext.New()
	return ctx.CompileString(`{...}`)
}

func sampleFormData() webform.FormData {
	return webform.FormData{
		Title: "Test Config",
		Sections: []webform.Section{
			{
				Name:    "server",
				Label:   "Server",
				Columns: 2,
				Fields: []webform.Field{
					{Name: "host", Path: "server.host", Type: "string", Label: "Host", InputType: "text", Widget: "input"},
					{Name: "port", Path: "server.port", Type: "int", Label: "Port", InputType: "number", Widget: "input", Min: "1", Max: "65535"},
				},
			},
		},
	}
}

func mustNewHandler(t *testing.T, fd webform.FormData, schema cue.Value, configPath string) http.Handler {
	t.Helper()
	h, err := NewHandler(fd, schema, configPath)
	require.NoError(t, err)
	return h
}

func tempConfigPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "config.json")
}

func TestNewHandler_FormPage(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "text/html"), "Content-Type should be text/html")
	body := rec.Body.String()
	require.Contains(t, body, "Test Config", "response body missing title")
	require.Contains(t, body, "server.host", "response body missing field path server.host")
	require.Contains(t, body, "server.port", "response body missing field path server.port")
}

func TestNewHandler_NotFound(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNewHandler_CSS(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css"), "Content-Type should be text/css")
	require.Contains(t, rec.Body.String(), ".container", "CSS response missing .container rule")
}

func TestNewHandler_SubmitPost(t *testing.T) {
	configPath := tempConfigPath(t)
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), configPath)
	form := url.Values{}
	form.Set("server.host", "localhost")
	form.Set("server.port", "8080")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/", rec.Header().Get("Location"))

	// Verify JSON file was written
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	server := parsed["server"].(map[string]any)
	require.Equal(t, "localhost", server["host"])
	require.Equal(t, float64(8080), server["port"])
}

func TestNewHandler_SubmitGetRedirects(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/submit", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/", rec.Header().Get("Location"))
}

func TestNewHandler_FormRenders_SelectWidget(t *testing.T) {
	fd := webform.FormData{
		Title: "Select Test",
		Sections: []webform.Section{{
			Name: "net", Label: "Network", Columns: 2,
			Fields: []webform.Field{
				{Name: "protocol", Path: "protocol", Type: "string", Label: "Protocol", Widget: "select", Options: []string{"http", "https"}},
			},
		}},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, "<select", "response missing <select> element")
	require.Contains(t, body, "http", "response missing select option 'http'")
	require.Contains(t, body, "https", "response missing select option 'https'")
}

func TestNewHandler_FormRenders_CheckboxWidget(t *testing.T) {
	fd := webform.FormData{
		Title: "Checkbox Test",
		Sections: []webform.Section{{
			Name: "flags", Label: "Flags", Columns: 1,
			Fields: []webform.Field{
				{Name: "enabled", Path: "enabled", Type: "bool", Label: "Enabled", Widget: "checkbox"},
			},
		}},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), `type="checkbox"`, "response missing checkbox input")
}

func TestNewHandler_LoadsValuesFromConfigFile(t *testing.T) {
	fd := webform.FormData{
		Title: "Stored Values Test",
		Sections: []webform.Section{{
			Name: "server", Label: "Server", Columns: 2,
			Fields: []webform.Field{
				{Name: "host", Path: "server.host", Type: "string", Label: "Host", Widget: "input", InputType: "text"},
				{Name: "protocol", Path: "server.protocol", Type: "string", Label: "Protocol", Widget: "select", Options: []string{"http", "https"}},
				{Name: "enabled", Path: "server.enabled", Type: "bool", Label: "Enabled", Widget: "checkbox"},
			},
		}},
	}
	configPath := tempConfigPath(t)
	configJSON := `{"server":{"host":"stored.example","protocol":"https","enabled":true}}`
	require.NoError(t, os.WriteFile(configPath, []byte(configJSON), 0644))

	handler := mustNewHandler(t, fd, permissiveCUESchema(), configPath)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, `value="stored.example"`, "response missing stored input value")
	require.Contains(t, body, `value="https"  selected`, "response missing stored select value")
	require.Contains(t, body, `name="server.enabled" value="true" checked`, "response missing checked checkbox")
}

func TestNewHandler_FormRenders_TextareaWidget(t *testing.T) {
	fd := webform.FormData{
		Title: "Textarea Test",
		Sections: []webform.Section{{
			Name: "content", Label: "Content", Columns: 1,
			Fields: []webform.Field{
				{Name: "notes", Path: "notes", Type: "string", Label: "Notes", Widget: "textarea"},
			},
		}},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), "<textarea", "response missing <textarea> element")
}

func TestNewHandler_FormRenders_RadioWidget(t *testing.T) {
	fd := webform.FormData{
		Title: "Radio Test",
		Sections: []webform.Section{{
			Name: "level", Label: "Level", Columns: 1,
			Fields: []webform.Field{
				{Name: "log_level", Path: "log_level", Type: "string", Label: "Log Level", Widget: "radio", Options: []string{"debug", "info", "error"}},
			},
		}},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, `type="radio"`, "response missing radio input")
	for _, opt := range []string{"debug", "info", "error"} {
		require.Contains(t, body, opt, "response missing radio option %q", opt)
	}
}

func TestNewHandler_FormRenders_HiddenField(t *testing.T) {
	fd := webform.FormData{
		Title: "Hidden Test",
		Sections: []webform.Section{{
			Name: "misc", Label: "Misc", Columns: 1,
			Fields: []webform.Field{
				{Name: "secret", Path: "secret", Type: "string", Label: "Secret", Widget: "input", Hidden: true},
				{Name: "visible", Path: "visible", Type: "string", Label: "Visible", Widget: "input"},
			},
		}},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.NotContains(t, body, `name="secret"`, "hidden field 'secret' should not be rendered")
	require.Contains(t, body, `name="visible"`, "visible field should be rendered")
}

func TestNewHandler_SubmitSavesToConfigFile(t *testing.T) {
	fd := webform.FormData{
		Title: "Persist Test",
		Sections: []webform.Section{{
			Name: "server", Label: "Server", Columns: 2,
			Fields: []webform.Field{
				{Name: "host", Path: "server.host", Type: "string", Label: "Host", Widget: "input", InputType: "text"},
				{Name: "enabled", Path: "server.enabled", Type: "bool", Label: "Enabled", Widget: "checkbox"},
				{Name: "protocol", Path: "server.protocol", Type: "string", Label: "Protocol", Widget: "select", Options: []string{"http", "https"}, Readonly: true},
			},
		}},
	}
	configPath := tempConfigPath(t)
	// Pre-populate config file with existing values
	initialJSON := `{"server":{"enabled":true,"protocol":"https"}}`
	require.NoError(t, os.WriteFile(configPath, []byte(initialJSON), 0644))

	schema := permissiveCUESchema()
	handler := mustNewHandler(t, fd, schema, configPath)

	form := url.Values{}
	form.Set("server.host", "api.internal")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)

	// Verify JSON file content
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var parsed map[string]any
	require.NoError(t, json.Unmarshal(data, &parsed))
	server := parsed["server"].(map[string]any)
	require.Equal(t, "api.internal", server["host"])
	require.Equal(t, false, server["enabled"])
	require.Equal(t, "https", server["protocol"])
}

func TestNewHandler_SubmitValidationError(t *testing.T) {
	fd := webform.FormData{
		Title: "Validation Test",
		Sections: []webform.Section{{
			Name: "server", Label: "Server", Columns: 2,
			Fields: []webform.Field{
				{Name: "host", Path: "server.host", Type: "string", Label: "Host", Widget: "input", InputType: "text"},
				{Name: "port", Path: "server.port", Type: "int", Label: "Port", Widget: "input", InputType: "number"},
			},
		}},
	}
	schema := sampleCUESchema()
	handler := mustNewHandler(t, fd, schema, tempConfigPath(t))

	form := url.Values{}
	form.Set("server.host", "localhost")
	form.Set("server.port", "99999") // exceeds <=65535

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "error-banner", "response missing error banner")
	require.Contains(t, body, "Validation error", "response missing validation error message")
	// Form should be re-rendered with submitted values
	require.Contains(t, body, `value="localhost"`, "response missing submitted host value")
}

func TestNewHandler_FormRenders_NestedSections(t *testing.T) {
	fd := webform.FormData{
		Title: "Nested Test",
		Sections: []webform.Section{{
			Name: "outer", Label: "Outer", Columns: 2,
			Sections: []webform.Section{{
				Name: "inner", Label: "Inner", Columns: 1,
				Fields: []webform.Field{
					{Name: "val", Path: "outer.inner.val", Type: "string", Label: "Val", Widget: "input", InputType: "text"},
				},
			}},
		}},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, "Outer", "missing outer section label")
	require.Contains(t, body, "Inner", "missing inner section label")
	require.Contains(t, body, "outer.inner.val", "missing nested field path")
}

func TestNewHandler_FormRenders_TabNavigation(t *testing.T) {
	fd := webform.FormData{
		Title:      "Tabbed Test",
		ID:         "root-config",
		Navigation: "tabs",
		Sections: []webform.Section{
			{
				ID:      "network",
				Name:    "network",
				Label:   "Network",
				Columns: 2,
				Fields: []webform.Field{
					{Name: "host", Path: "network.host", Type: "string", Label: "Host", Widget: "input", InputType: "text"},
				},
			},
			{
				ID:      "logging",
				Name:    "logging",
				Label:   "Logging",
				Columns: 1,
				Fields: []webform.Field{
					{Name: "level", Path: "logging.level", Type: "string", Label: "Level", Widget: "select", Options: []string{"debug", "info"}},
				},
			},
		},
	}
	handler := mustNewHandler(t, fd, permissiveCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, `class="tabset"`, "response missing tab container")
	require.Contains(t, body, `name="tabs-root-config"`, "response missing grouped tab toggle name")
	require.Contains(t, body, `for="tab-network"`, "response missing network tab label")
	require.Contains(t, body, `for="tab-logging"`, "response missing logging tab label")
	require.Contains(t, body, "network.host", "response missing first tab field")
	require.Contains(t, body, "logging.level", "response missing second tab field")
}

func TestNewHandler_CSSContent(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Result().Body)
	require.Equal(t, StyleCSS(), string(body), "served CSS does not match embedded CSS")
}

func TestNewHandler_NoConfigFileRendersDefaults(t *testing.T) {
	fd := sampleFormData()
	fd.Sections[0].Fields[0].Default = "default-host"

	handler := mustNewHandler(t, fd, sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `value="default-host"`, "response missing default value")
}

func TestNewHandler_SchemaJSON_ContentType(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/schema.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "application/json"), "Content-Type should be application/json")
}

func TestNewHandler_SchemaJSON_ValidJSON(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/schema.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result), "response body must be valid JSON")
}

func TestNewHandler_SchemaJSON_PlainStruct_ContainsFields(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData(), sampleCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/schema.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	// The schema should describe a "server" object with "host" and "port" properties.
	props, ok := result["properties"].(map[string]any)
	require.True(t, ok, "JSON Schema missing top-level 'properties'")
	serverSchema, ok := props["server"].(map[string]any)
	require.True(t, ok, "JSON Schema missing 'server' property")
	serverProps, ok := serverSchema["properties"].(map[string]any)
	require.True(t, ok, "JSON Schema 'server' missing 'properties'")
	require.Contains(t, serverProps, "host", "server properties missing 'host'")
	require.Contains(t, serverProps, "port", "server properties missing 'port'")
}

func TestNewHandler_SchemaJSON_DefinitionSchema_NotEmpty(t *testing.T) {
	// Real-world schemas compiled from .cue files use definitions (#Name).
	// The /schema.json endpoint must produce a non-trivial schema for them,
	// not just {"$schema":...,"type":"object"}.
	handler := mustNewHandler(t, sampleFormData(), definitionCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/schema.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	// A schema generated from a CUE definition should have 'properties',
	// confirming that the root definition (#Configuration) was used as input.
	require.Contains(t, result, "properties", "JSON Schema from definition schema must have 'properties' (not just empty type:object)")
	props := result["properties"].(map[string]any)
	require.Contains(t, props, "connection", "JSON Schema must expose the 'connection' field from #Configuration")
}

func TestNewHandler_SchemaJSON_DefinitionSchema_ContainsDefs(t *testing.T) {
	// When the root definition references other definitions they must appear in $defs.
	handler := mustNewHandler(t, sampleFormData(), definitionCUESchema(), tempConfigPath(t))
	req := httptest.NewRequest(http.MethodGet, "/schema.json", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var result map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &result))

	defs, ok := result["$defs"].(map[string]any)
	require.True(t, ok, "JSON Schema must contain '$defs' for referenced definitions")
	// #Connection is referenced by #Configuration so it must appear in $defs.
	var found bool
	for key := range defs {
		if strings.Contains(key, "Connection") {
			found = true
			break
		}
	}
	require.True(t, found, "$defs must contain a 'Connection' entry for the nested definition")
}
