package main

import (
	_ "embed"
)

// mysql proto 模板
//
//go:embed templates/mysql.tmpl
var mysqlTemplate string

// tcaplus proto 模板
//
//go:embed templates/tcaplus.tmpl
var tcaplusTemplate string

// pgsql proto 模板
//
//go:embed templates/pgsql.tmpl
var pgsqlTemplate string

//go:embed templates/kv.tmpl
var kvTemplate string

// po 函数模板
//
//go:embed templates/methods.tmpl
var methodsTemplate string

// context hook 模板
//
//go:embed templates/context.tmpl
var contextTemplate string

//go:embed templates/json_codec.tmpl
var jsonCodecTemplate string


// 通用元数据模板
//
//go:embed templates/metadata.tmpl
var metadataTemplate string

// tcaplus proto option 模板
//
//go:embed templates/tcaplus_option.tmpl
var tcaplusOptionTemplate string

// vo 函数模板
//
//go:embed templates/values.tmpl
var valuesTemplate string
