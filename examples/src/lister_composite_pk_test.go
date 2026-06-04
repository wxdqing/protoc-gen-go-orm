//go:build db

package src_test

import (
	"context"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/mysql"
	"gorm.io/gorm"
)

// BDD「复合主键删除」：Delete 使用 ToPrimaryKeyMap 含 rid+id（T-CR-001 / Lister）。
func TestLister_CompositePrimaryKeyDelete(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&mysql.Lister{}); err != nil {
		t.Fatal(err)
	}
	row := &mysql.Lister{Rid: 10, Id: 20, Data: []byte("x")}
	if err := db.Create(row).Error; err != nil {
		t.Fatal(err)
	}
	pk := row.ToPrimaryKeyMap()
	sql := db.ToSQL(func(tx *gorm.DB) *gorm.DB {
		return tx.Model(row).Where(pk).Delete(row)
	})
	if !strings.Contains(sql, "rid") || !strings.Contains(sql, "id") {
		t.Fatalf("DELETE should use composite pk, got: %s", sql)
	}
	if err := db.Model(row).Where(pk).Delete(row).Error; err != nil {
		t.Fatal(err)
	}
	var n int64
	if err := db.Model(&mysql.Lister{}).Where(pk).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("count after delete = %d", n)
	}
	_ = context.Background()
}
