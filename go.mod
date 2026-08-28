module httphq

// The version CI installs: setup-go reads this line and then sets
// GOTOOLCHAIN=local, which refuses the automatic download a `toolchain`
// directive would need, so the toolchain has to be named here.
//
// It is a floor, not a preference. Below it the standard library carries
// advisories this code reaches, and govulncheck in the pipeline fails. Above
// 1.26 the pinned golangci-lint refuses the module, because it will not read a
// language version newer than the one it was built with.
go 1.26.6

require (
	github.com/atrox/haikunatorgo/v2 v2.0.1
	github.com/glebarez/sqlite v1.11.0
	github.com/gofiber/contrib/v3/socketio v1.1.4
	github.com/gofiber/contrib/v3/websocket v1.1.4
	github.com/gofiber/fiber/v3 v3.2.0
	github.com/gofiber/template/html/v2 v2.1.3
	github.com/google/uuid v1.6.0
	github.com/robfig/cron/v3 v3.0.1
	github.com/stretchr/testify v1.11.1
	github.com/valyala/fasthttp v1.71.0
	gorm.io/datatypes v1.2.7
	gorm.io/gorm v1.31.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/andybalholm/brotli v1.2.1 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fasthttp/websocket v1.5.12 // indirect
	github.com/glebarez/go-sqlite v1.21.2 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/gofiber/schema v1.7.1 // indirect
	github.com/gofiber/template v1.8.3 // indirect
	github.com/gofiber/utils v1.2.0 // indirect
	github.com/gofiber/utils/v2 v2.0.4 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/klauspost/compress v1.18.6 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/remyoudompheng/bigfft v0.0.0-20230129092748-24d4a6f8daec // indirect
	github.com/savsgio/gotils v0.0.0-20250924091648-bce9a52d7761 // indirect
	github.com/tinylib/msgp v1.6.4 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	golang.org/x/crypto v0.51.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.39.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	gorm.io/driver/mysql v1.5.6 // indirect
	modernc.org/libc v1.22.5 // indirect
	modernc.org/mathutil v1.5.0 // indirect
	modernc.org/memory v1.5.0 // indirect
	modernc.org/sqlite v1.23.1 // indirect
)
