package acceptor

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"sylr.dev/fix/config"
	"sylr.dev/fix/pkg/acceptor"
	"sylr.dev/fix/pkg/acceptor/application"
	"sylr.dev/fix/pkg/initiator"
)

var InitiatorCmd = &cobra.Command{
	Use:   "acceptor",
	Short: "Launch a FIX acceptor",
	Long:  "Launch a FIX acceptor.",
	RunE:  Execute,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := initiator.ValidateOptions(cmd, args)
		if err != nil {
			return err
		}

		if cmd.HasParent() {
			parent := cmd.Parent()
			if parent.PersistentPreRunE != nil {
				err = parent.PersistentPreRunE(parent, args)
				if err != nil {
					return err
				}
			}
		}

		return nil
	},
}

func init() {
	initiator.AddPersistentFlags(InitiatorCmd)
	initiator.AddPersistentFlagCompletions(InitiatorCmd)
}

func Execute(cmd *cobra.Command, args []string) error {
	logger := config.GetLogger()

	context, err := config.GetCurrentContext()
	if err != nil {
		return err
	}

	sessions, err := context.GetSessions()
	if err != nil {
		return err
	}

	settings, err := context.ToQuickFixAcceptorSettings()
	if err != nil {
		return err
	}

	transportDict, appDict, err := sessions[0].GetFIXDictionaries()
	if err != nil {
		return err
	}

	app, err := application.NewServer()
	if err != nil {
		return err
	}

	app.TransportDataDictionary = transportDict
	app.AppDataDictionary = appDict
	app.Logger = logger
	app.NatsConnect("nats://a:a@127.0.0.1:4222,nats://a:a@127.0.0.1:4223")
	if err != nil {
		return err
	}

	acceptor, err := acceptor.NewAcceptor(app, settings)
	if err != nil {
		return err
	}

	err = acceptor.Start()
	if err != nil {
		return err
	}

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt
	acceptor.Stop()
	os.Exit(0)

	return nil
}