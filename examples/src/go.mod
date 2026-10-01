module github.com/wxdqing/protoc-gen-go-orm/examples/src

go 1.27.1

require (
	git.wxdqing.com/sprout/logger.git v0.2.0-alpha.2
	github.com/glebarez/sqlite v1.11.0
	github.com/wxdqing/go-orm v0.0.0-20260719022438-ef99df53014b
	github.com/wxdqing/protoc-gen-go-orm v0.0.0-20260713152656-fb815bf3a546
	google.golang.org/protobuf v1.36.12
	gorm.io/gorm v1.31.2
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.1.0 // indirect
	github.com/glebarez/go-sqlite v1.23.0 // indirect
	github.com/go-sql-driver/mysql v1.10.1 // indirect
	github.com/golang/protobuf v1.5.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/pgx/v5 v5.11.0 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.20.1 // indirect
	github.com/mattn/go-isatty v0.0.24 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/ncruces/go-strftime v1.1.0 // indirect
	github.com/redis/go-redis/v9 v9.22.0 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/samber/lo v1.53.0 // indirect
	github.com/samber/slog-common v0.22.0 // indirect
	github.com/sirupsen/logrus v1.10.2 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/tencentyun/tcaplusdb-go-sdk v0.2.3 // indirect
	github.com/tencentyun/tsf4g v0.0.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.2.0 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver/v2 v2.9.1 // indirect
	go.uber.org/atomic v1.12.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	go.uber.org/zap v1.28.0 // indirect
	golang.org/x/crypto v0.57.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/text v0.42.0 // indirect
	gorm.io/driver/mysql v1.6.0 // indirect
	gorm.io/driver/postgres v1.6.3 // indirect
	gorm.io/plugin/dbresolver v1.6.2 // indirect
	modernc.org/libc v1.77.1 // indirect
	modernc.org/mathutil v1.7.1 // indirect
	modernc.org/memory v1.12.1 // indirect
	modernc.org/sqlite v1.60.1 // indirect
)

replace (
	git.wxdqing.com/sprout/logger.git => ../../../../../../../rmini/sprout/logger
	github.com/wxdqing/go-orm => ../../../../../libs/go-orm
	github.com/wxdqing/protoc-gen-go-orm => ./../..
)
