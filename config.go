package migrations

import (
	"database/sql"
	"embed"

	"gopkg.in/gorp.v1"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

var dialects = map[string]gorp.Dialect{
	"sqlite3":    gorp.SqliteDialect{},
	"postgresql": gorp.PostgresDialect{},
	"mysql":      gorp.MySQLDialect{Engine: "InnoDB", Encoding: "UTF8"},
}

type Config struct {
	CmdIndex   int
	Name       string
	EmbeddedFS embed.FS
	DB         *sql.DB
	IsEmbedded bool
	Dir        string `yaml:"directory"`
	TableName  string `yaml:"table"`
	Dialect    string `yaml:"dialect"`
}

var (
	defaultDir       = "./database/migrations"
	defaultDialect   = "postgresql"
	defaultTableName = "migrations"
)
