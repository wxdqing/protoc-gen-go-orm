package main

import (
	"testing"
)

func TestParseCompositeIndexSpec(t *testing.T) {
	idx, ok := parseCompositeIndexSpec("idx_lister_rid_id(rid,id)")
	if !ok || idx.Name != "idx_lister_rid_id" || len(idx.Columns) != 2 || idx.Columns[0] != "rid" || idx.Columns[1] != "id" {
		t.Fatalf("parse: %+v ok=%v", idx, ok)
	}
	if _, ok := parseCompositeIndexSpec("bad"); ok {
		t.Fatal("expected invalid spec")
	}
}

func TestCollectCompositeIndexes_AutoShardingPlusIndex(t *testing.T) {
	msg := MessageDesc{
		Name: "FieldsPlayer",
		OrmOptions: MessageOrmOptions{
			ShardingKeyField: OptionalValue{Valid: true, Value: FieldDesc{Name: "id"}},
		},
		Fields: []FieldDesc{
			{Name: "id", OrmOptions: &FieldOrmOptions{HasPrimaryKey: true, HasShardingKey: true}},
			{Name: "name", OrmOptions: &FieldOrmOptions{HasIndex: true}},
		},
	}
	indexes := collectCompositeIndexes(msg)
	if len(indexes) != 1 || indexes[0].Name != "idx_fields_player_name" {
		t.Fatalf("indexes: %+v", indexes)
	}
	if got := joinFieldsIndex(msg); len(got) != 1 || got[0] != "idx_fields_player_name(id,name)" {
		t.Fatalf("join: %v", got)
	}
	tag := indexTagForField(msg, msg.Fields[0])
	if tag != "idx_fields_player_name;column:id" {
		t.Fatalf("id tag: %q", tag)
	}
	tag = indexTagForField(msg, msg.Fields[1])
	if tag != "idx_fields_player_name;column:name" {
		t.Fatalf("name tag: %q", tag)
	}
}

func TestCollectCompositeIndexes_ExplicitOption(t *testing.T) {
	msg := MessageDesc{
		Name: "Lister",
		OrmOptions: MessageOrmOptions{
			CompositeIndexSpecs: []string{"idx_lister_rid_id(rid,id)"},
		},
		Fields: []FieldDesc{
			{Name: "rid", OrmOptions: &FieldOrmOptions{HasPrimaryKey: true, HasIndex: true}},
			{Name: "id", OrmOptions: &FieldOrmOptions{HasPrimaryKey: true}},
		},
	}
	got := joinFieldsIndex(msg)
	want := map[string]bool{
		"idx_lister_rid_id(rid,id)": true,
		"idx_lister_rid(rid)":       true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want keys %v", got, want)
	}
	for _, s := range got {
		if !want[s] {
			t.Fatalf("unexpected index %q in %v", s, got)
		}
	}
}
