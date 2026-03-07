package webui

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/require"
)

func TestCueTypeToInputType(t *testing.T) {
	tests := []struct {
		kind cue.Kind
		want string
	}{
		{cue.IntKind, "number"},
		{cue.FloatKind, "number"},
		{cue.NumberKind, "number"},
		{cue.BoolKind, "checkbox"},
		{cue.StringKind, "text"},
		{cue.BytesKind, "text"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, CueTypeToInputType(tt.kind))
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"hello", "Hello"},
		{"Hello", "Hello"},
		{"address", "Address"},
		{"a", "A"},
	}
	for _, tt := range tests {
		require.Equal(t, tt.want, TitleCase(tt.in))
	}
}

func TestHasStructFields(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("with struct fields", func(t *testing.T) {
		val := ctx.CompileString(`{ sub: { x: int } }`)
		require.True(t, HasStructFields(val))
	})

	t.Run("without struct fields", func(t *testing.T) {
		val := ctx.CompileString(`{ x: int; y: string }`)
		require.False(t, HasStructFields(val))
	})
}

func TestParseSection_ScalarFields(t *testing.T) {
	src := `
// UI_Label: Server Address
// UI_Help: Hostname or IP
address: string

// UI_Help: TCP port number
port: int & >=1 & <=65535
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	require.NoError(t, val.Err())
	hints := ParseUIHints(val)
	section := ParseSection("connection", val, "", hints)

	require.Equal(t, "Connection", section.Label)
	require.Equal(t, 2, section.Columns)
	require.Len(t, section.Fields, 2)

	addr := section.Fields[0]
	require.Equal(t, "address", addr.Name)
	require.Equal(t, "Server Address", addr.Label)
	require.Equal(t, "Hostname or IP", addr.Help)
	require.Equal(t, "text", addr.InputType)
	require.Equal(t, "input", addr.Widget)

	port := section.Fields[1]
	require.Equal(t, "port", port.Name)
	require.Equal(t, "number", port.InputType)
	require.Equal(t, "1", port.Min)
	require.Equal(t, "65535", port.Max)
}

func TestParseSection_WidgetInference(t *testing.T) {
	src := `
protocol: "http" | "https"

enabled: bool
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("test", val, "", UIHints{})

	require.Len(t, section.Fields, 2)

	// Options present → select widget
	proto := findField(section.Fields, "protocol")
	require.NotNil(t, proto, "field 'protocol' not found")
	require.Equal(t, "select", proto.Widget)

	// Bool → checkbox widget
	enabled := findField(section.Fields, "enabled")
	require.NotNil(t, enabled, "field 'enabled' not found")
	require.Equal(t, "checkbox", enabled.Widget)
}

func TestParseSection_WidgetOverride(t *testing.T) {
	src := `
// UI_Widget: textarea
notes: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("test", val, "", UIHints{})

	require.Len(t, section.Fields, 1)
	require.Equal(t, "textarea", section.Fields[0].Widget)
}

func TestParseSection_PathPrefix(t *testing.T) {
	src := `name: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("sub", val, "parent", UIHints{})

	require.Len(t, section.Fields, 1)
	require.Equal(t, "parent.name", section.Fields[0].Path)
}

func TestParseSection_NestedStruct(t *testing.T) {
	src := `
outer: {
	inner: string
}
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("root", val, "", UIHints{})

	require.Len(t, section.Sections, 1)
	sub := section.Sections[0]
	require.Equal(t, "outer", sub.Name)
	require.Len(t, sub.Fields, 1)
	require.Equal(t, "outer.inner", sub.Fields[0].Path)
}

func TestParseSection_FieldOrder(t *testing.T) {
	src := `
// UI_Order: 3
c: string
// UI_Order: 1
a: string
// UI_Order: 2
b: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("test", val, "", UIHints{})

	require.Len(t, section.Fields, 3)
	want := []string{"a", "b", "c"}
	for i, f := range section.Fields {
		require.Equal(t, want[i], f.Name)
	}
}

func TestParseSection_DefaultValues(t *testing.T) {
	src := `
name: string | *"hello"
count: int | *42
ratio: float | *3.14
active: bool | *true
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("test", val, "", UIHints{})

	defaults := map[string]string{
		"name":   "hello",
		"count":  "42",
		"ratio":  "3.14",
		"active": "true",
	}
	for _, f := range section.Fields {
		want, ok := defaults[f.Name]
		if !ok {
			continue
		}
		require.Equal(t, want, f.Default, "field %q default", f.Name)
	}
}

func TestParseSection_SectionHints(t *testing.T) {
	src := `x: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	hints := UIHints{Label: "Custom Label", Help: "Custom help", Columns: 4}
	section := ParseSection("test", val, "", hints)

	require.Equal(t, "Custom Label", section.Label)
	require.Equal(t, "Custom help", section.Help)
	require.Equal(t, 4, section.Columns)
}

func TestBuildFormData_SingleRoot(t *testing.T) {
	src := `
// UI_Label: My App
#Config: {
	// UI_Columns: 3
	db: {
		host: string
		port: int
	}
}
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	require.NoError(t, val.Err())

	fd, err := BuildFormData(val)
	require.NoError(t, err)

	require.Equal(t, "My App", fd.Title)
	require.Len(t, fd.Sections, 1)
	require.Equal(t, "db", fd.Sections[0].Name)
}

func TestBuildFormData_MultipleRoots(t *testing.T) {
	src := `
#Server: {
	conn: {
		host: string
	}
}
#Client: {
	api: {
		url: string
	}
}
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	require.NoError(t, val.Err())

	fd, err := BuildFormData(val)
	require.NoError(t, err)

	require.Equal(t, "Configuration", fd.Title)
	require.Len(t, fd.Sections, 2)
}

func TestBuildFormData_SingleRootWithScalars(t *testing.T) {
	src := `
#Config: {
	name: string
	db: {
		host: string
	}
}
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	fd, err := BuildFormData(val)
	require.NoError(t, err)

	// Should have "General" section prepended + "db" sub-section
	require.Len(t, fd.Sections, 2)
	require.Equal(t, "General", fd.Sections[0].Label)
	require.Equal(t, "db", fd.Sections[1].Name)
}

func TestBuildFormData_NoStructRootsFallback(t *testing.T) {
	src := `
#Simple: {
	name: string
	age: int
}
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	fd, err := BuildFormData(val)
	require.NoError(t, err)

	// #Simple has no struct sub-fields, so it falls back to rendering all defs.
	// Single fallback root gets unwrapped — title comes from the definition name.
	require.Equal(t, "Simple", fd.Title)
	require.Len(t, fd.Sections, 1)
}

func TestBuildFormData_InvalidCUE(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`invalid:::cue`)
	_, err := BuildFormData(val)
	require.Error(t, err, "BuildFormData should return error for invalid CUE")
}

func TestBuildFormData_NoDefs(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`42`)
	_, err := BuildFormData(val)
	require.Error(t, err, "BuildFormData should return error when no definitions found")
}

// findField returns a pointer to the Field with the given name, or nil.
func findField(fields []Field, name string) *Field {
	for i := range fields {
		if fields[i].Name == name {
			return &fields[i]
		}
	}
	return nil
}
