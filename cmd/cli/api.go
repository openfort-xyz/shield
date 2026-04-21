package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/openfort-xyz/shield/di"
	"github.com/openfort-xyz/shield/internal/tracing"
	"github.com/spf13/cobra"
)

func NewCmdServer() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "server",
		Short:   "Run the server",
		Long:    "Run the OpenFort Shield server",
		Example: "shield server",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			shutdownTracing, err := tracing.Init(cmd.Context())
			if err != nil {
				return fmt.Errorf("init tracing: %w", err)
			}
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = shutdownTracing(ctx)
			}()

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

			if err = server.Start(cmd.Context()); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}

			wg.Wait()
			return nil
		},
	}
	return cmd
}
