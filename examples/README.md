# protoc-gen-go-orm examples

| Proto | 说明 |
|-------|------|
| `players.proto` | FIELDS 模式、JSON/blob、`BaseModel` embedded |
| `list.proto` | 复合主键 + `composite_index` |
| `fields_player.proto` | FIELDS 列存储 + `-tags=db` 集成 |
| `integration_game.proto` | 复合 PK + 联合索引 + embed/blob/json（`GameRole`） |
| `players.proto` / `list.proto` | 复杂 Player、复合主键 Lister |

```bash
bash build.sh
go test ./... -count=1
go test -tags=db ./src -run TestFieldsMode -count=1
go test -tags=db ./src -run TestLister_CompositePrimaryKeyDelete -count=1
go test -tags=db ./src -run TestIntegration_Complex -count=1 -v
```

MySQL / PostgreSQL 默认连接见 [go-orm 文档](https://github.com/wxdqing/go-orm)；可用 `ORM_TEST_MYSQL_*`、`ORM_TEST_PGSQL_*` 覆盖。

```bash
# 仓库根目录
make gen-examples test-integration-db
```
