package migration

import (
	"flag"
	"fmt"
	"strings"
)

type UpCommand struct {
	migrate *Migrate
}

func (c *UpCommand) Help() string {
	helpText := `
Usage: %s up [options] ...

  Migrates the database to the most recent version available.

Options:

  -config=dbconfig.yml   Configuration file to use.
  -env="development"     Environment.
  -limit=0               Limit the number of migrations (0 = unlimited).
  -dryrun                Don't apply migrations, just print them.

`
	return strings.TrimSpace(fmt.Sprintf(helpText, Cmd))
}

func (c *UpCommand) Synopsis() string {
	return "Migrates the database to the most recent version available"
}

func (c *UpCommand) Run(args []string) int {
	var limit int
	var dryrun bool

	cmdFlags := flag.NewFlagSet("up", flag.ContinueOnError)
	cmdFlags.Usage = func() { ui.Output(c.Help()) }
	cmdFlags.IntVar(&limit, "limit", 0, "Max number of migrations to apply.")
	cmdFlags.BoolVar(&dryrun, "dryrun", false, "Don't apply migrations, just print them.")

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	err := c.migrate.Apply(Up, dryrun, limit)
	if err != nil {
		ui.Error(err.Error())
		return 1
	}

	return 0
}
