package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// TagInfo 存储tag信息
type TagInfo struct {
	FilePath    string
	MessageTags map[string]map[string]string // messageName -> fieldName -> tagString
}

type FieldTag struct {
	Pk    string `json:"pk"`
	Index string `json:"index"`
	JSON  string `json:"json"`
	Blob  string `json:"blob"`
}

func defaultFieldTagsForPath(filePath string) map[string]string {
	dataType := "blob"
	if strings.Contains(strings.ToLower(filePath), string(DBTypePostgresSQL)) ||
		strings.Contains(filepath.ToSlash(filePath), "/pgsql/") {
		dataType = "bytea"
	}
	return map[string]string{
		"Data":      fmt.Sprintf(`gorm:"column:data;type:%s"`, dataType),
		"CreatedAt": `gorm:"column:created_at;autoCreateTime;<-:create"`,
		"UpdatedAt": `gorm:"column:updated_at;autoUpdateTime"`,
	}
}

func appendGormTags(dir string) error {
	// 遍历目录中的所有.pb.go文件
	// exists check
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return err
	}
	// travel dir
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		if filepath.Ext(file.Name()) == ".go" && strings.HasSuffix(file.Name(), ".pb.go") {
			path := filepath.Join(dir, file.Name())
			fmt.Println("Processing file:", path)
			modified, err := appendTagsToFile(path)

			fmt.Println("Processed file:", path, " modified:", modified, " err:", err)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func appendTagsToFile(filePath string) (bool, error) {
	// 读取生成的.go文件
	content, err := os.ReadFile(filePath)
	if err != nil {
		return false, fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	// 解析Go文件AST
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "", content, parser.ParseComments)
	if err != nil {
		return false, fmt.Errorf("failed to parse file %s: %w", filePath, err)
	}

	// 修改AST添加tags
	info := &TagInfo{
		FilePath:    filePath,
		MessageTags: make(map[string]map[string]string),
	}
	modified := modifyAST(file, info)
	if !modified {
		return false, nil // 如果没有修改则直接返回
	}

	// 生成修改后的代码
	var buf bytes.Buffer
	if err := format.Node(&buf, fset, file); err != nil {
		return false, fmt.Errorf("failed to format modified file %s: %w", filePath, err)
	}

	// 写回文件
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return false, fmt.Errorf("failed to write modified file %s: %w", filePath, err)
	}

	fmt.Fprintf(os.Stderr, "Successfully added tags to %s\n", filePath)
	return true, nil
}

func modifyAST(file *ast.File, info *TagInfo) bool {
	modified := false

	// 遍历所有类型声明
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.TYPE {
			continue
		}

		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			structType, ok := typeSpec.Type.(*ast.StructType)
			if !ok {
				continue
			}

			hasData := false
			hasCreatedAt := false
			hasUpdatedAt := false
			tagsOfStruct := make(map[string]string)
			for _, f := range structType.Fields.List {
				if f.Names[0].Name == "Data" /* && fmt.Sprintf("%s", f.Type) == "[]byte"*/ {
					arrayType, ok := f.Type.(*ast.ArrayType)
					if !ok {
						continue
					}
					ident, ok := arrayType.Elt.(*ast.Ident)
					if !ok {
						continue
					}
					if ident.Name != "byte" {
						continue
					}
					hasData = true
				}
				if f.Names[0].Name == "CreatedAt" && fmt.Sprintf("%s", f.Type) == "int64" {
					hasCreatedAt = true
				}
				if f.Names[0].Name == "UpdatedAt" && fmt.Sprintf("%s", f.Type) == "int64" {
					hasUpdatedAt = true
				}
				// parse comment for field tag
				if str := strings.Trim(f.Comment.Text(), " "); str != "" {
					obj := FieldTag{}
					err := json.Unmarshal([]byte(extractFieldTagJSON(str)), &obj)
					if err != nil {
						fmt.Println("failed to unmarshal field tag comment:", err, " struct:", typeSpec.Name.Name, " field:", f.Names[0].Name, " comment:", str)
						continue
					}
					if obj.Pk == "" && obj.Index == "" && obj.JSON == "" && obj.Blob == "" {
						continue
					}
					tagsOfStruct[f.Names[0].Name] = buildGormTag(obj)
				}

			}
			if len(tagsOfStruct) == 0 {
				continue
			}
			if hasData && hasCreatedAt && hasUpdatedAt {
				for k, v := range defaultFieldTagsForPath(info.FilePath) {
					tagsOfStruct[k] = v
				}
			}
			//messageName := typeSpec.Name.Name
			//defaultFieldTags, exists := info.MessageTags[messageName]
			//if !exists {
			//	continue
			//}

			// 修改结构体字段的tags
			if modifyStructTags(structType, tagsOfStruct) {
				modified = true
			}
		}
	}

	return modified
}

func extractFieldTagJSON(s string) string {
	start := strings.Index(s, "{")
	if start < 0 {
		return s
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inString {
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			continue
		}
		if inString {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return s[start:]
}

func buildGormTag(tag FieldTag) string {
	parts := make([]string, 0, 3)
	if tag.Pk != "" {
		parts = append(parts, tag.Pk)
	}
	if tag.Index != "" {
		parts = append(parts, tag.Index)
	}
	if tag.JSON != "" {
		parts = append(parts, tag.JSON)
	}
	if tag.Blob != "" {
		parts = append(parts, tag.Blob)
	}
	if len(parts) == 0 {
		return ""
	}
	return fmt.Sprintf(`gorm:"%s"`, strings.Join(parts, ";"))
}

func modifyStructTags(structType *ast.StructType, fieldTags map[string]string) bool {
	modified := false

	for _, field := range structType.Fields.List {
		if len(field.Names) == 0 {
			continue
		}

		fieldName := field.Names[0].Name
		customTag, exists := fieldTags[fieldName]
		if !exists {
			continue
		}

		// 处理现有的tag
		currentTag := ""
		if field.Tag != nil {
			currentTag = strings.Trim(field.Tag.Value, "`")
		}

		// 合并tags
		mergedTag := mergeTags(currentTag, customTag)

		// 创建或更新tag
		if field.Tag == nil {
			field.Tag = &ast.BasicLit{
				Kind:  token.STRING,
				Value: "`" + mergedTag + "`",
			}
		} else {
			field.Tag.Value = "`" + mergedTag + "`"
		}

		modified = true
	}

	return modified
}

func mergeTags(currentTag, customTag string) string {
	if currentTag == "" {
		return customTag
	}

	// 解析现有的tags
	currentTags := parseTagString(currentTag)
	customTags := parseTagString(customTag)

	// 合并：自定义tags会覆盖现有的同名tags
	for key, value := range customTags {
		currentTags[key] = value
	}

	// 重新格式化为字符串
	return formatTagMap(currentTags)
}

func parseTagString(tagStr string) map[string]string {
	result := make(map[string]string)
	if tagStr == "" {
		return result
	}

	// 按空格分割不同的tag
	parts := strings.Fields(tagStr)
	for _, part := range parts {
		// 处理 key:"value" 格式
		if idx := strings.Index(part, ":\""); idx > 0 {
			key := part[:idx]
			value := strings.Trim(part[idx+2:], `"`)
			result[key] = value
		} else if idx := strings.Index(part, ":"); idx > 0 {
			// 处理 key:value 格式（无引号）
			key := part[:idx]
			value := part[idx+1:]
			result[key] = value
		} else {
			// 只有key没有value
			result[part] = ""
		}
	}

	return result
}

func formatTagMap(tags map[string]string) string {
	var parts []string
	// sort keys
	sortedKeys := make([]string, 0, len(tags))
	for key := range tags {
		sortedKeys = append(sortedKeys, key)
	}
	sort.SliceStable(sortedKeys, func(i, j int) bool {
		// reversed alphabetical order (protobuf->json->gorm)
		return sortedKeys[i] > sortedKeys[j]
	})

	for _, key := range sortedKeys {
		value := tags[key]
		if value == "" {
			parts = append(parts, key)
		} else {
			parts = append(parts, key+":\""+value+"\"")
		}
	}
	return strings.Join(parts, " ")
}
