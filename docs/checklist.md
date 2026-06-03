# ORM Generator Upgrade Checklist

## Documentation

- [x] Write requirements document.
- [x] Write development and testing notes.
- [ ] Keep this checklist updated while implementing.

## Tests

- [ ] Add failing tests for default and explicit node type parsing.
- [ ] Add failing tests for simple vs JSON field classification.
- [ ] Add failing tests for database type metadata helpers.
- [ ] Add generated template tests for Context hook method signatures.

## Options and Parsing

- [ ] Add `orm.node_type` message option.
- [ ] Add `TableStoreMode` with `TABLE_STORE_MODE_PAYLOAD` and `TABLE_STORE_MODE_FIELDS`.
- [ ] Do not add or generate `TABLE_STORE_MODE_HYBRID`.
- [ ] Regenerate `options/orm.option.pb.go`.
- [ ] Extend `MessageOrmOptions` with node type.
- [ ] Extend `MessageOrmOptions` with table store mode.
- [ ] Extend `FieldOrmOptions` with JSON field classification.

## Generation

- [ ] Add PostgreSQL database type.
- [ ] Update MySQL template to branch by table store mode.
- [ ] In payload mode, emit key/index/version fields plus `data`, `created_at`, and `updated_at`.
- [ ] In field mode, emit all declared fields.
- [ ] Do not emit automatic `data`, `created_at`, or `updated_at` fields in field mode.
- [ ] Add PostgreSQL template or reuse a database-aware table template.
- [ ] Update methods template with Context-aware encode/decode wrappers.
- [ ] Update metadata template with `NodeTables` and `GetNodeTables`.

## Examples

- [ ] Add node type ownership examples to proto files.
- [ ] Ensure complex fields demonstrate JSON mapping.

## Verification

- [ ] Run `go test ./...`.
- [ ] Build the plugin.
- [ ] Review changed code and generated API against requirements.
