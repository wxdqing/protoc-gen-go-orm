package main

import (
	"fmt"
	"regexp"
	"strings"
)

// PartialIndex describes a PostgreSQL partial (filtered) index.
type PartialIndex struct {
	Unique  bool
	Name    string
	Columns string // raw column list inside parentheses, e.g. "scope, version DESC"
	Where   string
}

var partialIndexRe = regexp.MustCompile(`(?i)^(UNIQUE\s+)?([A-Za-z_][A-Za-z0-9_]*)\(([^)]*)\)(?:\s+WHERE\s+(.+))?$`)

func parsePartialIndexSpec(spec string) (PartialIndex, bool) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return PartialIndex{}, false
	}
	m := partialIndexRe.FindStringSubmatch(spec)
	if m == nil {
		return PartialIndex{}, false
	}
	cols := strings.TrimSpace(m[3])
	if cols == "" {
		return PartialIndex{}, false
	}
	return PartialIndex{
		Unique:  strings.TrimSpace(m[1]) != "",
		Name:    m[2],
		Columns: cols,
		Where:   strings.TrimSpace(m[4]),
	}, true
}

func collectPartialIndexes(msg MessageDesc) []PartialIndex {
	var out []PartialIndex
	seen := make(map[string]struct{})
	for _, spec := range msg.OrmOptions.PartialIndexSpecs {
		idx, ok := parsePartialIndexSpec(spec)
		if !ok {
			continue
		}
		key := idx.Name + "|" + idx.Columns + "|" + idx.Where
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, idx)
	}
	return out
}

func partialIndexSQL(table string, idx PartialIndex) string {
	kind := "INDEX"
	if idx.Unique {
		kind = "UNIQUE INDEX"
	}
	sql := fmt.Sprintf("CREATE %s IF NOT EXISTS %s ON %s (%s)", kind, idx.Name, table, idx.Columns)
	if idx.Where != "" {
		sql += " WHERE " + idx.Where
	}
	return sql
}

func partialIndexMigrations(table string, indexes []PartialIndex) []string {
	if len(indexes) == 0 {
		return nil
	}
	out := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		out = append(out, partialIndexSQL(table, idx))
	}
	return out
}
