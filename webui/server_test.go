package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

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

func sampleFormData() FormData {
	return FormData{
		Title: "Test Config",
		Sections: []Section{
			{
				Name:    "server",
				Label:   "Server",
				Columns: 2,
				Fields: []Field{
					{Name: "host", Path: "server.host", Label: "Host", InputType: "text", Widget: "input"},
					{Name: "port", Path: "server.port", Label: "Port", InputType: "number", Widget: "input", Min: "1", Max: "65535"},
				},
			},
		},
	}
}

func mustNewHandler(t *testing.T, fd FormData) http.Handler {
	t.Helper()
	h, err := NewHandler(fd)
	require.NoError(t, err)
	return h
}

func TestNewHandler_FormPage(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
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
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Code)
}

func TestNewHandler_CSS(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	require.True(t, strings.HasPrefix(rec.Header().Get("Content-Type"), "text/css"), "Content-Type should be text/css")
	require.Contains(t, rec.Body.String(), ".container", "CSS response missing .container rule")
}

func TestNewHandler_SubmitPost(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	form := url.Values{}
	form.Set("server.host", "localhost")
	form.Set("server.port", "8080")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	require.Contains(t, body, "localhost", "result page missing submitted value 'localhost'")
	require.Contains(t, body, "8080", "result page missing submitted value '8080'")
	require.Contains(t, body, "server.host", "result page missing field key 'server.host'")
}

func TestNewHandler_SubmitGetRedirects(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/submit", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Equal(t, http.StatusSeeOther, rec.Code)
	require.Equal(t, "/", rec.Header().Get("Location"))
}

func TestNewHandler_FormRenders_SelectWidget(t *testing.T) {
	fd := FormData{
		Title: "Select Test",
		Sections: []Section{{
			Name: "net", Label: "Network", Columns: 2,
			Fields: []Field{
				{Name: "protocol", Path: "protocol", Label: "Protocol", Widget: "select", Options: []string{"http", "https"}},
			},
		}},
	}
	handler := mustNewHandler(t, fd)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, "<select", "response missing <select> element")
	require.Contains(t, body, "http", "response missing select option 'http'")
	require.Contains(t, body, "https", "response missing select option 'https'")
}

func TestNewHandler_FormRenders_CheckboxWidget(t *testing.T) {
	fd := FormData{
		Title: "Checkbox Test",
		Sections: []Section{{
			Name: "flags", Label: "Flags", Columns: 1,
			Fields: []Field{
				{Name: "enabled", Path: "enabled", Label: "Enabled", Widget: "checkbox"},
			},
		}},
	}
	handler := mustNewHandler(t, fd)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), `type="checkbox"`, "response missing checkbox input")
}

func TestNewHandler_FormRenders_TextareaWidget(t *testing.T) {
	fd := FormData{
		Title: "Textarea Test",
		Sections: []Section{{
			Name: "content", Label: "Content", Columns: 1,
			Fields: []Field{
				{Name: "notes", Path: "notes", Label: "Notes", Widget: "textarea"},
			},
		}},
	}
	handler := mustNewHandler(t, fd)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	require.Contains(t, rec.Body.String(), "<textarea", "response missing <textarea> element")
}

func TestNewHandler_FormRenders_RadioWidget(t *testing.T) {
	fd := FormData{
		Title: "Radio Test",
		Sections: []Section{{
			Name: "level", Label: "Level", Columns: 1,
			Fields: []Field{
				{Name: "log_level", Path: "log_level", Label: "Log Level", Widget: "radio", Options: []string{"debug", "info", "error"}},
			},
		}},
	}
	handler := mustNewHandler(t, fd)
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
	fd := FormData{
		Title: "Hidden Test",
		Sections: []Section{{
			Name: "misc", Label: "Misc", Columns: 1,
			Fields: []Field{
				{Name: "secret", Path: "secret", Label: "Secret", Widget: "input", Hidden: true},
				{Name: "visible", Path: "visible", Label: "Visible", Widget: "input"},
			},
		}},
	}
	handler := mustNewHandler(t, fd)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.NotContains(t, body, `name="secret"`, "hidden field 'secret' should not be rendered")
	require.Contains(t, body, `name="visible"`, "visible field should be rendered")
}

func TestNewHandler_SubmitResultSorted(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	form := url.Values{}
	form.Set("z_field", "last")
	form.Set("a_field", "first")

	req := httptest.NewRequest(http.MethodPost, "/submit", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	aIdx := strings.Index(body, "a_field")
	zIdx := strings.Index(body, "z_field")
	require.NotEqual(t, -1, aIdx, "result page missing 'a_field'")
	require.NotEqual(t, -1, zIdx, "result page missing 'z_field'")
	require.Less(t, aIdx, zIdx, "result fields not sorted alphabetically")
}

func TestNewHandler_FormRenders_NestedSections(t *testing.T) {
	fd := FormData{
		Title: "Nested Test",
		Sections: []Section{{
			Name: "outer", Label: "Outer", Columns: 2,
			Sections: []Section{{
				Name: "inner", Label: "Inner", Columns: 1,
				Fields: []Field{
					{Name: "val", Path: "outer.inner.val", Label: "Val", Widget: "input", InputType: "text"},
				},
			}},
		}},
	}
	handler := mustNewHandler(t, fd)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	require.Contains(t, body, "Outer", "missing outer section label")
	require.Contains(t, body, "Inner", "missing inner section label")
	require.Contains(t, body, "outer.inner.val", "missing nested field path")
}

func TestNewHandler_FormRenders_TabNavigation(t *testing.T) {
	fd := FormData{
		Title:      "Tabbed Test",
		ID:         "root-config",
		Navigation: "tabs",
		Sections: []Section{
			{
				ID:      "network",
				Name:    "network",
				Label:   "Network",
				Columns: 2,
				Fields: []Field{
					{Name: "host", Path: "network.host", Label: "Host", Widget: "input", InputType: "text"},
				},
			},
			{
				ID:      "logging",
				Name:    "logging",
				Label:   "Logging",
				Columns: 1,
				Fields: []Field{
					{Name: "level", Path: "logging.level", Label: "Level", Widget: "select", Options: []string{"debug", "info"}},
				},
			},
		},
	}
	handler := mustNewHandler(t, fd)
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
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Result().Body)
	require.Equal(t, StyleCSS(), string(body), "served CSS does not match embedded CSS")
}
