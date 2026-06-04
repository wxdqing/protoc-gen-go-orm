package main

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/wxdqing/protoc-gen-go-orm/options"
)

func TestWarnFieldsModeOnKV_EmitsForRedis(t *testing.T) {
	desc := newMessageDescWithOrmDefaults()
	desc.Name = "Player"
	desc.OrmOptions.IsTable = true
	desc.OrmOptions.TableStoreMode = options.TableStoreMode_TABLE_STORE_MODE_FIELDS

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	warnFieldsModeOnKV("players.proto", desc, DBTypeRedis)
	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	out := buf.String()
	if out == "" {
		t.Fatal("expected warning on stderr")
	}
	if !bytes.Contains(buf.Bytes(), []byte("TABLE_STORE_MODE_FIELDS")) || !bytes.Contains(buf.Bytes(), []byte("redis")) {
		t.Fatalf("unexpected warning: %s", out)
	}
}

func TestWarnFieldsModeOnKV_SilentForMysql(t *testing.T) {
	desc := newMessageDescWithOrmDefaults()
	desc.Name = "Player"
	desc.OrmOptions.IsTable = true
	desc.OrmOptions.TableStoreMode = options.TableStoreMode_TABLE_STORE_MODE_FIELDS

	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	warnFieldsModeOnKV("players.proto", desc, DBTypeMySQL)
	_ = w.Close()
	os.Stderr = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if buf.Len() != 0 {
		t.Fatalf("mysql should not warn: %s", buf.String())
	}
}
