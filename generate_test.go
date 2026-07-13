package main

import (
	"bytes"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/wxdqing/protoc-gen-go-orm/options"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
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

func newMessageDescWithOrmDefaults() *MessageDesc {
	return &MessageDesc{
		OrmOptions: MessageOrmOptions{
			NodeType:       OptionalString{Value: defaultNodeType(), Valid: true},
			TableStoreMode: options.TableStoreMode_TABLE_STORE_MODE_PAYLOAD,
		},
	}
}

func TestExplicitNodeTypeParsing(t *testing.T) {
	msgOpts := &descriptorpb.MessageOptions{}
	proto.SetExtension(msgOpts, options.E_Table, true)
	proto.SetExtension(msgOpts, options.E_NodeType, "game")

	desc := newMessageDescWithOrmDefaults()
	applyOrmMessageOptions(desc, msgOpts)

	if !desc.OrmOptions.IsTable {
		t.Fatal("expected IsTable=true")
	}
	if !desc.OrmOptions.NodeType.Valid || desc.OrmOptions.NodeType.Value != "game" {
		t.Fatalf("node type = %#v, want game", desc.OrmOptions.NodeType)
	}
}

func TestTableStoreModeParsing(t *testing.T) {
	msgOpts := &descriptorpb.MessageOptions{}
	proto.SetExtension(msgOpts, options.E_TableStoreMode, options.TableStoreMode_TABLE_STORE_MODE_FIELDS)

	desc := newMessageDescWithOrmDefaults()
	applyOrmMessageOptions(desc, msgOpts)

	if desc.OrmOptions.TableStoreMode != options.TableStoreMode_TABLE_STORE_MODE_FIELDS {
		t.Fatalf("table store mode = %v, want FIELDS", desc.OrmOptions.TableStoreMode)
	}
}

func TestDBTypesIncludesAllDrivers(t *testing.T) {
	dbTypes := supportedDBTypes()
	want := map[DBType]bool{
		DBTypeMySQL:       true,
		DBTypeTcaplus:     true,
		DBTypePostgresSQL: true,
		DBTypeRedis:       true,
		DBTypeMongo:       true,
	}
	for _, typ := range dbTypes {
		delete(want, typ)
	}
	if len(want) != 0 {
		t.Fatalf("missing db types: %#v", want)
	}
}

func TestHasBlobTagSkipsJSONClassification(t *testing.T) {
	f := FieldDesc{
		Type: "bytes",
		List: true,
		Tags: OptionalString{Value: `gorm:"type:blob"`, Valid: true},
	}
	if !hasBlobTag(f) {
		t.Fatal("hasBlobTag = false, want true")
	}
	if isJSONDBField(f) {
		t.Fatal("blob tag field should not be JSON field")
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

func TestShardingKeyExpr(t *testing.T) {
	if got := shardingKeyExpr(FieldDesc{Name: "id", Type: "int64"}); got != "x.Id" {
		t.Fatalf("int64 = %q", got)
	}
	if got := shardingKeyExpr(FieldDesc{Name: "rid", Type: "uint32"}); got != "int64(x.Rid)" {
		t.Fatalf("uint32 = %q", got)
	}
}

func TestContextTemplateIncludesJSONCodec(t *testing.T) {
	codecTmpl := template.Must(template.New("codec").Funcs(funcMap).Parse(jsonCodecTemplate))
	var buf bytes.Buffer
	if err := codecTmpl.Execute(&buf, nil); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"func marshalProtoFieldToJSON",
		"func unmarshalProtoFieldFromJSON",
		"func marshalProtoFieldToWire",
		"func unmarshalProtoFieldFromWire",
		"func unmarshalProtoListWire",
		"func unmarshalScalarProtoList",
		"protowire.AppendVarint",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in context output", want)
		}
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
	msgs := []MessageDesc{
		{
			Name:      "Player",
			TableName: "player",
			OrmOptions: MessageOrmOptions{
				IsTable:        true,
				TableStoreMode: 1,
			},
			Fields: []FieldDesc{
				{Name: "id", Type: "int64", OrmOptions: &FieldOrmOptions{HasPrimaryKey: true}},
			},
		},
	}
	data := struct {
		Package     string
		Messages    []MessageDesc
		AllMessages []MessageDesc
		DBType      string
	}{
		Package:     "src",
		DBType:      string(DBTypeMySQL),
		Messages:    msgs,
		AllMessages: msgs,
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

func TestMetadataTemplateIncludesNodeTableHelpers(t *testing.T) {
	tmpl := template.Must(template.New("metadata").Funcs(funcMap).Parse(metadataTemplate))
	messages := []MessageDesc{
		{
			Name: "Player",
			OrmOptions: MessageOrmOptions{
				IsTable:  true,
				NodeType: OptionalString{Value: "game", Valid: true},
			},
		},
		{
			Name: "Lister",
			OrmOptions: MessageOrmOptions{
				IsTable:  true,
				NodeType: OptionalString{Value: "social", Valid: true},
			},
		},
	}
	dbTypes := supportedDBTypes()
	messagesByDBType := make(map[DBType][]MessageDesc, len(dbTypes))
	for _, dbType := range dbTypes {
		messagesByDBType[dbType] = filterMessagesForDBType(messages, dbType)
	}
	data := struct {
		Version          string
		ProtocVersion    string
		Package          string
		GoPackage        string
		Messages         []MessageDesc
		MessagesByDBType map[DBType][]MessageDesc
		PostgresDBType   DBType
		DirectMessages   map[string][]MessageDesc
		DBTypes          []DBType
		Source           string
		Enums            []EnumDesc
	}{
		Package:          "src",
		Messages:         messages,
		MessagesByDBType: messagesByDBType,
		PostgresDBType:   DBTypePostgresSQL,
		DirectMessages:   map[string][]MessageDesc{},
		DBTypes:          dbTypes,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute metadata template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		`var NodeTables = make(map[string]map[string][]proto.Message)`,
		`func GetAllTables(dbType string) []proto.Message`,
		`func GetNodeTables(dbType string, nodeType string) []proto.Message`,
		`NodeTables["mysql"]["game"]`,
		`NodeTables["pgsql"]["social"]`,
		`Tables["tcaplus"]`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("metadata template missing %q in:\n%s", want, out)
		}
	}
}

func TestMetadataTemplateFiltersMessagesByDBType(t *testing.T) {
	tmpl := template.Must(template.New("metadata").Funcs(funcMap).Parse(metadataTemplate))
	messages := []MessageDesc{
		{
			Name:      "SharedTable",
			TableName: "shared_table",
			OrmOptions: MessageOrmOptions{
				IsTable:  true,
				NodeType: OptionalString{Value: "game", Valid: true},
			},
		},
		{
			Name:      "PgsqlTable",
			TableName: "pgsql_table",
			OrmOptions: MessageOrmOptions{
				IsTable:   true,
				NodeType:  OptionalString{Value: "game", Valid: true},
				DbDrivers: []string{"pgsql"},
			},
		},
		{
			Name:      "RedisTable",
			TableName: "redis_table",
			OrmOptions: MessageOrmOptions{
				IsTable:   true,
				NodeType:  OptionalString{Value: "social", Valid: true},
				DbDrivers: []string{"redis"},
			},
		},
	}
	dbTypes := []DBType{DBTypePostgresSQL, DBTypeRedis}
	data := struct {
		Messages         []MessageDesc
		MessagesByDBType map[DBType][]MessageDesc
		PostgresDBType   DBType
		DirectMessages   map[string][]MessageDesc
		DBTypes          []DBType
		Package          string
	}{
		Messages: messages,
		MessagesByDBType: map[DBType][]MessageDesc{
			DBTypePostgresSQL: filterMessagesForDBType(messages, DBTypePostgresSQL),
			DBTypeRedis:       filterMessagesForDBType(messages, DBTypeRedis),
		},
		PostgresDBType: DBTypePostgresSQL,
		DirectMessages: map[string][]MessageDesc{},
		DBTypes:        dbTypes,
		Package:        "schema",
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		t.Fatalf("execute metadata template: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"TableSharedTable",
		"TablePgsqlTable",
		"TableRedisTable",
		"PgsqlSharedTable",
		"PgsqlPgsqlTable",
		"&pgsql.SharedTable{}",
		"&pgsql.PgsqlTable{}",
		"&redis.SharedTable{}",
		"&redis.RedisTable{}",
		`NodeTables["pgsql"]["game"]`,
		`NodeTables["redis"]["social"]`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("metadata template missing %q in:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{
		"PgsqlRedisTable",
		"&pgsql.RedisTable{}",
		"&redis.PgsqlTable{}",
	} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("metadata template unexpectedly contains %q in:\n%s", unwanted, out)
		}
	}
}

func TestBuildGormTagFromFieldTagIncludesJSON(t *testing.T) {
	tag := buildGormTag(FieldTag{
		Pk:    "primary_key;column:id",
		Index: "index:idx_id;column:id",
		JSON:  "type:json;column:profile",
	})
	want := `gorm:"primary_key;column:id;index:idx_id;type:json;column:profile"`
	if tag != want {
		t.Fatalf("buildGormTag() = %q, want %q", tag, want)
	}
}

func TestExtractGormFromOrmTags(t *testing.T) {
	got := extractGormFromOrmTags(FieldDesc{
		Tags: OptionalString{Value: `gorm:"autoCreateTime;<-:create" json:"createdAt"`, Valid: true},
	})
	want := "autoCreateTime;<-:create"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestFieldTagCommentIncludesCustomGorm(t *testing.T) {
	comment := fieldTagComment(FieldDesc{
		Name:       "created_at",
		OrmOptions: &FieldOrmOptions{},
		Tags:       OptionalString{Value: `gorm:"autoCreateTime;column:created_at"`, Valid: true},
	}, MessageDesc{Name: "RoleTimestamps", OrmOptions: MessageOrmOptions{}}, string(DBTypeMySQL))
	if !strings.Contains(comment, `"custom":"autoCreateTime;column:created_at"`) {
		t.Fatalf("missing custom gorm in %q", comment)
	}
}

func TestBuildGormTagDedupesDuplicateColumn(t *testing.T) {
	tag := buildGormTag(FieldTag{
		Pk:    "primary_key;column:server_id;autoIncrement:false",
		Index: "index:idx_game_role_server_name;column:server_id",
	})
	want := `gorm:"primary_key;column:server_id;autoIncrement:false;index:idx_game_role_server_name"`
	if tag != want {
		t.Fatalf("buildGormTag() = %q, want %q", tag, want)
	}
}

func TestDBTypeHelpers(t *testing.T) {
	if !DBTypeRedis.IsKV() || !DBTypeMongo.IsKV() {
		t.Fatal("redis/mongo should be KV")
	}
	if DBTypeMySQL.IsKV() || DBTypePostgresSQL.IsKV() {
		t.Fatal("mysql/pgsql should not be KV")
	}
	if !DBTypeMySQL.IsSQL() || !DBTypePostgresSQL.IsSQL() {
		t.Fatal("mysql/pgsql should be SQL")
	}
	if DBTypeRedis.IsSQL() {
		t.Fatal("redis should not be SQL")
	}
	for _, dt := range supportedDBTypes() {
		if err := dt.Validate(); err != nil {
			t.Fatalf("Validate(%s): %v", dt, err)
		}
		if dt.TemplateName() == "" {
			t.Fatalf("TemplateName(%s) empty", dt)
		}
	}
	if err := DBType("unknown").Validate(); err == nil {
		t.Fatal("unknown db type should fail Validate")
	}
}

func TestIsEmbeddedField(t *testing.T) {
	f := FieldDesc{Tags: OptionalString{Value: `gorm:"embedded"`, Valid: true}}
	if !isEmbeddedField(f) {
		t.Fatal("expected embedded")
	}
	if isJSONDBField(f) {
		t.Fatal("embedded must not be JSON field")
	}
	if getDBFieldType(f) != f.Type {
		f.Type = "BaseModel"
		if getDBFieldType(f) != "BaseModel" {
			t.Fatalf("getDBFieldType = %q", getDBFieldType(f))
		}
	}
}

func TestTemplateFail(t *testing.T) {
	_, err := templateFail("bad option")
	if err == nil || err.Error() != "bad option" {
		t.Fatalf("templateFail() = %v", err)
	}
}

func TestBuildGormTagFromFieldTagIncludesEmbedded(t *testing.T) {
	tag := buildGormTag(FieldTag{Embedded: "embedded"})
	want := `gorm:"embedded"`
	if tag != want {
		t.Fatalf("buildGormTag() = %q, want %q", tag, want)
	}
}

func TestBuildGormTagFromFieldTagIncludesBlob(t *testing.T) {
	tag := buildGormTag(FieldTag{Blob: "type:bytea;column:heros"})
	want := `gorm:"type:bytea;column:heros"`
	if tag != want {
		t.Fatalf("buildGormTag() = %q, want %q", tag, want)
	}
}
