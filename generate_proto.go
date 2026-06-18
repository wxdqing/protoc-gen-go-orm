package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/wxdqing/protoc-gen-go-orm/options"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// MessageDesc 消息描述
type MessageDesc struct {
	Name           string
	Comment        string
	Tcaplus        TcaplusMessageInfo
	OrmOptions     MessageOrmOptions
	Fields         []FieldDesc
	NestedMessages []*MessageDesc
	NestedEnums    []*EnumDesc
	TableName      string
	FilePath       string
	Desc           protoreflect.MessageDescriptor
}

// TcaplusMessageInfo Tcaplus特定信息
type TcaplusMessageInfo struct {
	PrimaryKey        OptionalString
	Indexs            OptionalArray
	FieldCipherSuite  OptionalString
	RecordCipherSuite OptionalString
	CipherMd5         OptionalString
	ShardingKey       OptionalString
	CustomAttr        OptionalString
	EnableCache       OptionalBool

	// Deprecated: rm in future
	CacheExpire int32
	TableName   string
	// Deprecated: rm in future
	ZoneTableMode string
}

type TcaplusFieldInfo struct {
	HasValue      bool
	TcaplusSize   OptionalInt
	TcaplusDesc   OptionalValue
	TcaplusCrypto OptionalBool
}

// FieldDesc 字段描述
type FieldDesc struct {
	Name    string
	Type    string
	List    bool
	Number  int32
	Comment string

	Tags           OptionalString
	SkipSetDefault bool
	Oneofs         OptionalValue
	// Deprecated: rm in future
	PrimaryKey bool
	// Deprecated: rm in future
	TcaplusPrimaryKey bool

	F          *protogen.Field
	OrmOptions *FieldOrmOptions
	Tcaplus    TcaplusFieldInfo
}

type EnumDesc struct {
	Name    string
	Comment string
	Values  []*EnumValueDesc
}

type EnumValueDesc struct {
	Name    string
	Number  int32
	Comment string
}

type OneOfDesc struct {
	Name    string
	Comment string
	Fields  []FieldDesc
}

type OptionalString struct {
	Value string
	Valid bool
}

type OptionalInt struct {
	Value int
	Valid bool
}

type OptionalBool struct {
	Value bool
	Valid bool
}

type OptionalValue struct {
	Value any
	Valid bool
}

type OptionalArray struct {
	Value []any
	Valid bool
}

// MessageOrmOptions 表选项
type MessageOrmOptions struct {
	IsTable             bool
	TableName           OptionalString
	NodeType            OptionalString
	TableStoreMode      options.TableStoreMode
	HasPrimaryKey       bool
	HasIndexes          bool
	HasShardingKey      bool
	HasVersion          bool
	ShardingKeyField    OptionalValue
	CompositeIndexSpecs []string // (orm.composite_index)，如 idx_pk(rid,id)
	PartialIndexSpecs   []string // (orm.partial_index)
	DbDrivers           []string // (orm.db_driver)，空=全部驱动
}

type FieldOrmOptions struct {
	HasPrimaryKey  bool
	HasIndex       bool
	HasShardingKey bool
	HasVersion     bool
	IsJSONField      bool
	IsBlobField      bool
	IsEmbeddedField  bool
	HasForeignKey    bool
	ForeignKeySpec   string
	Optional         bool // proto3 optional scalar
}

// generateFile 生成文件
func generate(gen *protogen.Plugin, file *protogen.File, cb func([]MessageDesc, []EnumDesc)) error {
	if len(file.Messages) == 0 {
		return nil
	}
	if err := generateFiles(gen, file, cb); err != nil {
		return err
	}
	return nil
}

func generateFiles(gen *protogen.Plugin, file *protogen.File, cb func([]MessageDesc, []EnumDesc)) error {
	// 收集消息
	messages, err := collectMessages(file)
	if err != nil {
		panic(fmt.Errorf("collect messages failed: %w", err))
	}
	enums, err := collectEnums(file)
	if err != nil {
		panic(fmt.Errorf("collect enums failed: %w", err))
	}

	for _, dbType := range targetDBTypes() {
		filtered := filterMessagesForDBType(messages, dbType)
		if len(filtered) == 0 {
			continue
		}
		if err = generateForDBType(gen, file, filtered, enums, dbType); err != nil {
			panic(fmt.Errorf("generate %s failed: %w", dbType, err))
		}
	}
	err = generateValues(gen, file, messages, enums)
	if err != nil {
		panic(err)
	}
	cb(messages, enums)
	return nil
}

// warnFieldsModeOnKV 在 redis/mongo 上生成 FIELDS 表时提示：KV 仅支持 PAYLOAD。
func warnFieldsModeOnKV(sourcePath string, msg *MessageDesc, dbType DBType) {
	if msg == nil || !dbType.IsKV() {
		return
	}
	if !msg.OrmOptions.IsTable {
		return
	}
	if msg.OrmOptions.TableStoreMode != options.TableStoreMode_TABLE_STORE_MODE_FIELDS {
		return
	}
	if len(msg.OrmOptions.DbDrivers) > 0 && !messageSupportsDBType(*msg, dbType) {
		return
	}
	if sourcePath == "" {
		sourcePath = "?"
	}
	fmt.Fprintf(os.Stderr,
		"%s: warning: table %q uses TABLE_STORE_MODE_FIELDS for %s; KV drivers only support PAYLOAD (pk/index/version + data). Generated code uses kv.tmpl.\n",
		sourcePath, msg.Name, dbType,
	)
}

func messageSupportsDBType(msg MessageDesc, dbType DBType) bool {
	if !msg.OrmOptions.IsTable {
		return true
	}
	if len(msg.OrmOptions.DbDrivers) == 0 {
		return true
	}
	for _, d := range msg.OrmOptions.DbDrivers {
		if DBType(strings.TrimSpace(d)) == dbType {
			return true
		}
	}
	return false
}

func filterMessagesForDBType(messages []MessageDesc, dbType DBType) []MessageDesc {
	var out []MessageDesc
	for _, msg := range messages {
		if messageSupportsDBType(msg, dbType) {
			out = append(out, msg)
		}
	}
	return out
}

func activeDBTypes(messages []MessageDesc) []DBType {
	allowed := targetDBTypeSet()
	used := make(map[DBType]bool)
	for _, msg := range messages {
		if !msg.OrmOptions.IsTable {
			continue
		}
		if len(msg.OrmOptions.DbDrivers) == 0 {
			for _, t := range targetDBTypes() {
				used[t] = true
			}
			continue
		}
		for _, d := range msg.OrmOptions.DbDrivers {
			t := DBType(strings.TrimSpace(d))
			if allowed != nil && !allowed[t] {
				continue
			}
			if err := t.Validate(); err != nil {
				continue
			}
			used[t] = true
		}
	}
	var out []DBType
	for _, t := range targetDBTypes() {
		if used[t] {
			out = append(out, t)
		}
	}
	return out
}

// generateForDBType 为特定数据库类型生成文件
func generateForDBType(gen *protogen.Plugin, file *protogen.File, messages []MessageDesc, enums []EnumDesc, dbType DBType) error {
	for _, msg := range messages {
		if msg.OrmOptions.IsTable {
			warnFieldsModeOnKV(file.Desc.Path(), &msg, dbType)
		}
	}

	// 过滤出需要生成的消息
	//var filteredMessages []MessageDesc
	//for _, msg := range messages {
	//	if isMessageTable(msg) {
	//		filteredMessages = append(filteredMessages, msg)
	//	}
	//}
	//
	//if len(filteredMessages) == 0 {
	//	return nil
	//}

	// 准备模板数据
	data := struct {
		Version       string
		ProtocVersion string
		Package       string
		GoPackage     string
		GoImportPath  string
		Messages      []MessageDesc
		DBType        string
		Source        string
		Enums         []EnumDesc
	}{
		Version:       version,
		ProtocVersion: protocVersion,
		Package:       string(file.Desc.Package()),
		GoPackage:     string(file.GoPackageName),
		GoImportPath:  string(file.GoImportPath),
		Messages:      messages,
		DBType:        string(dbType),
		Source:        file.Desc.Path(),
		Enums:         enums,
	}

	// 创建输出文件
	filename := generateOutputFilename(file, dbType)
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	// 解析模板
	tmpl := template.New(fmt.Sprintf("orm-%s", dbType)).Funcs(funcMap)

	//// 解析函数模板
	//tmpl, err := tmpl.Parse(funcTemplate)
	//if err != nil {
	//	return fmt.Errorf("parse func template failed: %w", err)
	//}

	if err := dbType.Validate(); err != nil {
		return err
	}
	dbTemplate := dbType.TemplateName()
	if dbTemplate == "" {
		return fmt.Errorf("unsupported db type: %s", dbType)
	}

	tmpl, err := tmpl.Parse(dbTemplate)
	if err != nil {
		return fmt.Errorf("parse db template failed: %w", err)
	}

	// 执行模板
	if err := tmpl.Execute(g, data); err != nil {
		return fmt.Errorf("execute template failed: %w", err)
	}

	// generate methods file
	err = generateMethods(gen, file, messages, enums, dbType, protocVersion)
	if err != nil {
		return err
	}
	err = generateContext(gen, file, dbType)
	if err != nil {
		return err
	}
	return nil
}

// collectMessages 收集文件中的所有消息
func collectMessages(file *protogen.File) ([]MessageDesc, error) {
	var messages []MessageDesc

	for _, msg := range file.Messages {
		messageDesc, err := parseMessageOfProto(msg)
		if err != nil {
			return nil, fmt.Errorf("collect message %s failed: %w", msg.Desc.Name(), err)
		}
		messageDesc.FilePath = file.Desc.Path()
		messages = append(messages, messageDesc)
	}

	return messages, nil
}

func parseMessageOfProto(msg *protogen.Message) (MessageDesc, error) {
	messageDesc := MessageDesc{
		Name:      string(msg.Desc.Name()),
		Comment:   msg.Comments.Leading.String(),
		TableName: toSnakeCase(string(msg.Desc.Name())),
		Desc:      msg.Desc,
		OrmOptions: MessageOrmOptions{
			NodeType:       OptionalString{Value: defaultNodeType(), Valid: true},
			TableStoreMode: options.TableStoreMode_TABLE_STORE_MODE_PAYLOAD,
		},
	}

	// 处理Tcaplus特定选项
	//messageDesc.Tcaplus = parseTcaplusOptions(msg)
	fts := parseTcaplusOptions(msg)
	fos := parseOrmOptionsFromMessage(msg)
	for _, ft := range fts {
		ft(&messageDesc)
	}
	for _, fo := range fos {
		fo(&messageDesc)
	}
	// 收集字段
	fields, err := collectFields(msg, func(fd *FieldDesc) {
		// tcaplus pk
		if !messageDesc.Tcaplus.PrimaryKey.Valid {
			return
		}
		arr := strings.Split(messageDesc.Tcaplus.PrimaryKey.Value, ",")
		for _, s := range arr {
			trimmed := strings.TrimSpace(s)
			if fd.Name == trimmed {
				fd.TcaplusPrimaryKey = true
				break
			}
		}
	})
	if err != nil {
		return messageDesc, err
	}
	messageDesc.Fields = fields

	// nested messages
	for _, nestedMsg := range msg.Messages {
		if nestedMsg.Desc.IsMapEntry() {
			continue
		}
		nestedDesc, err := parseMessageOfProto(nestedMsg)
		if err != nil {
			return messageDesc, fmt.Errorf("collect nested message %s failed: %w", nestedMsg.Desc.Name(), err)
		}
		messageDesc.NestedMessages = append(messageDesc.NestedMessages, &nestedDesc)
	}

	// nested enums
	for _, nestedEnum := range msg.Enums {
		enumDesc, err := parseEnumsOfProto(nestedEnum)
		if err != nil {
			return messageDesc, fmt.Errorf("collect nested enum %s failed: %w", nestedEnum.Desc.Name(), err)
		}
		messageDesc.NestedEnums = append(messageDesc.NestedEnums, &enumDesc)
	}

	// fill orm options HasPrimaryKey and HasIndexes
	for _, f := range messageDesc.Fields {
		if f.OrmOptions.HasPrimaryKey {
			messageDesc.OrmOptions.HasPrimaryKey = true
		}
		if f.OrmOptions.HasIndex {
			messageDesc.OrmOptions.HasIndexes = true
		}
		if f.OrmOptions.HasShardingKey {
			messageDesc.OrmOptions.HasShardingKey = true
		}
		if f.OrmOptions.HasVersion {
			messageDesc.OrmOptions.HasVersion = true
		}
	}

	// tcaplus sharding key设置
	shardingKeyPresent := false
	for _, field := range fields {
		if field.OrmOptions.HasShardingKey {
			shardingKeyPresent = true
			break
		}
	}
	// tcaplus sharding key设置，如未找到默认使用第一个pk字段
	if !shardingKeyPresent && messageDesc.OrmOptions.HasIndexes {
		for _, field := range fields {
			if field.OrmOptions.HasPrimaryKey {
				field.OrmOptions.HasShardingKey = true
				//fmt.Println("no tcaplus sharding key found, use first pk field:", field.Name)
				break
			}
		}
	}
	for _, f := range messageDesc.Fields {
		if f.OrmOptions.HasShardingKey {
			messageDesc.OrmOptions.HasShardingKey = true
			messageDesc.OrmOptions.ShardingKeyField = OptionalValue{Value: f, Valid: true}
		}
	}
	return messageDesc, nil
}

func collectEnums(file *protogen.File) ([]EnumDesc, error) {
	var enums []EnumDesc
	for _, enum := range file.Enums {
		enumDesc, err := parseEnumsOfProto(enum)
		if err != nil {
			return nil, fmt.Errorf("collect enum %s failed: %w", enum.Desc.Name(), err)
		}
		enums = append(enums, enumDesc)
	}
	return enums, nil
}

func parseEnumsOfProto(enum *protogen.Enum) (EnumDesc, error) {
	enumDesc := EnumDesc{
		Name:    string(enum.Desc.Name()),
		Comment: enum.Comments.Leading.String(),
	}
	for _, value := range enum.Values {
		valueDesc := &EnumValueDesc{
			Name:    string(value.Desc.Name()),
			Number:  int32(value.Desc.Number()),
			Comment: value.Comments.Leading.String(),
		}
		enumDesc.Values = append(enumDesc.Values, valueDesc)
	}
	return enumDesc, nil
}

// collectFields 收集消息中的所有字段
func collectFields(msg *protogen.Message, cb ...func(fd *FieldDesc)) ([]FieldDesc, error) {
	var fields []FieldDesc
	var oneofs = make(map[string]struct{})
	for _, field := range msg.Fields {
		var fieldDesc FieldDesc
		if field.Oneof != nil && !field.Oneof.Desc.IsSynthetic() {
			name := string(field.Oneof.Desc.Name())
			if _, ok := oneofs[name]; ok {
				continue
			}
			fieldDesc = FieldDesc{
				Name:       name,
				Type:       "oneof",
				Comment:    field.Oneof.Comments.Leading.String(),
				Number:     -1,
				F:          field,
				OrmOptions: &FieldOrmOptions{},
			}
			oneofs[name] = struct{}{}
			oneofFns := parseFieldOneOfs(field, msg)
			for _, ofn := range oneofFns {
				ofn(&fieldDesc)
			}

		} else {
			fos := parseOrmFieldOptions(field)
			fts := parseTcaplusFieldOptions(field)
			fieldDesc = FieldDesc{
				Name:       string(field.Desc.Name()),
				Type:       getFieldType(field),
				Comment:    field.Comments.Leading.String(),
				Number:     int32(field.Desc.Number()),
				List:       field.Desc.IsList(),
				F:          field,
				OrmOptions: &FieldOrmOptions{},
			}
			for _, fo := range fos {
				fo(&fieldDesc)
			}
			for _, ft := range fts {
				ft(&fieldDesc)
			}
			if field.Desc.HasOptionalKeyword() {
				fieldDesc.OrmOptions.Optional = true
			}
			fieldDesc.OrmOptions.IsEmbeddedField = isEmbeddedField(fieldDesc)
			fieldDesc.OrmOptions.IsJSONField = isJSONDBField(fieldDesc)
			fieldDesc.OrmOptions.IsBlobField = hasBlobTag(fieldDesc)
		}

		// 获取Tcaplus类型
		//if tcaplusType, ok := fieldDesc.MessageOrmOptions["tcaplus_type"]; ok {
		//	fieldDesc.TcaplusType = tcaplusType
		//}

		for _, f := range cb {
			f(&fieldDesc)
		}
		fields = append(fields, fieldDesc)
	}
	return fields, nil
}

// parseTcaplusOptions 解析Tcaplus特定选项
func parseTcaplusOptions(msg *protogen.Message) []func(desc *MessageDesc) {
	//tcaplus := TcaplusMessageInfo{}
	//options := parseOrmOptionsFromMessage(msg)

	fns := make([]func(desc *MessageDesc), 0)
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusPrimaryKey); opt != nil {
		//opts["tcaplusservice.tcaplus_primary_key"] =  stringEscape(opt.(string))
		s := opt.(string)
		if s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.Tcaplus.PrimaryKey = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusIndex); opt != nil {
		//opts["tcaplusservice.tcaplus_index"] =  stringEscape(opt.(string))
		arr := opt.([]string)
		if len(arr) != 0 {
			fns = append(fns, func(desc *MessageDesc) {
				v := make([]any, 0, len(arr))
				for _, s := range arr {
					v = append(v, s)
				}
				desc.Tcaplus.Indexs = OptionalArray{Value: v, Valid: true}
			})
		}
	}
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusFieldCipherSuite); opt != nil {
		//opts["tcaplusservice.tcaplus_field_cipher_suite"] =  stringEscape(opt.(string))
		s := opt.(string)
		if s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.Tcaplus.FieldCipherSuite = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusRecordCipherSuite); opt != nil {
		//opts["tcaplusservice.tcaplus_record_cipher_suite"] =  stringEscape(opt.(string))
		s := opt.(string)
		if s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.Tcaplus.RecordCipherSuite = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusCipherMd5); opt != nil {
		//opts["tcaplusservice.tcaplus_cipher_md5"] = fmt.Sprintf("%t", *opt.(*bool))
		s := opt.(string)
		if s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.Tcaplus.CipherMd5 = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusShardingKey); opt != nil {
		//opts["tcaplusservice.tcaplus_sharding_key"] =  stringEscape( stringEscape(opt.(string)))
		s := opt.(string)
		if s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.Tcaplus.ShardingKey = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	if opt := proto.GetExtension(msg.Desc.Options(), options.E_TcaplusCustomattr); opt != nil {
		//opts["tcaplusservice.tcaplus_customattr"] =  stringEscape(opt.(string))
		s := opt.(string)
		if s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.Tcaplus.CustomAttr = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	return fns
}

func parseOrmOptionsFromMessage(msg *protogen.Message) []func(desc *MessageDesc) {
	return messageOrmOptionAppliers(msg.Desc.Options())
}

func applyOrmMessageOptions(desc *MessageDesc, msgOpts protoreflect.ProtoMessage) {
	for _, fn := range messageOrmOptionAppliers(msgOpts) {
		fn(desc)
	}
}

func messageOrmOptionAppliers(msgOpts protoreflect.ProtoMessage) []func(desc *MessageDesc) {
	fns := make([]func(desc *MessageDesc), 0)
	if opt := proto.GetExtension(msgOpts, options.E_Table); opt != nil {
		fns = append(fns, func(desc *MessageDesc) {
			desc.OrmOptions.IsTable = opt.(bool)
		})
	}
	if proto.HasExtension(msgOpts, options.E_NodeType) {
		opt := proto.GetExtension(msgOpts, options.E_NodeType)
		if s := opt.(string); s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.OrmOptions.NodeType = OptionalString{Value: stringEscape(s), Valid: true}
			})
		}
	}
	if proto.HasExtension(msgOpts, options.E_TableStoreMode) {
		opt := proto.GetExtension(msgOpts, options.E_TableStoreMode)
		mode := opt.(options.TableStoreMode)
		fns = append(fns, func(desc *MessageDesc) {
			desc.OrmOptions.TableStoreMode = mode
		})
	}
	if proto.HasExtension(msgOpts, options.E_CompositeIndex) {
		opt := proto.GetExtension(msgOpts, options.E_CompositeIndex)
		if arr, ok := opt.([]string); ok && len(arr) > 0 {
			specs := make([]string, len(arr))
			copy(specs, arr)
			fns = append(fns, func(desc *MessageDesc) {
				desc.OrmOptions.CompositeIndexSpecs = specs
				desc.OrmOptions.HasIndexes = true
			})
		}
	}
	if proto.HasExtension(msgOpts, options.E_TableName) {
		opt := proto.GetExtension(msgOpts, options.E_TableName)
		if s := strings.TrimSpace(opt.(string)); s != "" {
			fns = append(fns, func(desc *MessageDesc) {
				desc.TableName = s
				desc.OrmOptions.TableName = OptionalString{Value: s, Valid: true}
			})
		}
	}
	if proto.HasExtension(msgOpts, options.E_PartialIndex) {
		opt := proto.GetExtension(msgOpts, options.E_PartialIndex)
		if arr, ok := opt.([]string); ok && len(arr) > 0 {
			specs := make([]string, len(arr))
			copy(specs, arr)
			fns = append(fns, func(desc *MessageDesc) {
				desc.OrmOptions.PartialIndexSpecs = specs
				desc.OrmOptions.HasIndexes = true
			})
		}
	}
	if proto.HasExtension(msgOpts, options.E_DbDriver) {
		opt := proto.GetExtension(msgOpts, options.E_DbDriver)
		if arr, ok := opt.([]string); ok && len(arr) > 0 {
			drivers := make([]string, 0, len(arr))
			for _, d := range arr {
				if s := strings.TrimSpace(d); s != "" {
					drivers = append(drivers, s)
				}
			}
			if len(drivers) > 0 {
				fns = append(fns, func(desc *MessageDesc) {
					desc.OrmOptions.DbDrivers = drivers
				})
			}
		}
	}
	return fns
}

func parseFieldOneOfs(field *protogen.Field, msg *protogen.Message) []func(desc *FieldDesc) {
	fns := make([]func(desc *FieldDesc), 0)
	if field.Oneof != nil && !field.Oneof.Desc.IsSynthetic() {
		fields := make([]FieldDesc, 0)
		// create oneof fields descriptions
		for _, f := range field.Oneof.Fields {
			desc := FieldDesc{
				Name:    string(f.Desc.Name()),
				Type:    getFieldType(f),
				Comment: f.Comments.Leading.String(),
				Number:  int32(f.Desc.Number()),
				Tags: OptionalString{
					Value: "opqqq",
					Valid: true,
				},
				F:          f,
				OrmOptions: &FieldOrmOptions{},
			}

			var origin *protogen.Field
			for _, mf := range msg.Fields {
				if mf.Desc.Number() == f.Desc.Number() {
					origin = mf
					break
				}
			}
			fos := parseOrmFieldOptions(origin)
			fts := parseTcaplusFieldOptions(origin)
			for _, fo := range fos {
				fo(&desc)
			}
			for _, ft := range fts {
				ft(&desc)
			}
			fields = append(fields, desc)
		}

		fns = append(fns, func(desc *FieldDesc) {
			val := &OneOfDesc{
				Name:    string(field.Oneof.Desc.Name()),
				Fields:  fields,
				Comment: field.Oneof.Comments.Leading.String(),
			}
			desc.Oneofs = OptionalValue{Value: val, Valid: true}
		})
	}
	return fns
}

// getFieldType 获取字段类型
func getFieldType(field *protogen.Field) string {
	return getTypeOfFieldDescriptor(field.Desc)
}

func getTypeOfFieldDescriptor(desc protoreflect.FieldDescriptor) string {
	if desc.IsMap() {
		keyKind := desc.MapKey()
		valKind := desc.MapValue()
		return fmt.Sprintf("map<%s, %s>", getTypeOfFieldDescriptor(keyKind), getTypeOfFieldDescriptor(valKind))

	}
	switch desc.Kind() {
	case protoreflect.BoolKind:
		return "bool"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64"
	case protoreflect.FloatKind:
		return "float32"
	case protoreflect.DoubleKind:
		return "float64"
	case protoreflect.StringKind:
		return "string"
	case protoreflect.BytesKind:
		return "bytes"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		s := string(desc.Message().FullName())
		arr := strings.Split(s, ".")
		if len(arr) > 1 {
			arr = arr[1:] // remove package name
		}
		return strings.Join(arr, ".")
		//return string(desc.Message().Name())
	case protoreflect.EnumKind:
		return string(desc.Enum().Name())
	}

	return desc.Kind().String()
}

func getMessageGoType(g *protogen.GeneratedFile, msg protoreflect.MessageDescriptor) string {
	//arr := make([]string, 0)
	//parent := msg.Parent()
	//for parent != nil {
	//	pd, ok := parent.(protoreflect.MessageDescriptor)
	//	if !ok || pd == nil {
	//		parent = nil // end
	//	} else {
	//		//p := getMessageGoType(g, pd)
	//		//if p != "" {
	//		//	arr = append(arr, p)
	//		//}
	//		arr = append(arr, string(pd.FullName()))
	//		parent = pd.Parent()
	//	}
	//}
	//arr = append(arr, string(msg.FullName()))
	//return strings.Join(arr, "_")

	s := string(msg.FullName())
	arr := strings.Split(s, ".")
	if len(arr) > 1 {
		arr = arr[1:] // remove package name
	}
	return strings.Join(arr, "_")
}

func getTypeNewValue(g *protogen.GeneratedFile, f FieldDesc) string {
	desc := f.F.Desc
	if desc.IsMap() {
		keyKind := getTypeOfFieldDescriptor(desc.MapKey())
		//keyKind := getMessageGoType(g, desc.MapKey().Message())
		if desc.MapKey().Kind() == protoreflect.MessageKind {
			keyKind = "*" + keyKind
		}
		valKind := getTypeOfFieldDescriptor(desc.MapValue())
		if desc.MapValue().Kind() == protoreflect.MessageKind {
			valKind = getMessageGoType(g, desc.MapValue().Message())
			valKind = "*" + valKind
		}
		return fmt.Sprintf("make(map[%s]%s)", keyKind, valKind)
		//return g.QualifiedGoIdent(f.F.Message.GoIdent)
	}
	if desc.IsList() {
		el := getTypeOfFieldDescriptor(desc)
		if desc.Kind() == protoreflect.MessageKind {
			el = getMessageGoType(g, desc.Message())
			el = "*" + el
		}
		return "make([]" + el + ", 0)"
	}
	switch desc.Kind() {
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return "int32(0)"
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return "int64(0)"
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return "uint32(0)"
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return "uint64(0)"
	case protoreflect.FloatKind:
		return "float32(0)"
	case protoreflect.DoubleKind:
		return "float64(0)"
	case protoreflect.BoolKind:
		return "false"
	case protoreflect.StringKind:
		return `""`
	case protoreflect.BytesKind:
		return "make([]byte, 0)"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		n := g.QualifiedGoIdent(f.F.Message.GoIdent)
		return fmt.Sprintf("&%s{}", n)
		//return fmt.Sprintf("&%s{}", desc.Message().Name())
	case protoreflect.EnumKind:
		// 获取枚举的第一个值作为默认值
		//enum := desc.Enum()
		//if enum.Values().Len() > 0 {
		//	return fmt.Sprintf("%s_%s", enum.Name(), enum.Values().Get(0).Name())
		//}
		return "0"
	}
	return "nil"
}

func getTypeDefaultValue(desc protoreflect.FieldDescriptor) string {
	if desc.IsMap() {
		return "nil"
	}
	if desc.IsList() {
		return "nil"
	}
	switch desc.Kind() {
	case protoreflect.BoolKind:
		return "false"
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind,
		protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint32Kind, protoreflect.Fixed32Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind,
		protoreflect.FloatKind, protoreflect.DoubleKind:
		return "0"
	case protoreflect.StringKind:
		return `""`
	case protoreflect.BytesKind:
		return "nil"
	case protoreflect.MessageKind, protoreflect.GroupKind:
		return "nil"
	case protoreflect.EnumKind:
		// 获取枚举的第一个值作为默认值
		//enum := desc.Enum()
		//if enum.Values().Len() > 0 {
		//	return fmt.Sprintf("%s_%s", enum.Name(), enum.Values().Get(0).Name())
		//}
		return "0"
	}
	return "nil"
}

func isNeedNewValue(desc protoreflect.FieldDescriptor) bool {
	if desc.IsMap() || desc.IsList() {
		return true
	}
	if desc.Kind() == protoreflect.BytesKind {
		return true
	}
	return desc.Kind() == protoreflect.MessageKind || desc.Kind() == protoreflect.GroupKind
}

func isMessageValue(desc protoreflect.FieldDescriptor) bool {
	if desc.IsMap() || desc.IsList() {
		return false
	}
	return desc.Kind() == protoreflect.MessageKind || desc.Kind() == protoreflect.GroupKind
}

func parseTcaplusFieldOptions(field *protogen.Field) []func(desc *FieldDesc) {
	fns := make([]func(desc *FieldDesc), 0)
	if opt := proto.GetExtension(field.Desc.Options(), options.E_TcaplusDesc); opt != nil {
		if s := opt.(string); s != "" {
			fns = append(fns, func(desc *FieldDesc) {
				desc.Tcaplus.TcaplusDesc = OptionalValue{Value: stringEscape(s), Valid: true}
				desc.Tcaplus.HasValue = true
			})
		}
	}
	if opt := proto.GetExtension(field.Desc.Options(), options.E_TcaplusSize); opt != nil {
		if i := opt.(uint32); i != 0 {
			fns = append(fns, func(desc *FieldDesc) {
				desc.Tcaplus.TcaplusSize = OptionalInt{Value: int(i), Valid: true}
				desc.Tcaplus.HasValue = true
			})
		}
	}
	if opt := proto.GetExtension(field.Desc.Options(), options.E_TcaplusCrypto); opt != nil {
		if b := opt.(bool); b {
			fns = append(fns, func(desc *FieldDesc) {
				desc.Tcaplus.TcaplusCrypto = OptionalBool{Value: b, Valid: true}
				desc.Tcaplus.HasValue = true
			})
		}
	}
	return fns
}

// parseOrmFieldOptions 解析字段级别的ORM选项
func parseOrmFieldOptions(field *protogen.Field) []func(desc *FieldDesc) {
	//opts := make(map[string]string)
	fns := make([]func(desc *FieldDesc), 0)
	// 解析orm option选项
	if opt := proto.GetExtension(field.Desc.Options(), options.E_Tags); opt != nil {
		if s := opt.(string); s != "" {
			fns = append(fns, func(desc *FieldDesc) {
				desc.Tags = OptionalString{Value: stringEscape(s), Valid: true}
				//if strings.Contains(strings.ToLower(s), "primary_key") ||
				//	strings.Contains(strings.ToLower(s), "primarykey") {
				//	desc.OrmOptions.HasPrimaryKey = true
				//}
				//if strings.Contains(strings.ToLower(s), "index:") ||
				//	strings.Contains(strings.ToLower(s), "unique_index:") {
				//	desc.OrmOptions.HasIndex = true
				//}
			})
		}
	}
	if opt := proto.GetExtension(field.Desc.Options(), options.E_PrimaryKey); opt != nil {
		if b := opt.(bool); b {
			fns = append(fns, func(desc *FieldDesc) {
				desc.PrimaryKey = b
				desc.OrmOptions.HasPrimaryKey = b
			})
		}
	}
	if opt := proto.GetExtension(field.Desc.Options(), options.E_SkipSetDefault); opt != nil {
		if b := opt.(bool); b {
			fns = append(fns, func(desc *FieldDesc) {
				desc.SkipSetDefault = b
			})
		}
	}
	if opt := proto.GetExtension(field.Desc.Options(), options.E_Index); opt != nil {
		if b := opt.(bool); b {
			fns = append(fns, func(desc *FieldDesc) {
				desc.OrmOptions.HasIndex = b
			})
		}
	}
	if opt := proto.GetExtension(field.Desc.Options(), options.E_ShardingKey); opt != nil {
		if b := opt.(bool); b {
			fns = append(fns, func(desc *FieldDesc) {
				desc.OrmOptions.HasShardingKey = b
			})
		}
	}
	if proto.HasExtension(field.Desc.Options(), options.E_ForeignKey) {
		opt := proto.GetExtension(field.Desc.Options(), options.E_ForeignKey)
		if s := strings.TrimSpace(opt.(string)); s != "" {
			fns = append(fns, func(desc *FieldDesc) {
				desc.OrmOptions.HasForeignKey = true
				desc.OrmOptions.ForeignKeySpec = s
			})
		}
	}
	// is version field
	if field.Desc.Kind() == protoreflect.Int64Kind {
		if strings.ToLower(string(field.Desc.Name())) == "version" {
			fns = append(fns, func(desc *FieldDesc) {
				desc.OrmOptions.HasVersion = true
			})
		}
	}
	// is google.protobuf.* message, skip set default
	if field.Desc.Kind() == protoreflect.MessageKind {
		msgName := string(field.Desc.Message().FullName())
		if strings.HasPrefix(msgName, "google.protobuf.") {
			fns = append(fns, func(desc *FieldDesc) {
				desc.SkipSetDefault = true
			})
		}
	}
	return fns
}

func generateTcaplusOptionProto(gen *protogen.Plugin, file *protogen.File, dbType string) error {

	filename := outputTcaplusOptionProtoFile
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	// 解析模板
	tmpl := template.New("to").Funcs(funcMap)

	data := struct {
		Version       string
		ProtocVersion string
		Package       string
		GoPackage     string
		GoImportPath  string
		DBType        string
		Source        string
	}{
		Version:       version,
		ProtocVersion: protocVersion,
		Package:       string(file.Desc.Package()),
		GoPackage:     string(file.GoPackageName),
		GoImportPath:  string(file.GoImportPath),
		DBType:        dbType,
		Source:        file.Desc.Path(),
	}

	tmpl, err := tmpl.Parse(tcaplusOptionTemplate)
	if err != nil {
		return fmt.Errorf("parse tcaplus option template failed: %w", err)
	}

	// 执行模板
	if err := tmpl.Execute(g, data); err != nil {
		return fmt.Errorf("execute template failed: %w", err)
	}
	return nil
}

// generateOutputFilename 生成输出文件名
func generateOutputFilename(file *protogen.File, dbType DBType) string {
	// 获取原始文件名（不带扩展名）
	origFilename := strings.TrimSuffix(filepath.Base(file.Desc.Path()), ".proto")
	suffix := dbType.Suffix()
	// 生成新文件名：原始文件名 + .dbtype.proto
	return fmt.Sprintf("%s/%s/%s/%s%s.proto", outputBaseInternalDir, dbType, outputProtoDir, suffix, origFilename)
}
