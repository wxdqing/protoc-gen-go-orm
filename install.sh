#!/bin/bash

set -e

# 设置代理和构建参数
export GOPROXY=https://goproxy.cn,direct
export GOSUMDB=off

# 获取module名称作为二进制文件名基础
MODULE_NAME=$(go list -m)
echo "Module名称: $MODULE_NAME"

# 提取module名最后的部分并添加.git后缀
BINARY_NAME="${MODULE_NAME##*/}"

# 获取当前操作系统类型
OS="$(uname -s)"
echo "检测到操作系统: $OS"

# Windows系统下添加.exe后缀
if [ "$OS" = "Windows_NT" ] || [ "$OS" = "MINGW64_NT" ] || [ "$OS" = "CYGWIN_NT" ]; then
    BINARY_NAME="${BINARY_NAME}.exe"
fi

# 设置安装路径
INSTALL_PATH="$(go env GOPATH)/bin"
if [ ! -d "$INSTALL_PATH" ]; then
    mkdir -p "$INSTALL_PATH"
fi

echo "开始编译protoc-gen-go-orm..."
echo "安装路径: $INSTALL_PATH/$BINARY_NAME"

# 编译并安装
go build -o "$INSTALL_PATH/$BINARY_NAME" .

if [ $? -eq 0 ]; then
    echo "编译成功!"
    echo "protoc-gen-go-orm已安装到: $INSTALL_PATH/$BINARY_NAME"
    
    # 检查是否在PATH中
    if echo "$PATH" | grep -q "$INSTALL_PATH"; then
        echo "可以直接使用 protoc-gen-go-orm 命令"
    else
        echo "请将 $INSTALL_PATH 添加到PATH环境变量中"
    fi
else
    echo "编译失败!"
    exit 1
fi