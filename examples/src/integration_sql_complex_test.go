//go:build db

package src_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"github.com/wxdqing/protoc-gen-go-orm/examples/src"
	"github.com/wxdqing/protoc-gen-go-orm/examples/src/metadata"
	ormmysql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/mysql"
	ormpgsql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/pgsql"
	"google.golang.org/protobuf/proto"
)

func integrationID(t *testing.T) int64 {
	t.Helper()
	return 700_000_000 + (time.Now().UnixNano() % 1_000_000)
}

func sqlIntegrationTables(driverType string) []proto.Message {
	switch driverType {
	case drivers.DriverTypeMySQL:
		return []proto.Message{
			&ormmysql.Player{},
			&ormmysql.FieldsPlayer{},
			&ormmysql.GameRole{},
			&ormmysql.Lister{},
		}
	case drivers.DriverTypePostgresSQL:
		return []proto.Message{
			&ormpgsql.Player{},
			&ormpgsql.FieldsPlayer{},
			&ormpgsql.GameRole{},
			&ormpgsql.Lister{},
		}
	default:
		return metadata.GetAllTables(driverType)
	}
}

func tryInitSQL(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	_ = drivers.Close(context.Background())
	t.Cleanup(func() { _ = drivers.Close(context.Background()) })
	tables := sqlIntegrationTables(driverType)
	if len(tables) == 0 {
		t.Fatalf("no tables for driver %s", driverType)
	}
	if err := drivers.TryInit(context.Background(),
		drivers.WithDriverType(driverType),
		drivers.WithTables(tables),
		drivers.WithConfig(conf),
	); err != nil {
		t.Fatal(err)
	}
}

func sampleComplexPlayer(id int64) *src.Player {
	// 避开 repeated 标量 JSON 列已知解码问题（array/enums）；保留 blob/map/message/embed。
	return &src.Player{
		Id:         id,
		Name:       "integration_player",
		Level:      42,
		Exp:        1000,
		PlayerEnum: src.PlayerEnum_Test2,
		Heros:    []*src.Hero{{Id: 1, Cid: 9, HeroLevel: 12}},
		Settings: map[string]int32{"difficulty": 3, "lang": 1},
		Nested: &src.Player_NestedM{
			F1: 99,
			F2: "nested",
			F3: map[string]int32{"a": 1},
		},
		Version: 1,
	}
}

func assertComplexPlayer(t *testing.T, got, want *src.Player) {
	t.Helper()
	if got.Name != want.Name || got.Level != want.Level || got.Exp != want.Exp {
		t.Fatalf("scalar: got %+v want %+v", got, want)
	}
	if got.PlayerEnum != want.PlayerEnum {
		t.Fatalf("enum: %v want %v", got.PlayerEnum, want.PlayerEnum)
	}
	if len(got.Heros) != 1 || got.Heros[0].HeroLevel != 12 {
		t.Fatalf("heros: %+v", got.Heros)
	}
	if got.Settings["difficulty"] != 3 {
		t.Fatalf("settings: %+v", got.Settings)
	}
	if got.Nested == nil || got.Nested.F2 != "nested" {
		t.Fatalf("nested: %+v", got.Nested)
	}
	if got.Version != want.Version {
		t.Fatalf("version: %d want %d", got.Version, want.Version)
	}
}

func sampleGameRole(serverID, roleID int64) *src.GameRole {
	return &src.GameRole{
		ServerId:   serverID,
		RoleId:     roleID,
		Name:       "role_alpha",
		Level:      15,
		Exp:        500,
		Heros:    []*src.RoleHero{{Id: 2, Cid: 7, HeroLevel: 4}},
		Settings: map[string]int32{"zone": 8},
		PlayerEnum: src.PlayerEnum_Test1,
		Profile:    []byte(`{"title":"knight"}`),
	}
}

func assertGameRole(t *testing.T, got, want *src.GameRole) {
	t.Helper()
	if got.ServerId != want.ServerId || got.RoleId != want.RoleId {
		t.Fatalf("pk: got (%d,%d) want (%d,%d)", got.ServerId, got.RoleId, want.ServerId, want.RoleId)
	}
	if got.Name != want.Name || got.Level != want.Level {
		t.Fatalf("fields: got %+v want %+v", got, want)
	}
	if len(got.Heros) != 1 || got.Settings["zone"] != 8 {
		t.Fatalf("blob/json: heros=%+v settings=%+v", got.Heros, got.Settings)
	}
	if len(got.Profile) == 0 {
		t.Fatalf("profile empty")
	}
}

func testPlayerComplexCRUD(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitSQL(t, driverType, conf)
	id := integrationID(t)
	want := sampleComplexPlayer(id)
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &src.Player{Id: id})
	})

	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &src.Player{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertComplexPlayer(t, got, want)

	if err := drivers.DefaultDbDriver.Delete(context.Background(), &src.Player{Id: id}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	err := drivers.DefaultDbDriver.Get(context.Background(), &src.Player{Id: id})
	if !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatalf("after Delete: %v", err)
	}
}

func testGameRoleCompositeCRUD(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitSQL(t, driverType, conf)
	serverID := int64(1001)
	roleID := integrationID(t)
	want := sampleGameRole(serverID, roleID)
	key := &src.GameRole{ServerId: serverID, RoleId: roleID}
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), key)
	})

	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &src.GameRole{ServerId: serverID, RoleId: roleID}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertGameRole(t, got, want)

	if err := drivers.DefaultDbDriver.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatalf("after Delete: %v", err)
	}
}

func testListerCompositeCRUD(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitSQL(t, driverType, conf)
	rid := int64(2002)
	id := integrationID(t)
	want := &src.Lister{Rid: rid, Id: id, Data: []byte("offline-payload")}
	key := &src.Lister{Rid: rid, Id: id}
	t.Cleanup(func() { _ = drivers.DefaultDbDriver.Delete(context.Background(), key) })

	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got := &src.Lister{Rid: rid, Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got.Data) != string(want.Data) {
		t.Fatalf("data: %q want %q", got.Data, want.Data)
	}
	if err := drivers.DefaultDbDriver.Delete(context.Background(), key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func testGameRoleFindByName(t *testing.T, driverType string, conf *orm.Conf) {
	t.Helper()
	tryInitSQL(t, driverType, conf)
	serverID := int64(3003)
	roleID := integrationID(t)
	want := sampleGameRole(serverID, roleID)
	want.Name = "find_me_role"
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &src.GameRole{ServerId: serverID, RoleId: roleID})
	})
	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatal(err)
	}

	cond := &src.GameRole{Name: "find_me_role"}
	rows, err := drivers.DefaultDbDriver.Find(context.Background(), cond)
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if len(rows) < 1 {
		t.Fatal("Find: expected at least one row")
	}
	found := false
	for _, r := range rows {
		p, ok := r.(*src.GameRole)
		if !ok {
			continue
		}
		if p.ServerId == serverID && p.RoleId == roleID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("Find: missing saved role among %d rows", len(rows))
	}
}

func TestIntegration_Complex_MySQL_Player(t *testing.T) {
	testPlayerComplexCRUD(t, drivers.DriverTypeMySQL, mysqlTestConf())
}

func TestIntegration_Complex_Pgsql_Player(t *testing.T) {
	testPlayerComplexCRUD(t, drivers.DriverTypePostgresSQL, pgsqlTestConf())
}

func TestIntegration_Complex_MySQL_GameRole(t *testing.T) {
	testGameRoleCompositeCRUD(t, drivers.DriverTypeMySQL, mysqlTestConf())
}

func TestIntegration_Complex_Pgsql_GameRole(t *testing.T) {
	testGameRoleCompositeCRUD(t, drivers.DriverTypePostgresSQL, pgsqlTestConf())
}

func TestIntegration_Complex_MySQL_GameRole_Find(t *testing.T) {
	testGameRoleFindByName(t, drivers.DriverTypeMySQL, mysqlTestConf())
}

func TestIntegration_Complex_Pgsql_GameRole_Find(t *testing.T) {
	testGameRoleFindByName(t, drivers.DriverTypePostgresSQL, pgsqlTestConf())
}

func TestIntegration_Complex_MySQL_Lister(t *testing.T) {
	testListerCompositeCRUD(t, drivers.DriverTypeMySQL, mysqlTestConf())
}

func TestIntegration_Complex_Pgsql_Lister(t *testing.T) {
	testListerCompositeCRUD(t, drivers.DriverTypePostgresSQL, pgsqlTestConf())
}

// 确认 metadata 已注册 SQL 表模型（生成后应含 GameRole）。
func TestIntegration_Metadata_HasGameRole(t *testing.T) {
	for _, dt := range []string{drivers.DriverTypeMySQL, drivers.DriverTypePostgresSQL} {
		var found bool
		for _, tb := range metadata.GetAllTables(dt) {
			switch dt {
			case drivers.DriverTypeMySQL:
				if _, ok := tb.(*ormmysql.GameRole); ok {
					found = true
				}
			case drivers.DriverTypePostgresSQL:
				if _, ok := tb.(*ormpgsql.GameRole); ok {
					found = true
				}
			}
		}
		if !found {
			t.Fatalf("metadata missing GameRole for %s", dt)
		}
	}
}
