package main

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"google.golang.org/protobuf/compiler/protogen"
)

func generateMetadata(gen *protogen.Plugin, file *protogen.File, messages []MessageDesc, enums []EnumDesc) error {
	filename := metaFileName()
	g := gen.NewGeneratedFile(filename, file.GoImportPath)

	var filteredMessages []MessageDesc
	directMessages := make(map[string][]MessageDesc)
	for _, msg := range messages {
		if msg.OrmOptions.IsTable {
			filteredMessages = append(filteredMessages, msg)
		}
	}
	for _, msg := range messages {
		if strings.HasPrefix(msg.FilePath, string(DBTypeTcaplus)+"_tb") {
			dm := directMessages[string(DBTypeTcaplus)]
			dm = append(dm, msg)
			directMessages[string(DBTypeTcaplus)] = dm
		}
	}
	data := struct {
		Version        string
		ProtocVersion  string
		Package        string
		GoPackage      string
		Messages       []MessageDesc
		DirectMessages map[string][]MessageDesc
		DBTypes        []DBType
		Source         string
		Enums          []EnumDesc
	}{
		Version:        version,
		ProtocVersion:  protocVersion,
		Package:        string(file.Desc.Package()),
		GoPackage:      string(file.GoPackageName),
		Messages:       filteredMessages,
		DirectMessages: directMessages,
		DBTypes:        activeDBTypes(filteredMessages),
		Source:         file.Desc.Path(),
		Enums:          enums,
	}

	// 解析模板
	tmpl := template.New("metadata").Funcs(funcMap)

	tmpl, err := tmpl.Parse(metadataTemplate)
	if err != nil {
		return fmt.Errorf("parse metadata template failed: %w", err)
	}

	// 执行模板
	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, data); err != nil {
		panic(fmt.Errorf("execute template failed: %w", err))
	}

	// 写入生成的内容
	generateHeader(gen, g, file, "")
	g.P("package ", "metadata")
	g.P("")

	// imports
	g.QualifiedGoIdent(protoPackage.Ident(""))
	if len(filteredMessages) > 0 {
		for _, dbType := range activeDBTypes(filteredMessages) {
			g.QualifiedGoIdent((file.GoImportPath + "/" + outputBaseInternalDir + "/" + protogen.GoImportPath(dbType)).Ident(""))
		}
	}

	if len(directMessages) > 0 {
		g.P("import ", file.GoPackageName, file.GoImportPath)
		//g.QualifiedGoIdent(file.GoImportPath.Ident(""))
	}
	// content
	g.P(strings.Trim(buf.String(), "\r\n"))
	return nil
}

func metaFileName() string {
	return "metadata/meta.pb.go"
}
