package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
)

var generatedContextFiles = map[DBType]bool{}

func generateMethods(gen *protogen.Plugin, file *protogen.File, messages []MessageDesc, enums []EnumDesc, dbType DBType, protocVersion string) error {
	filename := methodFileName(file, dbType)
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	var filteredMessages []MessageDesc
	for _, msg := range messages {
		if msg.OrmOptions.IsTable {
			filteredMessages = append(filteredMessages, msg)
		}
	}
	data := struct {
		Version       string
		ProtocVersion string
		Package       string
		GoPackage     string
		Messages      []MessageDesc
		AllMessages   []MessageDesc
		DBType        string
		Source        string
		Enums         []EnumDesc
	}{
		Version:       version,
		ProtocVersion: protocVersion,
		Package:       string(file.Desc.Package()),
		GoPackage:     string(file.GoPackageName),
		Messages:      filteredMessages,
		AllMessages:   messages,
		DBType:        string(dbType),
		Source:        file.Desc.Path(),
		Enums:         enums,
	}

	// 解析模板

	tmpl := template.New("method").Funcs(funcMap)

	tmpl, err := tmpl.Parse(methodsTemplate)
	if err != nil {
		return fmt.Errorf("parse method template failed: %w", err)
	}

	// 执行模板
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("execute template failed: %w", err)
	}

	// 写入生成的内容
	generateHeader(gen, g, file, dbType)
	g.P("package ", dbType)
	g.P("")
	if len(filteredMessages) > 0 {
		g.P("import ", file.GoImportPath)
		g.QualifiedGoIdent(protoPackage.Ident(""))
		g.QualifiedGoIdent(reflectPackage.Ident(""))
		//if dbType == DBTypeTcaplus {
		//	g.QualifiedGoIdent(mapstructurePackage.Ident(""))
		//}
		g.P("")
		//g.QualifiedGoIdent(file.GoImportPath.Ident("example"))
		//g.P("// 1", file.GoImportPath)
		//g.P("// 2", file.GoImportPath.Ident(""))
	}
	g.P(strings.Trim(buf.String(), "\r\n"))
	return nil
}

func methodFileName(file *protogen.File, dbType DBType) string {
	// 获取原始文件名（不带扩展名）
	origFilename := strings.TrimSuffix(filepath.Base(file.Desc.Path()), ".proto")
	return fmt.Sprintf("internal/%s/%s%s_methods.pb.go", dbType, dbType.Suffix(), origFilename)
}

func generateContext(gen *protogen.Plugin, file *protogen.File, dbType DBType) error {
	if generatedContextFiles[dbType] {
		return nil
	}
	generatedContextFiles[dbType] = true
	filename := fmt.Sprintf("internal/%s/orm_context.pb.go", dbType)
	g := gen.NewGeneratedFile(filename, file.GoImportPath)
	hooksTmpl, err := template.New("context").Funcs(funcMap).Parse(contextTemplate)
	if err != nil {
		return fmt.Errorf("parse context hooks template failed: %w", err)
	}
	codecTmpl, err := template.New("codec").Funcs(funcMap).Parse(jsonCodecTemplate)
	if err != nil {
		return fmt.Errorf("parse json codec template failed: %w", err)
	}
	var hooksBuf, codecBuf bytes.Buffer
	if err := hooksTmpl.Execute(&hooksBuf, nil); err != nil {
		return fmt.Errorf("execute context hooks template failed: %w", err)
	}
	if err := codecTmpl.Execute(&codecBuf, nil); err != nil {
		return fmt.Errorf("execute json codec template failed: %w", err)
	}
	generateHeader(gen, g, file, dbType)
	g.P("package ", dbType)
	g.P("")
	g.P("import (")
	g.P(`"bytes"`)
	g.P(`"encoding/json"`)
	g.P(`"fmt"`)
	g.P(`"strings"`)
	g.P(`"google.golang.org/protobuf/encoding/protojson"`)
	g.P(`"google.golang.org/protobuf/encoding/protowire"`)
	g.P(`"google.golang.org/protobuf/proto"`)
	g.P(`"google.golang.org/protobuf/reflect/protoreflect"`)
	g.P(")")
	g.P("")
	g.P(strings.Trim(hooksBuf.String(), "\r\n"))
	g.P("")
	g.P(strings.Trim(codecBuf.String(), "\r\n"))
	return nil
}
