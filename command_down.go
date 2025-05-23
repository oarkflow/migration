package migration

import (
	"flag"
	"fmt"
	"strings"
)

type DownCommand struct {
	migrate *Migrate
}

func (c *DownCommand) Help() string {
	helpText := `
Usage: %s down [options] ...

  Undo a database migration.

Options:

  -config=dbconfig.yml   Configuration file to use.
  -env="development"     Environment.
  -limit=1               Limit the number of migrations (0 = unlimited).
  -dryrun                Don't apply migrations, just print them.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, Cmd))
}

func (c *DownCommand) Synopsis() string {
	return "Undo a database migration"
}

func (c *DownCommand) Run(args []string) int {
	var limit int
	var dryrun bool

	cmdFlags := flag.NewFlagSet("down", flag.ContinueOnError)
	cmdFlags.Usage = func() { ui.Output(c.Help()) }
	cmdFlags.IntVar(&limit, "limit", 1, "Max number of migrations to apply.")
	cmdFlags.BoolVar(&dryrun, "dryrun", false, "Don't apply migrations, just print them.")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	err := c.migrate.Apply(Down, dryrun, limit)
	if err != nil {
		ui.Error(err.Error())
		return 1
	}

	return 0
}
