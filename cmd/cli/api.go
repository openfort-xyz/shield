package cli

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/spf13/cobra"
	"go.openfort.xyz/shield/di"
)

func NewCmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Run the server",
		Long:    "Run the OpenFort Shield server",
		Example: "shield server",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			database, err := di.ProvideSQL()
			if err != nil {
				return err
			}

			err = database.Migrate()
			if err != nil {
				return err
			}

			server, err := di.ProvideRESTServer()
			if err != nil {
				return err
			}

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				<-sigCh
				_ = server.Stop(cmd.Context())
				wg.Done()
			}()

			if err = server.Start(cmd.Context()); err != nil {
				return err
			}

			wg.Wait()
			return nil
		},
	}
	return cmd
}
