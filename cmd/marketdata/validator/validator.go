//go:build validator
// +build validator

package marketdatavalidator

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"sylr.dev/fix/config"
	"sylr.dev/fix/pkg/initiator"
	"sylr.dev/fix/pkg/initiator/application"
	"sylr.dev/fix/pkg/utils"
)

var (
	validatorOptions application.MarketDataValidatorOptions
)

var MarketDataValidatorCmd = &cobra.Command{
	Use:               "validator",
	Short:             "Validates market data retrieved",
	Long:              "Validates market data retrieved.",
	Args:              cobra.ExactArgs(0),
	ValidArgsFunction: cobra.NoFileCompletions,
	PersistentPreRunE: utils.MakePersistentPreRunE(Validate),
	RunE:              Execute,
}

func init() {
	MarketDataValidatorCmd.Flags().StringSliceVar(&validatorOptions.Symbols, "symbol", []string{}, "Symbol")
	MarketDataValidatorCmd.Flags().BoolVar(&validatorOptions.TradeHistory, "trade-history", false, "Subscribe to trade history")
	MarketDataValidatorCmd.Flags().BoolVar(&validatorOptions.ExitOnDisconnect, "exit-on-disconnect", false, "Subscribe to trade history")

	_ = MarketDataValidatorCmd.RegisterFlagCompletionFunc("symbol", cobra.NoFileCompletions)
}

func Validate(cmd *cobra.Command, args []string) error {
	err := utils.ReconcileBoolFlags(cmd.Flags())
	if err != nil {
		return err
	}

	return nil
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

	ctxInitiator, err := context.GetInitiator()
	if err != nil {
		return err
	}

	session := sessions[0]
	transportDict, appDict, err := session.GetFIXDictionaries()
	if err != nil {
		return err
	}

	settings, err := context.ToQuickFixInitiatorSettings()
	if err != nil {
		return err
	}

	app := application.NewMarketDataValidator(logger, validatorOptions, buildTimeoutDuration(ctxInitiator))
	app.Settings = settings
	app.TransportDataDictionary = transportDict
	app.AppDataDictionary = appDict

	var quickfixLogger *zerolog.Logger
	if options.QuickFixLogging {
		quickfixLogger = logger
	}

	init, err := initiator.Initiate(app, settings, quickfixLogger)
	if err != nil {
		return err
	}

	// Start session
	err = init.Start()
	if err != nil {
		return err
	}

	defer func() {
		app.Stop()
		init.Stop()
	}()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGINT, syscall.SIGTERM)

LOOP:
	for {
		select {
		case sig := <-interrupt:
			switch sig {
			case syscall.SIGINT, syscall.SIGTERM:
				logger.Debug().Msgf("Received signal: %v", sig)
				break LOOP
			default:
				logger.Info().Msgf("Received unhandled signal: %v", sig)
			}

		case msg, ok := <-app.AppInfoChan:
			if !ok {
				logger.Info().Msgf("Fix application not connected anymore")
				break LOOP
			} else {
				logger.Info().Str("msg", msg).Msgf("Marketdata validator event")
			}
		}
	}

	logger.Debug().Msgf("exiting")
	return nil
}

// Choose right timeout cli option > config > default value (5s)
func buildTimeoutDuration(ctxInitiator *config.Initiator) time.Duration {
	options := config.GetOptions()
	if options.Timeout != time.Duration(0) {
		return options.Timeout
	}
	if ctxInitiator.SocketTimeout != time.Duration(0) {
		return ctxInitiator.SocketTimeout
	}
	return 5 * time.Second
}
