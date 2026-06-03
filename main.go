package main

import (
	"flag"
	"fmt"
	"os"

	"google.golang.org/protobuf/compiler/protogen"
)

var mod string
var pbGoDir string
var debug bool
var help bool

func main() {
	// 解析命令行参数
	parseFlags()

	if mod == "tag" {
		fmt.Println("mode:", mod, "pb-go-dir:", pbGoDir, "debug:", debug)
		// 为pbGoDir的.pb.go文件追加gorm tag
		if err := appendGormTags(pbGoDir); err != nil {
			panic(err)
		}
	} else {
		// 创建protogen插件选项
		opts := protogen.Options{}

		// 运行代码生成器
		opts.Run(func(gen *protogen.Plugin) error {
			// 获取protoc版本
			protocVersion = "(unknown)"
			if v := gen.Request.GetCompilerVersion(); v != nil {
				protocVersion = fmt.Sprintf("v%v.%v.%v", v.GetMajor(), v.GetMinor(), v.GetPatch())
				if s := v.GetSuffix(); s != "" {
					protocVersion += "-" + s
				}
			}
			// 遍历所有需要生成代码的文件
			allMsgs := make([]MessageDesc, 0)
			allEnums := make([]EnumDesc, 0)
			var file *protogen.File
			callback := func(messages []MessageDesc, enums []EnumDesc) {
				allMsgs = append(allMsgs, messages...)
				allEnums = append(allEnums, enums...)
			}
			for _, f := range gen.Files {
				if !f.Generate {
					continue
				}
				if file == nil {
					file = f
				}
				// 调用generateFile函数生成Proto文件
				if err := generate(gen, f, callback); err != nil {
					return fmt.Errorf("generate file %s failed: %w", f.Desc.Path(), err)
				}
			}
			err := generateMetadata(gen, file, allMsgs, allEnums)
			if err != nil {
				panic(err)
			}
			//err = generateTcaplusOptionProto(gen, file, DBTypeTcaplus)
			//if err != nil {
			//	panic(err)
			//}

			return nil
		})
	}
}

func parseFlags() {
	m := flag.String("mode", "", "插件模式: tag")
	dir := flag.String("pb-go-dir", ".", "输出目录路径")
	d := flag.Bool("debug", false, "启用调试模式")
	h := flag.Bool("help", false, "显示帮助信息")

	// 自定义用法信息
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `protoc-gen-go-orm.git - ProtoBuf ORM 代码生成插件

使用方法:
  protoc-gen-go-orm.git [选项]

选项:
`)
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
示例:
  protoc-gen-go-orm.git -mode=tag -pb-go-dir=./generated -debug
`)
	}

	// 解析参数
	flag.Parse()

	mod = *m
	pbGoDir = *dir
	debug = *d
	help = *h

	if help {
		flag.Usage()
		os.Exit(0)
	}

}
