package migration

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"
	"time"

	"github.com/mitchellh/cli"
)

var ui cli.Ui

type Commands struct {
	Up     *UpCommand
	Down   *DownCommand
	Redo   *RedoCommand
	Status *StatusCommand
	New    *NewCommand
	Skip   *SkipCommand
}

type Migrate struct {
	CmdIndex   int
	Name       string
	EmbeddedFS embed.FS
	DB         *sql.DB
	IsEmbedded bool
	Dir        string `yaml:"directory"`
	TableName  string `yaml:"table"`
	Dialect    string `yaml:"dialect"`
	Cmd        *cli.CLI
	Commands   Commands
}

func New(cfg Config) *Migrate {

	if cfg.Name == "" {
		cfg.Name = "migrator"
	}

	if cfg.Dir == "" {
		cfg.Dir = defaultDir
	}
	os.MkdirAll(cfg.Dir, os.ModePerm)
	if cfg.Dialect == "" {
		cfg.Dialect = defaultDialect
	}
	if cfg.TableName == "" {
		cfg.TableName = defaultTableName
	}

	if cfg.TableName != "" {
		SetTable(cfg.TableName)
	} else {
		SetTable("")
	}
	i := &cli.BasicUi{Writer: os.Stdout}
	ui = &cli.ColoredUi{
		Ui:          i,
		OutputColor: cli.UiColorGreen,
		ErrorColor:  cli.UiColorRed,
		InfoColor:   cli.UiColorBlue,
		WarnColor:   cli.UiColorYellow,
	}

	m := &Migrate{
		CmdIndex:   cfg.CmdIndex,
		EmbeddedFS: cfg.EmbeddedFS,
		DB:         cfg.DB,
		Dir:        cfg.Dir,
		TableName:  cfg.TableName,
		Dialect:    cfg.Dialect,
	}
	m.Commands = Commands{
		Up:     &UpCommand{migrate: m},
		Down:   &DownCommand{migrate: m},
		Redo:   &RedoCommand{migrate: m},
		Status: &StatusCommand{migrate: m},
		New:    &NewCommand{migrate: m},
		Skip:   &SkipCommand{migrate: m},
	}
	m.Cmd = &cli.CLI{
		Commands: map[string]cli.CommandFactory{
			"up": func() (cli.Command, error) {
				return m.Commands.Up, nil
			},
			"down": func() (cli.Command, error) {
				return m.Commands.Down, nil
			},
			"redo": func() (cli.Command, error) {
				return m.Commands.Redo, nil
			},
			"status": func() (cli.Command, error) {
				return m.Commands.Status, nil
			},
			"new": func() (cli.Command, error) {
				return m.Commands.New, nil
			},
			"skip": func() (cli.Command, error) {
				return m.Commands.Skip, nil
			},
		},
		HelpFunc: cli.BasicHelpFunc(m.Name),
		Version:  "1.0.0",
	}
	return m
}

func (m *Migrate) Skip(limit int, dryRun bool) error {
	err := m.SkipMigration(m.Dialect, m.DB, Up, dryRun, limit)
	if err != nil {
		ui.Error(err.Error())
	}
	return err
}
func (m *Migrate) Status() error {
	return Status(m.Dir, m.Dialect, m.DB)
}
func (m *Migrate) New(name string) error {
	return m.Create(name)
}

func (m *Migrate) Up(limit int, dryRun bool) error {
	return m.Apply(Up, dryRun, limit)
}

func (m *Migrate) Down(limit int, dryRun bool) error {
	return m.Apply(Down, dryRun, limit)
}

func (m *Migrate) Redo(dryRun bool) error {
	return Redo(m.Dir, m.Dialect, m.DB, dryRun)
}

func (m *Migrate) Run() int {
	m.Cmd.Args = os.Args[m.CmdIndex:]
	exitCode, err := m.Cmd.Run()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}
	return exitCode
}

func (m *Migrate) Apply(dir MigrationDirection, dryrun bool, limit int) error {
	var source MigrationSource
	if m.IsEmbedded {
		source = EmbedFileSystemMigrationSource{
			FileSystem: m.EmbeddedFS,
			Root:       m.Dir,
		}
	} else {
		source = FileMigrationSource{
			Dir: m.Dir,
		}
	}
	if dryrun {
		migrations, _, err := PlanMigration(m.DB, m.Dialect, source, dir, limit)
		if err != nil {
			return fmt.Errorf("Cannot plan migration: %s", err)
		}

		for _, pm := range migrations {
			Print(pm, dir)
		}
	} else {
		n, err := ExecMax(m.DB, m.Dialect, source, dir, limit)
		if err != nil {
			return fmt.Errorf("Migration failed: %s", err)
		}

		if n == 1 {
			ui.Output("Applied 1 migration")
		} else {
			ui.Output(fmt.Sprintf("Applied %d migrations", n))
		}
	}

	return nil
}

func (m *Migrate) SkipMigration(dialect string, curBD *sql.DB, dir MigrationDirection, dryrun bool, limit int) error {
	source := FileMigrationSource{
		Dir: m.Dir,
	}

	n, err := SkipMax(curBD, dialect, source, dir, limit)
	if err != nil {
		return fmt.Errorf("Migration failed: %s", err)
	}

	switch n {
	case 0:
		ui.Output("All migrations have already been applied")
	case 1:
		ui.Output("Skipped 1 migration")
	default:
		ui.Output(fmt.Sprintf("Skipped %d migrations", n))
	}

	return nil
}

func (m *Migrate) Create(name string) error {
	name = strings.ToLower(name)
	if _, err := os.Stat(m.Dir); os.IsNotExist(err) {
		return err
	}
	query := m.GetQuery(name)
	if query != "" {
		tpl = template.Must(template.New("new_migration").Parse(query))
	}
	fileName := fmt.Sprintf("%s-%s.sql", time.Now().Format("20060102150405"), strings.TrimSpace(name))
	pathName := path.Join(m.Dir, fileName)
	f, err := os.Create(pathName)

	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	if err := tpl.Execute(f, nil); err != nil {
		return err
	}

	ui.Output(fmt.Sprintf("Created migration %s", pathName))
	return nil
}

func (m *Migrate) GetQuery(migrationName string) string {
	nameParts := strings.Split(migrationName, `_`)
	upQuery := ""
	downQuery := ""
	if nameParts[len(nameParts)-1] == "table" {
		switch nameParts[0] {
		case "create":
			if m.Dialect == "postgresql" {
				tableName := strings.Join(nameParts[1:(len(nameParts)-1)], `_`)
				createSequence := "CREATE SEQUENCE IF NOT EXISTS " + tableName + "_id_seq;\n"
				upQuery = createSequence + "CREATE TABLE IF NOT EXISTS " + tableName + `
(
	id int8 NOT NULL DEFAULT nextval('` + tableName + `_id_seq'::regclass) PRIMARY KEY, 
	is_active bool default false,
	created_at timestamptz,
	updated_at timestamptz,
	deleted_at timestamptz
)` + ";"
				dropSequenceQuery := "DROP SEQUENCE IF EXISTS " + tableName + "_seq;\n"
				downQuery = dropSequenceQuery + "DROP TABLE IF EXISTS " + tableName + ";"
			} else if m.Dialect == "mysql" {
				tableName := strings.Join(nameParts[1:(len(nameParts)-1)], `_`)
				upQuery = "CREATE TABLE IF NOT EXISTS " + tableName + `
(
	id BIGINT AUTO_INCREMENT PRIMARY KEY, 
	is_active bool default false,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME Null DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	deleted_at datetime Null
)` + ";"
				downQuery = "DROP TABLE IF EXISTS " + tableName + ";"
			}
			break
		case "drop":
			if m.Dialect == "postgresql" {
				tableName := strings.Join(nameParts[1:(len(nameParts)-1)], `_`)
				dropSequenceQuery := "DROP SEQUENCE IF EXISTS " + tableName + "_seq;\n"
				createSequence := "CREATE SEQUENCE IF NOT EXISTS " + tableName + "_id_seq;\n"
				upQuery = dropSequenceQuery + "DROP TABLE IF EXISTS " + tableName + ";"
				downQuery = createSequence + "CREATE TABLE IF NOT EXISTS " + tableName + `
(
	id int8 NOT NULL DEFAULT nextval('` + tableName + `_id_seq'::regclass) PRIMARY KEY, 
	is_active bool default false,
	created_at timestamptz,
	updated_at timestamptz,
	deleted_at timestamptz
)` + ";"
			} else if m.Dialect == "mysql" {
				tableName := strings.Join(nameParts[1:(len(nameParts)-1)], `_`)
				upQuery = "DROP TABLE IF EXISTS " + tableName + ";"
				downQuery = "CREATE TABLE IF NOT EXISTS " + tableName + `
(
	id BIGINT AUTO_INCREMENT PRIMARY KEY, 
	is_active bool default false,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME Null DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
	deleted_at datetime Null
)` + ";"
			}
			break
		case "add":
			for i, part := range nameParts {
				if part == "in" {
					field := strings.Join(nameParts[1:i], `_`)
					tableName := strings.Join(nameParts[(i+1):(len(nameParts)-1)], `_`)
					upQuery = "ALTER TABLE " + tableName + " ADD COLUMN " + field + " VARCHAR(200)" + ";"
					downQuery = "ALTER TABLE " + tableName + " DROP COLUMN " + field + ";"
					break
				}
			}
		case "remove":
			for i, part := range nameParts {
				if part == "from" {
					field := strings.Join(nameParts[1:i], `_`)
					tableName := strings.Join(nameParts[(i+1):(len(nameParts)-1)], `_`)
					upQuery = "ALTER TABLE " + tableName + " DROP COLUMN " + field + ";"
					downQuery = "ALTER TABLE " + tableName + " ADD COLUMN " + field + " VARCHAR(200)" + ";"
					break
				}
			}
		case "rename":
			for i, part := range nameParts {
				if part == "in" {
					oldTableName := strings.Join(nameParts[1:i], `_`)
					newTableName := strings.Join(nameParts[(i+1):(len(nameParts)-1)], `_`)
					upQuery = "ALTER TABLE " + oldTableName + " RENAME TO " + newTableName + ";"
					downQuery = "ALTER TABLE " + newTableName + " RENAME TO " + oldTableName + ";"
					break
				}
			}
		case "alter", "change":
			for i, part := range nameParts {
				if part == "in" {
					field := strings.Join(nameParts[1:i], `_`)
					tableName := strings.Join(nameParts[(i+1):(len(nameParts)-1)], `_`)
					upQuery = "ALTER TABLE " + tableName + " ALTER COLUMN " + field + " VARCHAR(200)" + ";"
					downQuery = "ALTER TABLE " + tableName + " ALTER COLUMN " + field + " VARCHAR(200)" + ";"
					break
				}
			}
		}
	}
	query := fmt.Sprintf(`
-- +migrate Up
%s

-- +migrate Down
%s
`, upQuery, downQuery)
	return query
}
