package migrations

import (
	"flag"
	"strings"
)

type RedoCommand struct {
	migrate *Migrate
}

func (c *RedoCommand) Help() string {
	helpText := `
Usage: verify-rest redo [options] ...

  Reapply the last migration.

Options:

  -config=dbconfig.yml   Configuration file to use.
  -env="development"     Environment.
  -dryrun                Don't apply migrations, just print them.

`
	return strings.TrimSpace(helpText)
}

func (c *RedoCommand) Synopsis() string {
	return "Reapply the last migration"
}

func (c *RedoCommand) Run(args []string) int {
	var dryrun bool

	cmdFlags := flag.NewFlagSet("redo", flag.ContinueOnError)
	cmdFlags.Usage = func() { ui.Output(c.Help()) }
	cmdFlags.BoolVar(&dryrun, "dryrun", false, "Don't apply migrations, just print them.")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	err := Redo(c.migrate.Dir, c.migrate.Dialect, c.migrate.DB, dryrun)
	if err != nil {
		return 1
	}
	return 0
}
