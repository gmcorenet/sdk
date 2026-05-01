module gmcore-console

go 1.19

require (
	gmcore-config v0.0.0
	gmcore-i18n v0.0.0
	gmcore-orm v0.0.0
	gmcore-seed v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/go-sql-driver/mysql v1.8.1 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgconn v1.14.3 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.3.3 // indirect
	github.com/jackc/pgservicefile v0.0.0-20221227161230-091c0ba34f0a // indirect
	github.com/jackc/pgtype v1.14.0 // indirect
	github.com/jackc/pgx/v4 v4.18.3 // indirect
	github.com/mattn/go-sqlite3 v1.14.22 // indirect
	gmcore-debugbar v0.0.0 // indirect
	gmcore-validation v0.0.0 // indirect
	golang.org/x/crypto v0.20.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace gmcore-i18n => ../gmcore-i18n

replace gmcore-config => ../gmcore-config

replace gmcore-orm => ../gmcore-orm

replace gmcore-seed => ../gmcore-seed

replace gmcore-debugbar => ../gmcore-debugbar

replace gmcore-validation => ../gmcore-validation
