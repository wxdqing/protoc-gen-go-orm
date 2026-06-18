package main

import "testing"

func TestParseDBTypesParam(t *testing.T) {
	got := parseDBTypesParam("db_types=pgsql,redis")
	if len(got) != 2 || got[0] != DBTypePostgresSQL || got[1] != DBTypeRedis {
		t.Fatalf("parseDBTypesParam() = %v, want [pgsql redis]", got)
	}
	if parseDBTypesParam("") != nil {
		t.Fatal("empty param should return nil")
	}
	if got := parseDBTypesParam("db_types=bad,pgsql"); len(got) != 1 || got[0] != DBTypePostgresSQL {
		t.Fatalf("parseDBTypesParam(bad,pgsql) = %v, want [pgsql]", got)
	}
}

func TestTargetDBTypesDefault(t *testing.T) {
	pluginDBTypes = nil
	if len(targetDBTypes()) != len(supportedDBTypes()) {
		t.Fatalf("default targetDBTypes len = %d, want %d", len(targetDBTypes()), len(supportedDBTypes()))
	}
	pluginDBTypes = []DBType{DBTypePostgresSQL, DBTypeRedis}
	defer func() { pluginDBTypes = nil }()
	got := targetDBTypes()
	if len(got) != 2 || got[0] != DBTypePostgresSQL || got[1] != DBTypeRedis {
		t.Fatalf("targetDBTypes() = %v", got)
	}
}

func TestActiveDBTypesRespectsPluginFilter(t *testing.T) {
	pluginDBTypes = []DBType{DBTypePostgresSQL, DBTypeRedis}
	defer func() { pluginDBTypes = nil }()
	msgs := []MessageDesc{{
		Name: "Tenant",
		OrmOptions: MessageOrmOptions{
			IsTable:   true,
			DbDrivers: []string{"pgsql"},
		},
	}}
	got := activeDBTypes(msgs)
	if len(got) != 1 || got[0] != DBTypePostgresSQL {
		t.Fatalf("activeDBTypes() = %v, want [pgsql]", got)
	}
}
