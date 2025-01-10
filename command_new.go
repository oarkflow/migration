package migration

import (
	"errors"
	"flag"
	"fmt"
	"strings"
	"text/template"
)

var templateContent = `
-- +migrate Up

-- +migrate Down
`
var tpl *template.Template

func init() {
	tpl = template.Must(template.New("new_migration").Parse(templateContent))
}

type NewCommand struct {
	migrate *Migrate
}

func (c *NewCommand) Help() string {
	helpText := `
Usage: %s new [options] name

  Create a new a database migration.

Options:

  -config=dbconfig.yml   Configuration file to use.
  -env="development"     Environment.
  name                   The name of the migration
`
	return strings.TrimSpace(fmt.Sprintf(helpText, Cmd))
}

func (c *NewCommand) Synopsis() string {
	return "Create a new migration"
}

func (c *NewCommand) Run(args []string) int {
	cmdFlags := flag.NewFlagSet("new", flag.ContinueOnError)
	cmdFlags.Usage = func() { ui.Output(c.Help()) }

	if len(args) < 1 {
		err := errors.New("A name for the migration is needed")
		ui.Error(err.Error())
		return 1
	}

	if err := cmdFlags.Parse(args); err != nil {
		return 1
	}

	if err := c.migrate.Create(cmdFlags.Arg(0)); err != nil {
		ui.Error(err.Error())
		return 1
	}
	return 0
}
