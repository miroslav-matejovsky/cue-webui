package webui

import (
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
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
		if got := CueTypeToInputType(tt.kind); got != tt.want {
			t.Errorf("CueTypeToInputType(%v) = %q, want %q", tt.kind, got, tt.want)
		}
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
		if got := TitleCase(tt.in); got != tt.want {
			t.Errorf("TitleCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestHasStructFields(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("with struct fields", func(t *testing.T) {
		val := ctx.CompileString(`{ sub: { x: int } }`)
		if !HasStructFields(val) {
			t.Error("HasStructFields = false, want true")
		}
	})

	t.Run("without struct fields", func(t *testing.T) {
		val := ctx.CompileString(`{ x: int; y: string }`)
		if HasStructFields(val) {
			t.Error("HasStructFields = true, want false")
		}
	})
}

func TestParseSection_ScalarFields(t *testing.T) {
	src := `
// UI_Label: Server Address
// UI_Help: Hostname or IP
// UI_Placeholder: e.g. 0.0.0.0
address: string

// UI_Help: TCP port number
// UI_Min: 1
// UI_Max: 65535
port: int
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}
	hints := ParseUIHints(val)
	section := ParseSection("connection", val, "", hints)

	if section.Label != "Connection" {
		t.Errorf("Label = %q, want %q", section.Label, "Connection")
	}
	if section.Columns != 2 {
		t.Errorf("Columns = %d, want 2 (default)", section.Columns)
	}
	if len(section.Fields) != 2 {
		t.Fatalf("Fields count = %d, want 2", len(section.Fields))
	}

	addr := section.Fields[0]
	if addr.Name != "address" {
		t.Errorf("Field[0].Name = %q, want %q", addr.Name, "address")
	}
	if addr.Label != "Server Address" {
		t.Errorf("Field[0].Label = %q, want %q", addr.Label, "Server Address")
	}
	if addr.Help != "Hostname or IP" {
		t.Errorf("Field[0].Help = %q, want %q", addr.Help, "Hostname or IP")
	}
	if addr.Placeholder != "e.g. 0.0.0.0" {
		t.Errorf("Field[0].Placeholder = %q, want %q", addr.Placeholder, "e.g. 0.0.0.0")
	}
	if addr.InputType != "text" {
		t.Errorf("Field[0].InputType = %q, want %q", addr.InputType, "text")
	}
	if addr.Widget != "input" {
		t.Errorf("Field[0].Widget = %q, want %q", addr.Widget, "input")
	}

	port := section.Fields[1]
	if port.Name != "port" {
		t.Errorf("Field[1].Name = %q, want %q", port.Name, "port")
	}
	if port.InputType != "number" {
		t.Errorf("Field[1].InputType = %q, want %q", port.InputType, "number")
	}
	if port.Min != "1" {
		t.Errorf("Field[1].Min = %q, want %q", port.Min, "1")
	}
	if port.Max != "65535" {
		t.Errorf("Field[1].Max = %q, want %q", port.Max, "65535")
	}
}

func TestParseSection_WidgetInference(t *testing.T) {
	src := `
// UI_Options: http, https
protocol: string

enabled: bool
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("test", val, "", UIHints{})

	if len(section.Fields) != 2 {
		t.Fatalf("Fields count = %d, want 2", len(section.Fields))
	}

	// Options present → select widget
	proto := findField(section.Fields, "protocol")
	if proto == nil {
		t.Fatal("field 'protocol' not found")
	}
	if proto.Widget != "select" {
		t.Errorf("protocol Widget = %q, want %q", proto.Widget, "select")
	}

	// Bool → checkbox widget
	enabled := findField(section.Fields, "enabled")
	if enabled == nil {
		t.Fatal("field 'enabled' not found")
	}
	if enabled.Widget != "checkbox" {
		t.Errorf("enabled Widget = %q, want %q", enabled.Widget, "checkbox")
	}
}

func TestParseSection_WidgetOverride(t *testing.T) {
	src := `
// UI_Widget: textarea
notes: string
`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("test", val, "", UIHints{})

	if len(section.Fields) != 1 {
		t.Fatalf("Fields count = %d, want 1", len(section.Fields))
	}
	if section.Fields[0].Widget != "textarea" {
		t.Errorf("Widget = %q, want %q", section.Fields[0].Widget, "textarea")
	}
}

func TestParseSection_PathPrefix(t *testing.T) {
	src := `name: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	section := ParseSection("sub", val, "parent", UIHints{})

	if len(section.Fields) != 1 {
		t.Fatalf("Fields count = %d, want 1", len(section.Fields))
	}
	if section.Fields[0].Path != "parent.name" {
		t.Errorf("Path = %q, want %q", section.Fields[0].Path, "parent.name")
	}
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

	if len(section.Sections) != 1 {
		t.Fatalf("Sub-sections count = %d, want 1", len(section.Sections))
	}
	sub := section.Sections[0]
	if sub.Name != "outer" {
		t.Errorf("Sub-section Name = %q, want %q", sub.Name, "outer")
	}
	if len(sub.Fields) != 1 {
		t.Fatalf("Sub-section Fields count = %d, want 1", len(sub.Fields))
	}
	if sub.Fields[0].Path != "outer.inner" {
		t.Errorf("Sub-section Field Path = %q, want %q", sub.Fields[0].Path, "outer.inner")
	}
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

	if len(section.Fields) != 3 {
		t.Fatalf("Fields count = %d, want 3", len(section.Fields))
	}
	want := []string{"a", "b", "c"}
	for i, f := range section.Fields {
		if f.Name != want[i] {
			t.Errorf("Fields[%d].Name = %q, want %q", i, f.Name, want[i])
		}
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
		if f.Default != want {
			t.Errorf("Field %q Default = %q, want %q", f.Name, f.Default, want)
		}
	}
}

func TestParseSection_SectionHints(t *testing.T) {
	src := `x: string`
	ctx := cuecontext.New()
	val := ctx.CompileString(src)
	hints := UIHints{Label: "Custom Label", Help: "Custom help", Columns: 4}
	section := ParseSection("test", val, "", hints)

	if section.Label != "Custom Label" {
		t.Errorf("Label = %q, want %q", section.Label, "Custom Label")
	}
	if section.Help != "Custom help" {
		t.Errorf("Help = %q, want %q", section.Help, "Custom help")
	}
	if section.Columns != 4 {
		t.Errorf("Columns = %d, want 4", section.Columns)
	}
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
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	fd, err := BuildFormData(val)
	if err != nil {
		t.Fatalf("BuildFormData error: %v", err)
	}

	if fd.Title != "My App" {
		t.Errorf("Title = %q, want %q", fd.Title, "My App")
	}
	if len(fd.Sections) != 1 {
		t.Fatalf("Sections count = %d, want 1", len(fd.Sections))
	}
	if fd.Sections[0].Name != "db" {
		t.Errorf("Section[0].Name = %q, want %q", fd.Sections[0].Name, "db")
	}
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
	if val.Err() != nil {
		t.Fatalf("compile error: %v", val.Err())
	}

	fd, err := BuildFormData(val)
	if err != nil {
		t.Fatalf("BuildFormData error: %v", err)
	}

	if fd.Title != "Configuration" {
		t.Errorf("Title = %q, want %q", fd.Title, "Configuration")
	}
	if len(fd.Sections) != 2 {
		t.Fatalf("Sections count = %d, want 2", len(fd.Sections))
	}
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
	if err != nil {
		t.Fatalf("BuildFormData error: %v", err)
	}

	// Should have "General" section prepended + "db" sub-section
	if len(fd.Sections) != 2 {
		t.Fatalf("Sections count = %d, want 2", len(fd.Sections))
	}
	if fd.Sections[0].Label != "General" {
		t.Errorf("Sections[0].Label = %q, want %q", fd.Sections[0].Label, "General")
	}
	if fd.Sections[1].Name != "db" {
		t.Errorf("Sections[1].Name = %q, want %q", fd.Sections[1].Name, "db")
	}
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
	if err != nil {
		t.Fatalf("BuildFormData error: %v", err)
	}

	// #Simple has no struct sub-fields, so it falls back to rendering all defs.
	// Single fallback root gets unwrapped — title comes from the definition name.
	if fd.Title != "Simple" {
		t.Errorf("Title = %q, want %q", fd.Title, "Simple")
	}
	if len(fd.Sections) != 1 {
		t.Fatalf("Sections count = %d, want 1", len(fd.Sections))
	}
}

func TestBuildFormData_InvalidCUE(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`invalid:::cue`)
	_, err := BuildFormData(val)
	if err == nil {
		t.Error("BuildFormData should return error for invalid CUE")
	}
}

func TestBuildFormData_NoDefs(t *testing.T) {
	ctx := cuecontext.New()
	val := ctx.CompileString(`42`)
	_, err := BuildFormData(val)
	if err == nil {
		t.Error("BuildFormData should return error when no definitions found")
	}
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
