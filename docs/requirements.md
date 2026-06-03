# ORM Generator Requirements

## Goal

Upgrade `protoc-gen-go-orm` into a field-aware ORM generator that can group tables by service node, map proto fields into database fields, and leave hooks for custom persistence behavior.

## Scope

This repository owns code generation. Runtime database drivers are out of scope for this phase, but generated code must expose stable interfaces that drivers can call later.

Tables must declare or inherit a clear storage mode. `TableStoreMode` is supported so generated code and runtime drivers know whether a table stores a marshaled payload or expanded fields.

Supported storage modes:

```proto
enum TableStoreMode {
  TABLE_STORE_MODE_PAYLOAD = 0; // pk/index/version + data payload
  TABLE_STORE_MODE_FIELDS = 1;  // field expansion: simple columns, complex JSON
}
```

`TABLE_STORE_MODE_HYBRID` is not supported. The generator must not create a mode that double-writes expanded fields and payload data.

## Requirements

### Table Ownership

Tables can declare node ownership with a message option:

```proto
option (orm.node_type) = "game";
```

`orm.node_type` is a single-value table classification, such as `game`, `login`, or `social`. Generated metadata must support loading tables by database type and node type so a service only initializes its own tables.

Compatibility rule: existing `GetAllTables(dbType string)` remains available and returns all tables for that database type.

### Field Mapping

For a table message using `TABLE_STORE_MODE_FIELDS`, generated database table proto should include all table fields.

Simple proto fields map to ordinary database fields:

- integer types
- unsigned integer types
- float and double
- bool
- string
- bytes
- enum

Complex proto fields map to JSON-like database fields:

- message fields
- repeated fields
- map fields

MySQL should emit `gorm:"type:json"` tags for complex fields. PostgreSQL support is represented by a `pgsql` database type and should emit JSONB-oriented table proto definitions.

For `TABLE_STORE_MODE_PAYLOAD`, generated database table proto should include key/index/version fields plus payload data and managed timestamps:

```proto
bytes data = 999;
int64 created_at = 1000;
int64 updated_at = 1001;
```

For `TABLE_STORE_MODE_FIELDS`, the generator must not automatically append synthetic payload or timestamp fields:

```proto
bytes data = 999;
int64 created_at = 1000;
int64 updated_at = 1001;
```

If a field-mode table needs `created_at`, `updated_at`, or any other audit field, the source proto must declare those fields explicitly. The generated table follows the proto definition instead of adding hidden columns.

### Context and Custom Hooks

Generated table methods must support context-aware conversion without breaking existing call sites.

Existing methods remain:

```go
EncodeFrom(value proto.Message) error
DecodeTo(value proto.Message) error
```

New methods are added:

```go
EncodeFromContext(ctx Context, value proto.Message) error
DecodeToContext(ctx Context, value proto.Message) error
```

The generated package defines a minimal `Context` interface compatible with `context.Context`:

```go
type Context interface {
    Value(key any) any
}
```

Custom table logic is injected by implementing optional hook interfaces on generated table structs:

```go
type BeforeEncodeHook interface {
    BeforeEncode(ctx Context, value proto.Message) error
}

type AfterEncodeHook interface {
    AfterEncode(ctx Context, value proto.Message) error
}

type BeforeDecodeHook interface {
    BeforeDecode(ctx Context, value proto.Message) error
}

type AfterDecodeHook interface {
    AfterDecode(ctx Context, value proto.Message) error
}
```

Drivers can later pass operation-specific context values through `Context` and table-specific code can customize conversion through these hooks.

## Non-Goals

- Implement runtime SQL drivers.
- Implement `TABLE_STORE_MODE_HYBRID`.
- Remove Tcaplus generation.
- Introduce a new external dependency.
