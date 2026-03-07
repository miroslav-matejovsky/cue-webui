package webui

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestFormTemplate_NotEmpty(t *testing.T) {
	tmpl := FormTemplate()
	if tmpl == "" {
		t.Fatal("FormTemplate() returned empty string")
	}
	if !strings.Contains(tmpl, `{{define "form"}}`) {
		t.Error("template missing form definition")
	}
	if !strings.Contains(tmpl, `{{define "result"}}`) {
		t.Error("template missing result definition")
	}
	if !strings.Contains(tmpl, `{{define "section"}}`) {
		t.Error("template missing section definition")
	}
}

func TestStyleCSS_NotEmpty(t *testing.T) {
	css := StyleCSS()
	if css == "" {
		t.Fatal("StyleCSS() returned empty string")
	}
	if !strings.Contains(css, ".container") {
		t.Error("CSS missing .container rule")
	}
}

func TestParseFormTemplate(t *testing.T) {
	tmpl, err := ParseFormTemplate()
	if err != nil {
		t.Fatalf("ParseFormTemplate() error: %v", err)
	}
	if tmpl == nil {
		t.Fatal("ParseFormTemplate() returned nil")
	}
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
	if err != nil {
		t.Fatalf("NewHandler error: %v", err)
	}
	return h
}

func TestNewHandler_FormPage(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET / status = %d, want %d", rec.Code, http.StatusOK)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Test Config") {
		t.Error("response body missing title")
	}
	if !strings.Contains(body, "server.host") {
		t.Error("response body missing field path server.host")
	}
	if !strings.Contains(body, "server.port") {
		t.Error("response body missing field path server.port")
	}
}

func TestNewHandler_NotFound(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /nonexistent status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestNewHandler_CSS(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /static/style.css status = %d, want %d", rec.Code, http.StatusOK)
	}
	ct := rec.Header().Get("Content-Type")
	if !strings.HasPrefix(ct, "text/css") {
		t.Errorf("Content-Type = %q, want text/css", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, ".container") {
		t.Error("CSS response missing .container rule")
	}
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

	if rec.Code != http.StatusOK {
		t.Fatalf("POST /submit status = %d, want %d", rec.Code, http.StatusOK)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "localhost") {
		t.Error("result page missing submitted value 'localhost'")
	}
	if !strings.Contains(body, "8080") {
		t.Error("result page missing submitted value '8080'")
	}
	if !strings.Contains(body, "server.host") {
		t.Error("result page missing field key 'server.host'")
	}
}

func TestNewHandler_SubmitGetRedirects(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/submit", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Errorf("GET /submit status = %d, want %d", rec.Code, http.StatusSeeOther)
	}
	loc := rec.Header().Get("Location")
	if loc != "/" {
		t.Errorf("Location = %q, want %q", loc, "/")
	}
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
	if !strings.Contains(body, "<select") {
		t.Error("response missing <select> element")
	}
	if !strings.Contains(body, "http") || !strings.Contains(body, "https") {
		t.Error("response missing select options")
	}
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

	body := rec.Body.String()
	if !strings.Contains(body, `type="checkbox"`) {
		t.Error("response missing checkbox input")
	}
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

	body := rec.Body.String()
	if !strings.Contains(body, "<textarea") {
		t.Error("response missing <textarea> element")
	}
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
	if !strings.Contains(body, `type="radio"`) {
		t.Error("response missing radio input")
	}
	for _, opt := range []string{"debug", "info", "error"} {
		if !strings.Contains(body, opt) {
			t.Errorf("response missing radio option %q", opt)
		}
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
	// The hidden field's path should not appear as an input name
	if strings.Contains(body, `name="secret"`) {
		t.Error("hidden field 'secret' should not be rendered")
	}
	if !strings.Contains(body, `name="visible"`) {
		t.Error("visible field should be rendered")
	}
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
	if aIdx == -1 || zIdx == -1 {
		t.Fatal("result page missing submitted fields")
	}
	if aIdx > zIdx {
		t.Error("result fields not sorted alphabetically")
	}
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
	if !strings.Contains(body, "Outer") {
		t.Error("missing outer section label")
	}
	if !strings.Contains(body, "Inner") {
		t.Error("missing inner section label")
	}
	if !strings.Contains(body, "outer.inner.val") {
		t.Error("missing nested field path")
	}
}

func TestNewHandler_CSSContent(t *testing.T) {
	handler := mustNewHandler(t, sampleFormData())
	req := httptest.NewRequest(http.MethodGet, "/static/style.css", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body, _ := io.ReadAll(rec.Result().Body)
	css := StyleCSS()
	if string(body) != css {
		t.Error("served CSS does not match embedded CSS")
	}
}
