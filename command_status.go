package migrations

import (
	"flag"
	"strings"
	"time"
)

type StatusCommand struct {
	migrate *Migrate
}

func (c *StatusCommand) Help() string {
	helpText := `
Usage: verify-rest status [options] ...

  Show migration status.

Options:

  -config=dbconfig.yml   Configuration file to use.
  -env="development"     Environment.

`
	return strings.TrimSpace(helpText)
}

func (c *StatusCommand) Synopsis() string {
	return "Show migration status"
}

func (c *StatusCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("status", flag.ContinueOnError)
	cmdFlags.Usage = func() { ui.Output(c.Help()) }

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}
	err := Status(c.migrate.Dir, c.migrate.Dialect, c.migrate.DB)
	if err != nil {
		return 1
	}
	return 0
}

type statusRow struct {
	Id        string
	Migrated  bool
	AppliedAt time.Time
}
