package main

import "testing"

func TestParsePartialIndexSpecExisting(t *testing.T) {
	idx, ok := parsePartialIndexSpec(`UNIQUE uq_registry_global(category,code) WHERE scope='*'`)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if !idx.Unique || idx.Name != "uq_registry_global" || idx.Where != "scope='*'" {
		t.Fatalf("unexpected: %+v", idx)
	}
}

func TestParseForeignKeySpecExisting(t *testing.T) {
	fk, ok := parseForeignKeySpec("user_id", `auth_user(id) ON DELETE CASCADE`)
	if !ok {
		t.Fatal("expected parse ok")
	}
	if fk.RefTable != "auth_user" || fk.RefColumn != "id" || fk.OnDelete != "CASCADE" {
		t.Fatalf("unexpected: %+v", fk)
	}
}

func TestExtraMigrationsForMessage(t *testing.T) {
	msg := MessageDesc{
		Name:      "AuthRefreshToken",
		TableName: "auth_refresh_token",
		OrmOptions: MessageOrmOptions{
			IsTable: true,
		},
		Fields: []FieldDesc{{
			Name: "user_id",
			OrmOptions: &FieldOrmOptions{
				HasForeignKey:  true,
				ForeignKeySpec: "auth_user(id) ON DELETE CASCADE",
			},
		}},
	}
	stmts := extraMigrationsForMessage(msg)
	if len(stmts) != 1 {
		t.Fatalf("expected 1 migration, got %d: %v", len(stmts), stmts)
	}
}
