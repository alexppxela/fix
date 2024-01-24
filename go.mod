module sylr.dev/fix

go 1.21

toolchain go1.21.3

require (
	filippo.io/age v1.1.1
	github.com/dustin/go-humanize v1.0.1
	github.com/google/uuid v1.5.0
	github.com/hashicorp/go-set v0.1.14
	github.com/iancoleman/strcase v0.3.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.18
	github.com/nats-io/nats-server/v2 v2.10.7
	github.com/nats-io/nats.go v1.31.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/prometheus/client_golang v1.17.0
	github.com/quickfixgo/enum v0.1.0
	github.com/quickfixgo/field v0.1.0
	github.com/quickfixgo/fix50sp2 v0.0.0-00010101000000-000000000000
	github.com/quickfixgo/fixt11 v0.1.0
	github.com/quickfixgo/quickfix v0.9.0
	github.com/quickfixgo/tag v0.1.0
	github.com/rs/zerolog v1.31.0
	github.com/shopspring/decimal v1.3.1
	github.com/spf13/cobra v1.8.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/crypto v0.18.0
	golang.org/x/term v0.16.0
	sylr.dev/yaml/age/v3 v3.0.0-20221203153010-eb6b46db8d90
	sylr.dev/yaml/v3 v3.0.0-20220527135632-500fddf2b049
)

replace (
	github.com/quickfixgo/enum => github.com/alexppxela/quickfixgo-enum v0.0.0-20240123152700-3da353f95834
	github.com/quickfixgo/field => github.com/alexppxela/quickfixgo-field v0.0.0-20240123152653-f0a896744ceb
	github.com/quickfixgo/fix50sp2 => github.com/alexppxela/quickfixgo-fix50sp2 v0.0.0-20240123165040-5354803752b9
	github.com/quickfixgo/quickfix => github.com/alexppxela/quickfixgo v0.0.0-20240123152459-2877207c30e2
	github.com/quickfixgo/tag => github.com/alexppxela/quickfixgo-tag v0.0.0-20240123152647-cf0a4d9525ed
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/armon/go-proxyproto v0.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.15 // indirect
	github.com/matttproud/golang_protobuf_extensions/v2 v2.0.0 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/nats-io/jwt/v2 v2.5.3 // indirect
	github.com/nats-io/nkeys v0.4.6 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.5.0 // indirect
	github.com/prometheus/common v0.45.0 // indirect
	github.com/prometheus/procfs v0.12.0 // indirect
	github.com/rivo/uniseg v0.4.4 // indirect
	github.com/smartystreets/assertions v1.13.0 // indirect
	golang.org/x/net v0.20.0 // indirect
	golang.org/x/sys v0.16.0 // indirect
	golang.org/x/time v0.5.0 // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)
