package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
	"time"
)

func TestCamelCase(t *testing.T) {
	snake := "this_is_a_test_string"
	camel := "ThisIsATestString"

	badSnake := "this_Is_A_Test_String"
	badCamel := "This_Is_A_Test_String"

	badSnake2 := "This_is_Totally_wrong"
	badCamel2 := "ThisIs_TotallyWrong"
	if toCamelCase(snake) != camel {
		t.Errorf("Expected %s, got %s", camel, toCamelCase(snake))
	}
	if toCamelCase(badSnake) != badCamel {
		t.Errorf("Expected %s, got %s", badCamel, toCamelCase(badSnake))
	}
	if toCamelCase(badSnake2) != badCamel2 {
		t.Errorf("Expected %s, got %s", badCamel2, toCamelCase(badSnake2))
	}

	commonNumeric := "n2s"
	expectCommonNumeric := "N2S"
	if toCamelCase(commonNumeric) != expectCommonNumeric {
		t.Errorf("Expected %s, got %s", expectCommonNumeric, toCamelCase(commonNumeric))
	}
}

func TestZeroValue(t *testing.T) {
	tests := []struct {
		typeName string
		expected any
	}{
		{"int32", 0},
		{"int64", 0},
		{"uint32", 0},
		{"uint64", 0},
		{"float32", 0.0},
		{"float64", 0.0},
		{"bool", false},
		{"string", ""},
		{"[]byte", nil},
		{"[]int32", nil},
		{"map[string]int32", nil},
		{"*MyStruct", nil},
		{"MyStruct", MyStruct{}},
		{"time.Time", time.Time{}},
	}

	for _, test := range tests {
		result := isZeroValue(test.expected)
		if !result {
			t.Errorf("Expected true for type %s, got %v", test.typeName, result)
		}
	}

	nonZeroes := []struct {
		typeName string
		expected any
	}{
		{"[]int64", []int64{}},
		{"map[string]string", map[string]string{}},
		{"*MyStruct", &MyStruct{}},
	}
	for _, test := range nonZeroes {
		result := isZeroValue(test.expected)
		if result {
			t.Errorf("Expected false for type %s, got %v", test.typeName, result)
		}
	}
}

type MyStruct struct {
}

func TestDefaultNodeType(t *testing.T) {
	if defaultNodeType() != "default" {
		t.Fatalf("expected default node type to be default, got %q", defaultNodeType())
	}
}

func TestDBTypesIncludesPostgresSQL(t *testing.T) {
	dbTypes := supportedDBTypes()
	want := map[DBType]bool{
		DBTypeMySQL:       true,
		DBTypeTcaplus:     true,
		DBTypePostgresSQL: true,
	}
	for _, typ := range dbTypes {
		delete(want, typ)
	}
	if len(want) != 0 {
		t.Fatalf("missing db types: %#v", want)
	}
}

func TestFieldJSONClassification(t *testing.T) {
	tests := []struct {
		name string
		f    FieldDesc
		want bool
	}{
		{name: "scalar int64", f: FieldDesc{Type: "int64"}, want: false},
		{name: "scalar string", f: FieldDesc{Type: "string"}, want: false},
		{name: "repeated scalar", f: FieldDesc{Type: "int32", List: true}, want: true},
		{name: "map", f: FieldDesc{Type: "map<string, int32>"}, want: true},
		{name: "message", f: FieldDesc{Type: "PlayerProfile"}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isJSONDBField(tt.f); got != tt.want {
				t.Fatalf("isJSONDBField(%#v) = %v, want %v", tt.f, got, tt.want)
			}
		})
	}
}

func TestContextTemplateIncludesContextHooks(t *testing.T) {
	tmpl := template.Must(template.New("context").Funcs(funcMap).Parse(contextTemplate))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("execute context template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"type Context interface",
		"type BeforeEncodeHook interface",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("context template missing %q in:\n%s", want, out)
		}
	}
}

func TestMethodsTemplateIncludesContextMethods(t *testing.T) {
	tmpl := template.Must(template.New("method").Funcs(funcMap).Parse(methodsTemplate))
	data := struct {
		Package  string
		Messages []MessageDesc
		DBType   string
	}{
		Package: "src",
		DBType:  string(DBTypeMySQL),
		Messages: []MessageDesc{
			{
				Name:      "Player",
				TableName: "player",
				Fields: []FieldDesc{
					{Name: "id", Type: "int64", OrmOptions: &FieldOrmOptions{HasPrimaryKey: true}},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute methods template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"func (m *Player) EncodeFromContext(ctx Context, value proto.Message) error",
		"func (m *Player) DecodeToContext(ctx Context, value proto.Message) error",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("methods template missing %q in:\n%s", want, out)
		}
	}
}

func TestBuildGormTagFromFieldTagIncludesJSON(t *testing.T) {
	tag := buildGormTag(FieldTag{
		Pk:    "primary_key;column:id",
		Index: "index:idx_id;column:id",
		JSON:  "type:json;column:profile",
	})
	want := `gorm:"primary_key;column:id;index:idx_id;column:id;type:json;column:profile"`
	if tag != want {
		t.Fatalf("buildGormTag() = %q, want %q", tag, want)
	}
}
