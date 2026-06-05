package main

import (
	"reflect"
	"testing"
)

func TestDedupeGormSubparts(t *testing.T) {
	got := dedupeGormSubparts([]string{
		"primary_key;column:id;autoIncrement:false",
		"index:idx_player;column:id",
	})
	want := []string{"primary_key", "column:id", "autoIncrement:false", "index:idx_player"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestDedupeGormSubpartsKeepsDistinctColumns(t *testing.T) {
	got := dedupeGormSubparts([]string{
		"primary_key;column:id",
		"type:json;column:profile",
	})
	want := []string{"primary_key", "column:id", "type:json", "column:profile"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestDedupeGormSubpartsEmpty(t *testing.T) {
	if got := dedupeGormSubparts(nil); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}
