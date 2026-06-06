package main

import (
	_ "embed"
	"fmt"
	"reflect"
	"strings"
	"text/template"
	"time"
	"unicode"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type DBType string

// DBType 数据库类型
const (
	DBTypeMySQL       DBType = "mysql"
	DBTypeTcaplus     DBType = "tcaplus"
	DBTypePostgresSQL DBType = "pgsql"
	DBTypeRedis       DBType = "redis"
	DBTypeMongo       DBType = "mongo"

	timePackage         = protogen.GoImportPath("time")
	errorsPackage       = protogen.GoImportPath("errors")
	fmtPackage          = protogen.GoImportPath("fmt")
	reflectPackage      = protogen.GoImportPath("reflect")
	contextPackage      = protogen.GoImportPath("context")
	protoPackage        = protogen.GoImportPath("google.golang.org/protobuf/proto")
	mapstructurePackage = protogen.GoImportPath("github.com/mitchellh/mapstructure")

	outputBaseInternalDir        = "internal"
	outputProtoDir               = "proto"
	outputTcaplusOptionProtoFile = outputBaseInternalDir + "/" + string(DBTypeTcaplus) + "/" + outputProtoDir + "/tcaplusservice.optionv1.proto"
)

var funcMap = template.FuncMap{
	"ToLower":             strings.ToLower,
	"ToUpper":             strings.ToUpper,
	"ToCamelCase":         toCamelCase,
	"ToSnakeCase":         toSnakeCase,
	"Join":                strings.Join,
	"Contains":            strings.Contains,
	"HasPrefix":           strings.HasPrefix,
	"HasSuffix":           strings.HasSuffix,
	"inc":                 func(i int) int { return i + 1 },
	"needNewValue":        isNeedNewValue,
	"isMessageValue":      isMessageValue,
	"getTypeDefaultValue": getTypeDefaultValue,
	"getTypeNewValue":     getTypeNewValue,
	"getFieldType":        getFieldType,
	"getMessageGoType":    getMessageGoType,
	"ToGoVarName":         toGoVarName,
	"ToPkType":            toPkType,
	"IsZeroValue":         isZeroValue,
	"JoinPk":              joinFieldsPk,
	"JoinIndex":           joinFieldsIndex,
	"IndexTag":            indexTagForField,
	"IsJSONDBField":       isJSONDBField,
	"IsEmbeddedField":     isEmbeddedField,
	"IsEnumDBField":       isEnumDBField,
	"getDBFieldType":      getDBFieldType,
	"FieldTagComment":     fieldTagComment,
	"EmbedTypeFields":     embedTypeFields,
	"IsOneofGroupField":   isOneofGroupField,
	"fail":                templateFail,
	"add":                 func(a, b int32) int32 { return a + b },
	"dict": func(values ...interface{}) map[string]interface{} {
		if len(values)%2 != 0 {
			return nil
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil
			}
			dict[key] = values[i+1]
		}
		return dict
	},
	"ShardingKeyExpr": shardingKeyExprFromAny,
}

func shardingKeyExprFromAny(v interface{}) string {
	f, ok := v.(FieldDesc)
	if !ok {
		return "0"
	}
	return shardingKeyExpr(f)
}

func shardingKeyExpr(f FieldDesc) string {
	field := "x." + toCamelCase(f.Name)
	switch f.Type {
	case "int64":
		return field
	case "int32", "uint32", "uint64":
		return "int64(" + field + ")"
	default:
		return "int64(" + field + ")"
	}
}

func (t DBType) Suffix() string {
	switch t {
	case DBTypeTcaplus:
		return "tca_"
	case DBTypeMySQL:
		return "tb_"
	case DBTypePostgresSQL:
		return "pg_"
	case DBTypeRedis:
		return "rd_"
	case DBTypeMongo:
		return "mg_"
	}
	return ""
}

func (t DBType) TemplateName() string {
	switch t {
	case DBTypeTcaplus:
		return tcaplusTemplate
	case DBTypeMySQL:
		return mysqlTemplate
	case DBTypePostgresSQL:
		return pgsqlTemplate
	case DBTypeRedis, DBTypeMongo:
		return kvTemplate
	}
	return ""
}

// IsKV 是否为 KV 驱动（redis / mongo），仅支持 PAYLOAD 整包存储。
func (t DBType) IsKV() bool {
	return t == DBTypeRedis || t == DBTypeMongo
}

// IsSQL 是否为 GORM SQL 驱动（mysql / pgsql）。
func (t DBType) IsSQL() bool {
	return t == DBTypeMySQL || t == DBTypePostgresSQL
}

// Validate 校验插件支持的 dbType。
func (t DBType) Validate() error {
	switch t {
	case DBTypeMySQL, DBTypeTcaplus, DBTypePostgresSQL, DBTypeRedis, DBTypeMongo:
		return nil
	default:
		return fmt.Errorf("unsupported db type: %s", t)
	}
}

func supportedDBTypes() []DBType {
	return []DBType{DBTypeMySQL, DBTypeTcaplus, DBTypePostgresSQL, DBTypeRedis, DBTypeMongo}
}

// hasBlobTag 字段含 gorm type:blob 时走 protobuf 二进制列，而非 JSON 列编解码。
func hasBlobTag(field FieldDesc) bool {
	if !field.Tags.Valid {
		return false
	}
	return strings.Contains(strings.ToLower(field.Tags.Value), "type:blob")
}

func defaultNodeType() string {
	return "default"
}

func templateFail(args ...any) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("template fail")
	}
	return "", fmt.Errorf("%v", args[0])
}

func isEmbeddedField(field FieldDesc) bool {
	if !field.Tags.Valid {
		return false
	}
	return strings.Contains(strings.ToLower(field.Tags.Value), "embedded")
}

func isJSONDBField(field FieldDesc) bool {
	if isEmbeddedField(field) {
		return false
	}
	if hasBlobTag(field) {
		return false
	}
	if field.List {
		return true
	}
	if strings.HasPrefix(field.Type, "map<") {
		return true
	}
	if field.F != nil {
		desc := field.F.Desc
		if desc.IsMap() || desc.IsList() {
			return true
		}
		switch desc.Kind() {
		case protoreflect.MessageKind, protoreflect.GroupKind:
			return true
		default:
			return false
		}
	}
	switch field.Type {
	case "bool", "int32", "int64", "uint32", "uint64", "float32", "float64", "string", "bytes":
		return false
	}
	return true
}

func getDBFieldType(field FieldDesc) string {
	if isEmbeddedField(field) {
		return field.Type
	}
	if isJSONDBField(field) || hasBlobTag(field) {
		return "bytes"
	}
	if isEnumDBField(field) {
		return "int32"
	}
	switch field.Type {
	case "float32":
		return "float"
	case "float64":
		return "double"
	}
	return field.Type
}

func isEnumDBField(field FieldDesc) bool {
	return field.F != nil && field.F.Desc.Kind() == protoreflect.EnumKind
}

func extractGormFromOrmTags(field FieldDesc) string {
	if !field.Tags.Valid {
		return ""
	}
	tags := strings.ReplaceAll(field.Tags.Value, `\"`, `"`)
	tags = strings.ReplaceAll(tags, `\\`, `\`)
	const prefix = `gorm:"`
	idx := strings.Index(tags, prefix)
	if idx < 0 {
		return ""
	}
	start := idx + len(prefix)
	end := strings.Index(tags[start:], `"`)
	if end < 0 {
		return ""
	}
	return tags[start : start+end]
}

func fieldTagComment(field FieldDesc, msg MessageDesc, dbType string) string {
	parts := make([]string, 0, 6)
	opts := field.OrmOptions
	if opts == nil {
		opts = &FieldOrmOptions{}
	}
	if opts.HasPrimaryKey {
		parts = append(parts, fmt.Sprintf(`"pk":"primary_key;column:%s;autoIncrement:false"`, toSnakeCase(field.Name)))
	}
	if tag := indexTagForField(msg, field); tag != "" {
		parts = append(parts, fmt.Sprintf(`"index":"index:%s"`, tag))
	}
	if isEmbeddedField(field) {
		parts = append(parts, `"embedded":"embedded"`)
	}
	if opts.IsJSONField {
		colType := "json"
		if dbType == string(DBTypePostgresSQL) {
			colType = "jsonb"
		}
		parts = append(parts, fmt.Sprintf(`"json":"type:%s;column:%s;serializer:json"`, colType, toSnakeCase(field.Name)))
	}
	if opts.IsBlobField {
		colType := "blob"
		if dbType == string(DBTypePostgresSQL) {
			colType = "bytea"
		}
		parts = append(parts, fmt.Sprintf(`"blob":"type:%s;column:%s"`, colType, toSnakeCase(field.Name)))
	}
	// orm.tags 的 gorm 片段仅用于非 table 的 embed 子消息字段；table 列由生成器 pk/index/blob/json 负责。
	if custom := extractGormFromOrmTags(field); custom != "" && !msg.OrmOptions.IsTable {
		parts = append(parts, fmt.Sprintf(`"custom":"%s"`, custom))
	}
	parts = append(parts, fmt.Sprintf(`"origin":"%s"`, field.Name))
	return strings.Join(parts, ", ")
}

func isOneofGroupField(field FieldDesc) bool {
	return field.Type == "oneof"
}

func embedTypeFields(messages []MessageDesc, typeName string) []FieldDesc {
	for _, m := range messages {
		if m.Name == typeName {
			return m.Fields
		}
		for _, nested := range m.NestedMessages {
			if nested.Name == typeName {
				return nested.Fields
			}
		}
	}
	return nil
}

// toCamelCase 转换为驼峰命名
func toCamelCase(s string) string {
	var result strings.Builder
	upperNext := true

	for i, r := range s {
		if r == '_' {
			// 检查下一个字符是否大写，如果是则保留原样
			if i+1 < len(s) && unicode.IsUpper(rune(s[i+1])) {
				result.WriteRune('_')
				continue
			}
			upperNext = true
			continue
		}

		if upperNext {
			result.WriteRune(unicode.ToUpper(r))
			upperNext = false
		} else {
			// 数字后的字母大写
			if i > 0 && unicode.IsDigit(rune(s[i-1])) && unicode.IsLetter(r) {
				result.WriteRune(unicode.ToUpper(r))
			} else {
				result.WriteRune(r)
			}
		}
	}

	return result.String()
}

// toSnakeCase 转换为蛇形命名
func toSnakeCase(s string) string {
	var result strings.Builder
	for i, char := range s {
		if i > 0 && isUpper(char) && isLower(rune(s[i-1])) {
			result.WriteByte('_')
		}
		result.WriteRune(unicode.ToLower(char))
	}
	return result.String()
}

func isUpper(r rune) bool {
	return 'A' <= r && r <= 'Z'
}

func isLower(r rune) bool {
	return 'a' <= r && r <= 'z'
}

func stringEscape(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r == '"' || r == '\\' {
			result.WriteByte('\\')
		}
		result.WriteRune(r)
	}
	return result.String()
}

func toGoVarName(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// replace .
	s = strings.ReplaceAll(s, ".", "_")
	// replace -
	s = strings.ReplaceAll(s, "-", "_")
	// replace space
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func generateHeader(gen *protogen.Plugin, g *protogen.GeneratedFile, file *protogen.File, dbType DBType) {
	generateBaseHeader(gen, g, dbType)
	if file != nil {
		if file.Proto.GetOptions().GetDeprecated() {
			g.P("// ", file.Desc.Path(), " is a deprecated file.")
		} else {
			g.P("// source: ", file.Desc.Path())
		}
	}
	g.P("// time ", time.Now().Format("2006-01-02"))
	g.P()
}

func generateBaseHeader(gen *protogen.Plugin, g *protogen.GeneratedFile, dbType DBType) {
	// Code generated by protoc-gen-go-orm. DO NOT EDIT.
	// versions:
	//  protoc-gen-go-orm {{.Version}}
	//  protoc           {{.ProtocVersion}}
	// source: {{.Source}}
	// dbType: {{.DBType}}
	// Table-related enhancements for {{.DBType}}.{{.Message.Name}}

	g.P("// Code generated by protoc-gen-orm. DO NOT EDIT.")
	g.P("// versions:")
	g.P("//  protoc-gen-orm ", version)
	protocVersion := "(unknown)"
	if v := gen.Request.GetCompilerVersion(); v != nil {
		protocVersion = fmt.Sprintf("v%v.%v.%v", v.GetMajor(), v.GetMinor(), v.GetPatch())
		if s := v.GetSuffix(); s != "" {
			protocVersion += "-" + s
		}
	}
	g.P("//  protoc           ", protocVersion)
	if len(dbType) > 0 {
		g.P("// dbType           ", dbType)
	}
}

func toPkType(typ string) string {
	switch typ {
	case "int32", "uint32", "sint32", "fixed32", "sfixed32":
		return "int"
	case "int64", "uint64", "sint64", "fixed64", "sfixed64":
		return "bigint"
	case "string":
		return "varchar(255)"
	default:
		panic(fmt.Sprintf("unsupported pk type %v, only int32, int64, string supported", typ))
	}
}

func isZeroValue(val interface{}) bool {
	if val == nil {
		return true
	}
	rv := reflect.ValueOf(val)
	return rv.IsZero() || rv.IsNil()
}

func joinFieldsPk(msg *MessageDesc) string {

	var pkFields []string
	for _, field := range msg.Fields {
		if field.OrmOptions.HasPrimaryKey {
			pkFields = append(pkFields, toSnakeCase(field.Name))
		}
	}
	return strings.Join(pkFields, ", ")
}

