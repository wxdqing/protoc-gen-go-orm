# 项目根目录
ROOT_DIR := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

# Go相关设置
GO := go
GO_BUILD := $(GO) build
GO_TEST := $(GO) test
GO_INSTALL := $(GO) install

# 插件名称
PLUGIN_NAME := protoc-gen-go-orm

# 输出目录
OUTPUT_DIR := $(ROOT_DIR)/bin

# 测试数据目录
TEST_DATA_DIR := $(ROOT_DIR)/testdata

# 编译插件
.PHONY: build
build:
	@mkdir -p $(OUTPUT_DIR)
	$(GO_BUILD) -o $(OUTPUT_DIR)/$(PLUGIN_NAME) $(ROOT_DIR)

# 安装插件
.PHONY: install
install:
	$(GO_INSTALL) $(ROOT_DIR)

# 运行测试
.PHONY: test
test:
	@echo "Running tests..."
	@$(GO) run $(ROOT_DIR) --go_out=./testoutput --go-orm_out=./testoutput $(TEST_DATA_DIR)/test.proto

# 清理编译产物
.PHONY: clean
clean:
	@rm -rf $(OUTPUT_DIR)
	@rm -rf ./testoutput

# 帮助信息
.PHONY: help
help:
	@echo "Usage: make [TARGET]"
	@echo "Targets:"
	@echo "  build     编译插件"
	@echo "  install   安装插件到GOPATH/bin"
	@echo "  test      运行测试"
	@echo "  clean     清理编译产物"
	@echo "  help      显示帮助信息"

.DEFAULT_GOAL := build