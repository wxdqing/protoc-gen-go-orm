package main

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
)

func generateValues(gen *protogen.Plugin, file *protogen.File, messages []MessageDesc, enums []EnumDesc) error {
	filename := valueFileName(file)
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
		Source        string
		Enums         []EnumDesc
		G             *protogen.GeneratedFile
	}{
		Version:       version,
		ProtocVersion: protocVersion,
		Package:       string(file.Desc.Package()),
		GoPackage:     string(file.GoPackageName),
		Messages:      messages,
		Source:        file.Desc.Path(),
		Enums:         enums,
		G:             g,
	}

	// 解析模板
	tmpl := template.New("values").Funcs(funcMap)

	tmpl, err := tmpl.Parse(valuesTemplate)
	if err != nil {
		return fmt.Errorf("parse value template failed: %w", err)
	}

	// 执行模板
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		return fmt.Errorf("execute template failed: %w", err)
	}

	// 写入生成的内容
	generateHeader(gen, g, file, "")
	g.P("package ", file.Desc.Package())
	g.P("")
	if len(filteredMessages) > 0 {
		g.QualifiedGoIdent(protoPackage.Ident(""))
		g.P("")
	}
	g.P(strings.Trim(buf.String(), "\r\n"))
	return nil
}

func valueFileName(file *protogen.File) string {
	// 获取原始文件名（不带扩展名）
	origFilename := strings.TrimSuffix(filepath.Base(file.Desc.Path()), ".proto")
	return fmt.Sprintf("%s_values.pb.go", origFilename)
}
