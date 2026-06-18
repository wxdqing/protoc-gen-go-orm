package main

import (
	"fmt"
	"regexp"
	"strings"
)

// ForeignKey describes a PostgreSQL foreign key constraint.
type ForeignKey struct {
	Column     string
	RefTable   string
	RefColumn  string
	OnDelete   string
	Constraint string
}

var foreignKeyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*)\(([^)]+)\)(?:\s+ON\s+DELETE\s+([A-Za-z_ ]+))?$`)

func parseForeignKeySpec(column, spec string) (ForeignKey, bool) {
	spec = strings.TrimSpace(spec)
	column = strings.TrimSpace(column)
	if spec == "" || column == "" {
		return ForeignKey{}, false
	}
	m := foreignKeyRe.FindStringSubmatch(spec)
	if m == nil {
		return ForeignKey{}, false
	}
	refCol := strings.TrimSpace(m[2])
	if refCol == "" {
		refCol = "id"
	}
	onDelete := strings.TrimSpace(m[3])
	if onDelete == "" {
		onDelete = "NO ACTION"
	}
	constraint := fmt.Sprintf("fk_%s_%s", column, m[1])
	return ForeignKey{
		Column:     column,
		RefTable:   m[1],
		RefColumn:  refCol,
		OnDelete:   strings.ToUpper(onDelete),
		Constraint: constraint,
	}, true
}

func collectForeignKeys(msg MessageDesc) []ForeignKey {
	var out []ForeignKey
	seen := make(map[string]struct{})
	for _, field := range msg.Fields {
		if field.OrmOptions == nil || !field.OrmOptions.HasForeignKey {
			continue
		}
		fk, ok := parseForeignKeySpec(field.Name, field.OrmOptions.ForeignKeySpec)
		if !ok {
			continue
		}
		if _, dup := seen[fk.Constraint]; dup {
			continue
		}
		seen[fk.Constraint] = struct{}{}
		out = append(out, fk)
	}
	return out
}

func foreignKeySQL(table string, fk ForeignKey) string {
	return fmt.Sprintf(
		`DO $$ BEGIN ALTER TABLE %s ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s(%s) ON DELETE %s; EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
		table, fk.Constraint, toSnakeCase(fk.Column), fk.RefTable, fk.RefColumn, fk.OnDelete,
	)
}

func foreignKeyMigrations(table string, keys []ForeignKey) []string {
	if len(keys) == 0 {
		return nil
	}
	out := make([]string, 0, len(keys))
	for _, fk := range keys {
		out = append(out, foreignKeySQL(table, fk))
	}
	return out
}
