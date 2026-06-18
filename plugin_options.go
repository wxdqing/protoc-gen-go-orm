package main

import (
	"strings"

	"google.golang.org/protobuf/compiler/protogen"
)

// pluginDBTypes limits code generation to these drivers when set via protoc parameter
// db_types=pgsql,redis (comma-separated). Empty means all supported drivers.
var pluginDBTypes []DBType

func initPluginOptions(gen *protogen.Plugin) {
	pluginDBTypes = parseDBTypesParam(gen.Request.GetParameter())
}

func parseDBTypesParam(param string) []DBType {
	param = strings.TrimSpace(param)
	if param == "" {
		return nil
	}
	seen := make(map[DBType]struct{})
	var out []DBType
	for _, part := range strings.Split(param, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if key, val, ok := strings.Cut(part, "="); ok && strings.TrimSpace(key) == "db_types" {
			part = val
		}
		for _, name := range strings.Split(part, ",") {
			name = strings.TrimSpace(name)
			if name == "" {
				continue
			}
			t := DBType(name)
			if err := t.Validate(); err != nil {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	return out
}

func targetDBTypes() []DBType {
	if len(pluginDBTypes) == 0 {
		return supportedDBTypes()
	}
	return append([]DBType(nil), pluginDBTypes...)
}

func targetDBTypeSet() map[DBType]bool {
	types := targetDBTypes()
	if len(pluginDBTypes) == 0 {
		return nil
	}
	set := make(map[DBType]bool, len(types))
	for _, t := range types {
		set[t] = true
	}
	return set
}

func isTargetDBType(t DBType) bool {
	set := targetDBTypeSet()
	if set == nil {
		return true
	}
	return set[t]
}
