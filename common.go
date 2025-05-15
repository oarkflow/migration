package migration

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/olekukonko/tablewriter"
)

func Status(dir, dialect string, db *sql.DB) error {
	source := FileMigrationSource{
		Dir: dir,
	}
	migrations, err := source.FindMigrations()
	if err != nil {
		ui.Error(err.Error())
		return err
	}

	records, err := GetMigrationRecords(db, dialect)
	if err != nil {
		ui.Error(err.Error())
		return err
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{"Migration", "Applied"})

	rows := make(map[string]*statusRow)

	for _, m := range migrations {
		rows[m.Id] = &statusRow{
			Id:       m.Id,
			Migrated: false,
		}
	}

	for _, r := range records {
		if rows[r.Id] == nil {
			ui.Warn(fmt.Sprintf("Could not find migration file: %v", r.Id))
			continue
		}

		rows[r.Id].Migrated = true
		rows[r.Id].AppliedAt = r.AppliedAt
	}

	for _, m := range migrations {
		if rows[m.Id] != nil && rows[m.Id].Migrated {
			table.Append([]string{
				m.Id,
				rows[m.Id].AppliedAt.String(),
			})
		} else {
			table.Append([]string{
				m.Id,
				"no",
			})
		}
	}

	table.Render()

	return nil
}

func Redo(dir, dialect string, db *sql.DB, dryRun bool) error {
	source := FileMigrationSource{
		Dir: dir,
	}

	migrations, _, err := PlanMigration(db, dialect, source, Down, 1)
	if err != nil {
		ui.Error(fmt.Sprintf("Migration (redo) failed: %v", err))
		return err
	} else if len(migrations) == 0 {
		ui.Output("Nothing to do!")
		return nil
	}

	if dryRun {
		Print(migrations[0], Down)
		Print(migrations[0], Up)
	} else {
		_, err := ExecMax(db, dialect, source, Down, 1)
		if err != nil {
			ui.Error(fmt.Sprintf("Migration (down) failed: %s", err))
			return err
		}

		_, err = ExecMax(db, dialect, source, Up, 1)
		if err != nil {
			ui.Error(fmt.Sprintf("Migration (up) failed: %s", err))
			return err
		}

		ui.Output(fmt.Sprintf("Reapplied migration %s.", migrations[0].Id))
	}

	return nil
}

func Print(pm *PlannedMigration, dir MigrationDirection) {
	if dir == Up {
		ui.Output(fmt.Sprintf("==> Would apply migration %s (up)", pm.Id))
		for _, q := range pm.Up {
			ui.Output(q)
		}
	} else if dir == Down {
		ui.Output(fmt.Sprintf("==> Would apply migration %s (down)", pm.Id))
		for _, q := range pm.Down {
			ui.Output(q)
		}
	} else {
		panic("Not reached")
	}
}
