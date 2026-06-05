package main

import (
	"fmt"
	"regexp"
	"strings"
)

// CompositeIndex 描述多列联合索引（GORM 同 index 名；tcaplus idx_name(c1,c2)）。
type CompositeIndex struct {
	Name    string
	Columns []string // proto 字段名
}

var compositeIndexSpecRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\(([^)]+)\)$`)

func parseCompositeIndexSpec(spec string) (CompositeIndex, bool) {
	spec = strings.TrimSpace(spec)
	m := compositeIndexSpecRe.FindStringSubmatch(spec)
	if m == nil {
		return CompositeIndex{}, false
	}
	parts := strings.Split(m[2], ",")
	cols := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		cols = append(cols, p)
	}
	if len(cols) == 0 {
		return CompositeIndex{}, false
	}
	return CompositeIndex{Name: m[1], Columns: cols}, true
}

func shardingKeyFieldName(msg MessageDesc) string {
	if !msg.OrmOptions.ShardingKeyField.Valid {
		return ""
	}
	f, ok := msg.OrmOptions.ShardingKeyField.Value.(FieldDesc)
	if !ok {
		return ""
	}
	return f.Name
}

func collectCompositeIndexes(msg MessageDesc) []CompositeIndex {
	seen := make(map[string]struct{})
	var out []CompositeIndex
	add := func(idx CompositeIndex) {
		if len(idx.Columns) == 0 {
			return
		}
		key := idx.Name + "|" + strings.Join(idx.Columns, ",")
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, idx)
	}

	for _, spec := range msg.OrmOptions.CompositeIndexSpecs {
		if idx, ok := parseCompositeIndexSpec(spec); ok {
			add(idx)
		}
	}

	sk := shardingKeyFieldName(msg)
	for _, field := range msg.Fields {
		if !field.OrmOptions.HasIndex {
			continue
		}
		cols := []string{field.Name}
		if sk != "" && sk != field.Name {
			cols = []string{sk, field.Name}
		}
		name := fmt.Sprintf("idx_%s_%s", toSnakeCase(msg.Name), field.Name)
		add(CompositeIndex{Name: name, Columns: cols})
	}
	return out
}

func compositeIndexNameForField(msg MessageDesc, fieldName string) string {
	for _, idx := range collectCompositeIndexes(msg) {
		if len(idx.Columns) < 2 {
			continue
		}
		for _, col := range idx.Columns {
			if col == fieldName {
				return idx.Name
			}
		}
	}
	return ""
}

func indexTagForField(msg MessageDesc, field FieldDesc) string {
	col := toSnakeCase(field.Name)
	if name := compositeIndexNameForField(msg, field.Name); name != "" {
		return fmt.Sprintf("%s;column:%s", name, col)
	}
	if field.OrmOptions != nil && field.OrmOptions.HasIndex {
		return fmt.Sprintf("idx_%s;column:%s", col, col)
	}
	return ""
}

func joinFieldsIndex(msg MessageDesc) []string {
	indexes := collectCompositeIndexes(msg)
	out := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		cols := make([]string, len(idx.Columns))
		for i, c := range idx.Columns {
			cols[i] = toSnakeCase(c)
		}
		out = append(out, fmt.Sprintf("%s(%s)", idx.Name, strings.Join(cols, ",")))
	}
	return out
}
