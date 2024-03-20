package cli

import "github.com/spf13/cobra"

func NewCmdRoot() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "shield",
		Short: "Root command",
	}

	cmd.AddCommand(NewCmdDB())
	cmd.AddCommand(NewCmdServer())

	return cmd
}
