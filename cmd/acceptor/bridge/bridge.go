package bridge

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
	"sylr.dev/fix/pkg/acceptor"

	"sylr.dev/fix/config"
	"sylr.dev/fix/pkg/acceptor/application"
	"sylr.dev/fix/pkg/utils"
)

var BridgeCmd = &cobra.Command{
	Use:               "bridge",
	Short:             "Launch a FIX bridge",
	Long:              "Launch a FIX bridge to route Artex FIX messages to NYFIX.",
	RunE:              Execute,
	PersistentPreRunE: utils.MakePersistentPreRunE(acceptor.ValidateOptions),
}

func init() {
	acceptor.AddPersistentFlags(BridgeCmd)
	acceptor.AddPersistentFlagCompletions(BridgeCmd)
}

func Execute(cmd *cobra.Command, args []string) error {
	options := config.GetOptions()
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

	app := application.NewBridge()

	app.TransportDataDictionary = transportDict
	app.AppDataDictionary = appDict
	app.Logger = logger

	var quickfixLogger *zerolog.Logger
	if options.QuickFixLogging {
		quickfixLogger = logger
	}

	bridge, err := acceptor.NewAcceptor(app, settings, quickfixLogger)
	if err != nil {
		return err
	}

	// Start session
	if err = bridge.Start(); err != nil {
		return err
	}

	defer func() {
		bridge.Stop()
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt
	bridge.Stop()
	os.Exit(0)

	return nil
}
