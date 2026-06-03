#!/usr/bin/env bash
set -euo pipefail

PROTO_INPUT_DIR=./proto
OUT_BASE_DIR=./src
OUT_MYSQL_BASE_DIR=${OUT_BASE_DIR}/internal/mysql
OUT_TCAPLUS_BASE_DIR=${OUT_BASE_DIR}/internal/tcaplus
OUT_PGSQL_BASE_DIR=${OUT_BASE_DIR}/internal/pgsql
OUT_PROTO_DIR=proto

# 生成业务 proto、数据库 proto、metadata 和 methods
protoc \
  --go-orm.git_out=:${OUT_BASE_DIR} \
  --go_out=paths=source_relative:${OUT_BASE_DIR} \
  -I ../options -I ${PROTO_INPUT_DIR} \
  ${PROTO_INPUT_DIR}/*.proto

# 生成 mysql 数据库 .pb.go 文件
protoc \
  --go_out=paths=source_relative:${OUT_MYSQL_BASE_DIR} \
  -I ${OUT_MYSQL_BASE_DIR}/${OUT_PROTO_DIR} \
  ${OUT_MYSQL_BASE_DIR}/${OUT_PROTO_DIR}/*.proto

# mysql 追加 gorm tag
protoc-gen-go-orm.git -mode=tag -pb-go-dir=${OUT_MYSQL_BASE_DIR}

# 生成 pgsql 数据库 .pb.go 文件
protoc \
  --go_out=paths=source_relative:${OUT_PGSQL_BASE_DIR} \
  -I ${OUT_PGSQL_BASE_DIR}/${OUT_PROTO_DIR} \
  ${OUT_PGSQL_BASE_DIR}/${OUT_PROTO_DIR}/*.proto

# pgsql 追加 gorm tag
protoc-gen-go-orm.git -mode=tag -pb-go-dir=${OUT_PGSQL_BASE_DIR}

# 生成 tcaplus 数据库 .pb.go 文件
protoc \
  --go_out=paths=source_relative:${OUT_TCAPLUS_BASE_DIR} \
  -I ${OUT_TCAPLUS_BASE_DIR}/${OUT_PROTO_DIR} \
  ${OUT_TCAPLUS_BASE_DIR}/${OUT_PROTO_DIR}/*.proto
