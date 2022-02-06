package initiator

import (
	"fmt"

	"github.com/spf13/cobra"

	"sylr.dev/fix/config"
)

func ValidateOptions(cmd *cobra.Command, args []string) error {
	options := config.GetOptions()
	conf, err := config.ReadYAML(options.Config, options.Interactive)

	if err != nil {
		return fmt.Errorf("unable to read configuration: %w", err)
	}

	err = conf.Validate()
	if err != nil {
		return err
	}

	// Retrieve the global config pointer
	fixConfig := config.GetConfig()

	// Set the config retrieved in the config file into the global config pointer
	*fixConfig = *conf

	// Initialize the context name with the config current-context value
	contextName := fixConfig.CurrentContext

	if len(options.Context) > 0 {
		if len(options.Acceptor) > 0 || len(options.Session) > 0 {
			return fmt.Errorf("can't use --acceptor/--session with --context")
		}
		contextName = options.Context
	} else if len(contextName) == 0 {
		if len(options.Acceptor) == 0 || len(options.Session) == 0 {
			return fmt.Errorf("you need to specify either --context or --acceptor/--session")
		}
	}

	if len(contextName) == 0 {
		contextName = "no-context"
		fixConfig.Contexts = append(fixConfig.Contexts, &config.Context{
			Name:     contextName,
			Acceptor: options.Acceptor,
			Session:  options.Session,
		})
		(*options).Context = contextName
	}

	context, err := config.GetCurrentContext()
	if err != nil {
		return fmt.Errorf("%s: %w", config.ErrBadConfig, err)
	}

	_, err = context.GetAcceptor()
	if err != nil {
		return fmt.Errorf("%s: %w", config.ErrBadConfig, err)
	}

	_, err = context.GetSession()
	if err != nil {
		return fmt.Errorf("%s: %w", config.ErrBadConfig, err)
	}

	return nil
}

func AddPersistentFlags(cmd *cobra.Command) {
	options := config.GetOptions()

	cmd.PersistentFlags().StringVar(&options.Context, "context", "", "Context to use")
	cmd.PersistentFlags().StringVar(&options.Acceptor, "acceptor", "", "Acceptor to use (can't be used with --context)")
	cmd.PersistentFlags().StringVar(&options.Session, "session", "", "Session to use (can't be used with --context)")
	cmd.PersistentFlags().DurationVar(&options.Timeout, "timeout", 0, "Duration for timeouts")
}

func AddPersistentFlagCompletions(cmd *cobra.Command) {
	cmd.RegisterFlagCompletionFunc("context", completeContext)
	cmd.RegisterFlagCompletionFunc("acceptor", completeAcceptor)
	cmd.RegisterFlagCompletionFunc("session", completeSession)
}