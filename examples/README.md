# protoc-gen-go-orm examples

| Proto | 说明 |
|-------|------|
| `players.proto` | FIELDS 模式、JSON/blob、`BaseModel` embedded |
| `list.proto` | 复合主键 + `composite_index` |
| `fields_player.proto` | FIELDS 列存储 + `-tags=db` 集成 |

```bash
bash build.sh
go test ./... -count=1
go test -tags=db ./src -run TestFieldsMode -count=1
go test -tags=db ./src -run TestLister_CompositePrimaryKeyDelete -count=1
```
