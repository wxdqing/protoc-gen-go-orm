# ORM Generator Upgrade Checklist

## Documentation

- [x] Write requirements document.
- [x] Write development and testing notes.
- [x] Keep this checklist updated while implementing.

## Tests

- [x] Add failing tests for default and explicit node type parsing.
- [x] Add failing tests for simple vs JSON field classification.
- [x] Add failing tests for database type metadata helpers.
- [x] Add generated template tests for Context hook method signatures.

## Options and Parsing

- [x] Add `orm.node_type` message option.
- [x] Add `TableStoreMode` with `TABLE_STORE_MODE_PAYLOAD` and `TABLE_STORE_MODE_FIELDS`.
- [x] Do not add or generate `TABLE_STORE_MODE_HYBRID`.
- [x] Regenerate `options/orm.option.pb.go`.
- [x] Extend `MessageOrmOptions` with node type.
- [x] Extend `MessageOrmOptions` with table store mode.
- [x] Extend `FieldOrmOptions` with JSON field classification.

## Generation

- [x] Add PostgreSQL database type.
- [x] Update MySQL template to branch by table store mode.
- [x] In payload mode, emit key/index/version fields plus `data`, `created_at`, and `updated_at`.
- [x] In field mode, emit all declared fields.
- [x] Do not emit automatic `data`, `created_at`, or `updated_at` fields in field mode.
- [x] Add PostgreSQL template or reuse a database-aware table template.
- [x] Update methods template with Context-aware encode/decode wrappers.
- [x] Update metadata template with `NodeTables` and `GetNodeTables`.

## Examples

- [x] Add node type ownership examples to proto files.
- [x] Ensure complex fields demonstrate JSON mapping.

## KV & Custom Tags (2026-06-04)

- [x] Add `redis` / `mongo` to `supportedDBTypes()`.
- [x] Add `templates/kv.tmpl` (PAYLOAD record shape).
- [x] JSON field encode/decode (`json_codec.tmpl`).
- [x] Blob tag (`type:blob`) wire encode via `marshalProtoFieldToWire`.
- [x] Generate `ShardingKey()` in `values.tmpl`.
- [x] Remove `GormDeclared`; delete unused `*_legacy.tmpl`.
- [x] Cross-repo docs: `docs/protoc-gen-go-orm/checklist-tdd.md`, `tdd/*`.

## Verification

- [x] Run `go test ./...`.
- [x] Build the plugin.
- [x] Review changed code and generated API against requirements.
