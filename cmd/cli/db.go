package cli

import (
	"github.com/spf13/cobra"
	"go.openfort.xyz/shield/di"
	"go.openfort.xyz/shield/internal/infrastructure/repositories/sql"
)

func NewCmdDB() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db",
		Short: "Database operations",
	}
	cmd.AddCommand(NewCmdMigrate())
	cmd.AddCommand(NewCmdCreateMigration())
	return cmd
}

func NewCmdMigrate() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "migrate",
		Short:   "Migrate database",
		Long:    "Migrate the database to the latest version",
		Example: "shield db migrate",
		Args:    cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			repo, err := di.ProvideSQL()
			if err != nil {
				return err
			}

			return repo.Migrate()
		},
	}
	return cmd
}

func NewCmdCreateMigration() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create-migration [migration_name]",
		Short:   "Migrate database",
		Long:    "Create a new migration file with the given name. The migration file will be created in the migrations directory.",
		Example: "shield db create-migration [migration_name]",
		Args:    cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return sql.CreateMigration(args[0])
		},
	}
	return cmd
}
