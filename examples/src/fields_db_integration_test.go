//go:build db

package src_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/wxdqing/go-orm/orm"
	"github.com/wxdqing/go-orm/orm/drivers"
	"github.com/wxdqing/protoc-gen-go-orm/examples/src"
	ormmysql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/mysql"
	ormpgsql "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/pgsql"
	ormredis "github.com/wxdqing/protoc-gen-go-orm/examples/src/internal/redis"
	logger "git.wxdqing.com/sprout/logger.git"
	"google.golang.org/protobuf/proto"
)

func init() {
	logger.Init()
}

func fieldsTestPlayerID() int64 {
	return 800_000_000 + (time.Now().UnixNano() % 1_000_000)
}

func mysqlTestConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypeMySQL,
		Mysql: orm.MysqlConf{
			Addr:     envOr("ORM_TEST_MYSQL_ADDR", "127.0.0.1:3306"),
			Name:     envOr("ORM_TEST_MYSQL_DB", "game"),
			User:     envOr("ORM_TEST_MYSQL_USER", "root"),
			Password: envOr("ORM_TEST_MYSQL_PASSWORD", "root123"),
			Startup:  orm.DefaultGormStartup("mysql"),
		},
	}
}

func pgsqlTestConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypePostgresSQL,
		Pgsql: orm.PgsqlConf{
			Host:     envOr("ORM_TEST_PGSQL_HOST", "127.0.0.1"),
			Port:     envOr("ORM_TEST_PGSQL_PORT", "5432"),
			Name:     envOr("ORM_TEST_PGSQL_DB", "game"),
			User:     envOr("ORM_TEST_PGSQL_USER", "postgres"),
			Password: envOr("ORM_TEST_PGSQL_PASSWORD", "postgres123"),
			Startup:  orm.DefaultGormStartup("pgsql"),
		},
	}
}

func redisTestConf() *orm.Conf {
	return &orm.Conf{
		Driver: drivers.DriverTypeRedis,
		Redis: orm.RedisConf{
			Host: envOr("ORM_TEST_REDIS_ADDR", "127.0.0.1:16379"),
		},
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func sampleFieldsPlayer(id int64) *src.FieldsPlayer {
	return &src.FieldsPlayer{
		Id:         id,
		Name:       "fields_db",
		Level:      11,
		Exp:        99,
		PlayerEnum: src.PlayerEnum_Test1,
		Settings:   map[string]int32{"k": 42},
		Heros:      []*src.Hero{{Id: 1, Cid: 3, HeroLevel: 5}},
	}
}

func assertFieldsPlayer(t *testing.T, got, want *src.FieldsPlayer, assertJSON bool) {
	t.Helper()
	if got.Name != want.Name || got.Level != want.Level || got.Exp != want.Exp {
		t.Fatalf("scalar: got %+v want %+v", got, want)
	}
	if got.PlayerEnum != want.PlayerEnum {
		t.Fatalf("enum: got %v want %v", got.PlayerEnum, want.PlayerEnum)
	}
	if assertJSON {
		if len(got.Settings) != 1 || got.Settings["k"] != 42 {
			t.Fatalf("settings: %+v", got.Settings)
		}
	}
	if len(got.Heros) != 1 || got.Heros[0].Cid != 3 || got.Heros[0].HeroLevel != 5 {
		t.Fatalf("heros: %+v", got.Heros)
	}
}

func testFieldsPlayerDBCRUD(t *testing.T, driverType string, table proto.Message, conf *orm.Conf, assertJSON bool) {
	t.Helper()
	_ = drivers.Close(context.Background())
	t.Cleanup(func() { _ = drivers.Close(context.Background()) })

	if err := drivers.TryInit(context.Background(),
		drivers.WithDriverType(driverType),
		drivers.WithTables([]proto.Message{table}),
		drivers.WithConfig(conf),
	); err != nil {
		t.Fatal(err)
	}

	id := fieldsTestPlayerID()
	want := sampleFieldsPlayer(id)
	t.Cleanup(func() {
		_ = drivers.DefaultDbDriver.Delete(context.Background(), &src.FieldsPlayer{Id: id})
	})

	if err := drivers.DefaultDbDriver.Save(context.Background(), want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got := &src.FieldsPlayer{Id: id}
	if err := drivers.DefaultDbDriver.Get(context.Background(), got); err != nil {
		t.Fatalf("Get: %v", err)
	}
	assertFieldsPlayer(t, got, want, assertJSON)

	if err := drivers.DefaultDbDriver.Delete(context.Background(), &src.FieldsPlayer{Id: id}); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	err := drivers.DefaultDbDriver.Get(context.Background(), &src.FieldsPlayer{Id: id})
	if !errors.Is(err, orm.ErrRecordNotFound) {
		t.Fatalf("after Delete: %v", err)
	}
}

// T-GEN-001 集成：FIELDS 表复杂列（settings + heros blob）在 SQL 上往返。
func TestFieldsMode_MySQL_FieldsPlayer_DB_CRUD(t *testing.T) {
	testFieldsPlayerDBCRUD(t, drivers.DriverTypeMySQL, &ormmysql.FieldsPlayer{}, mysqlTestConf(), true)
}

func TestFieldsMode_Pgsql_FieldsPlayer_DB_CRUD(t *testing.T) {
	testFieldsPlayerDBCRUD(t, drivers.DriverTypePostgresSQL, &ormpgsql.FieldsPlayer{}, pgsqlTestConf(), true)
}

// 业务 proto 为 FIELDS，redis 生成物仍为 PAYLOAD：整包存取应一致。
func TestFieldsMode_Redis_FieldsPlayer_DB_CRUD(t *testing.T) {
	testFieldsPlayerDBCRUD(t, drivers.DriverTypeRedis, &ormredis.FieldsPlayer{}, redisTestConf(), true)
}
