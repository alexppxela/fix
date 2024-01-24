module sylr.dev/fix

go 1.20

require (
	filippo.io/age v1.1.1
	github.com/dustin/go-humanize v1.0.1
	github.com/google/uuid v1.3.0
	github.com/iancoleman/strcase v0.2.0
	github.com/lib/pq v1.10.7
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/nats-io/nats-server/v2 v2.9.14
	github.com/nats-io/nats.go v1.23.0
	github.com/olekukonko/tablewriter v0.0.5
	github.com/prometheus/client_golang v1.14.0
	github.com/quickfixgo/enum v0.1.0
	github.com/quickfixgo/field v0.1.0
	github.com/quickfixgo/fixt11 v0.1.0
	github.com/quickfixgo/quickfix v0.7.0
	github.com/quickfixgo/tag v0.1.0
	github.com/rs/zerolog v1.29.0
	github.com/shopspring/decimal v1.3.1
	github.com/spf13/cobra v1.6.1
	github.com/spf13/pflag v1.0.5
	github.com/sylr/quickfixgo-fix50sp2/marketdataincrementalrefresh v0.0.0-20220401195242-281940b8a21e
	github.com/sylr/quickfixgo-fix50sp2/marketdatasnapshotfullrefresh v0.0.0-20220401195242-281940b8a21e
	golang.org/x/crypto v0.12.0
	golang.org/x/term v0.11.0
	sylr.dev/yaml/age/v3 v3.0.0-20221203153010-eb6b46db8d90
	sylr.dev/yaml/v3 v3.0.0-20220527135632-500fddf2b049
)

replace (
	github.com/quickfixgo/enum => github.com/artex-io/quickfixgo-enum v0.0.0-20230808145624-9890aaad90be
	github.com/quickfixgo/field => github.com/artex-io/quickfixgo-field v0.0.0-20230808145912-11a2ffdb0b8f
	github.com/quickfixgo/quickfix => github.com/artex-io/quickfix-go v0.6.1-0.20230808141925-433a4c32211f
	github.com/quickfixgo/tag => github.com/artex-io/quickfixgo-tag v0.0.0-20230808150418-f063b6a05d9a
	github.com/sylr/quickfixgo-fix50sp2/marketdataincrementalrefresh => github.com/artex-io/quickfixgo-fix50sp2/marketdataincrementalrefresh v0.0.0-20230808162530-8db962299863
	github.com/sylr/quickfixgo-fix50sp2/marketdatasnapshotfullrefresh => github.com/artex-io/quickfixgo-fix50sp2/marketdatasnapshotfullrefresh v0.0.0-20230808162530-8db962299863
)

require (
	filippo.io/edwards25519 v1.0.0 // indirect
	github.com/armon/go-proxyproto v0.0.0-20210323213023-7e956b284f0a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/gopherjs/gopherjs v1.17.2 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.16.7 // indirect
	github.com/kr/pretty v0.3.0 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/mattn/go-runewidth v0.0.14 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/minio/highwayhash v1.0.2 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/nats-io/jwt/v2 v2.3.0 // indirect
	github.com/nats-io/nkeys v0.3.0 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/client_model v0.3.0 // indirect
	github.com/prometheus/common v0.40.0 // indirect
	github.com/prometheus/procfs v0.9.0 // indirect
	github.com/rivo/uniseg v0.4.3 // indirect
	github.com/rogpeppe/go-internal v1.8.1 // indirect
	github.com/smartystreets/assertions v1.13.0 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20201027041543-1326539a0a0a // indirect
	go.mongodb.org/mongo-driver v1.12.1 // indirect
	golang.org/x/net v0.14.0 // indirect
	golang.org/x/sync v0.3.0 // indirect
	golang.org/x/sys v0.11.0 // indirect
	golang.org/x/text v0.12.0 // indirect
	golang.org/x/time v0.3.0 // indirect
	google.golang.org/protobuf v1.28.1 // indirect
)
