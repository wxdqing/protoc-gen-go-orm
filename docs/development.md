# ORM Generator Development Notes

## Architecture

The generator reads proto descriptors, builds `MessageDesc` and `FieldDesc` values, then executes text templates for each database type.

The first implementation updates these areas:

- `options/orm.option.proto`: add `orm.node_type` ownership option and `TableStoreMode`.
- `generate_proto.go`: parse node type options and classify field complexity.
- `generate.go`: add PostgreSQL DB type and template helpers.
- `templates/mysql.tmpl`: branch by `TableStoreMode`; payload mode emits key/index/version plus `data`, `created_at`, and `updated_at`; field mode emits all table fields and JSON gorm tags for complex fields without synthetic fields.
- `templates/tcaplus.tmpl`: keep Tcaplus conservative and only emit key/index/version fields.
- `templates/methods.tmpl`: add Context and hook-aware encode/decode methods.
- `templates/metadata.tmpl`: generate node-aware table maps.
- `generate_test.go`: unit tests for field classification and metadata data preparation.
- `examples/proto/*.proto`: demonstrate node type ownership and complex fields.

## Testing Strategy

Use TDD for generator behavior.

Primary command:

```bash
go test ./...
```

Test coverage targets:

- default table node type is stable when no option is present.
- explicit node type is parsed from message options.
- simple fields are classified as normal DB fields.
- message, repeated, and map fields are classified as JSON fields.
- metadata supports all-table and node-filtered table loading.
- generated methods include backwards-compatible and context-aware encode/decode paths.

Manual generation check:

```bash
go build -o /tmp/protoc-gen-go-orm .
```

Then run the example script using the generated plugin path when needed.

## Compatibility

Existing generated method signatures remain valid. New Context methods are additive.

Generated table outputs are storage-mode driven. Payload mode keeps `data`, `created_at`, and `updated_at`. Field mode emits all declared table fields, using JSON tags for complex fields, and does not append synthetic columns.

`TableStoreMode` supports `PAYLOAD` and `FIELDS`. Do not add or generate `HYBRID`.

`GetAllTables(dbType string)` remains available. New code should prefer `GetNodeTables(dbType, node string)`.

## Review Checklist

Review generated code as part of implementation:

- no generated package has unused imports.
- MySQL field-mode output includes all declared fields with sensible gorm tags.
- Payload-mode output contains `data`, `created_at`, and `updated_at`.
- Field-mode output does not contain synthetic payload or timestamp fields.
- No generated code references `TABLE_STORE_MODE_HYBRID`.
- Tcaplus output keeps existing key/index constraints.
- metadata can answer by DB type and node.
- tests fail before implementation and pass after implementation.
