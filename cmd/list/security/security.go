package listsecurity

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/spf13/cobra"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fixt11"
	"github.com/quickfixgo/quickfix"

	"sylr.dev/fix/config"
	"sylr.dev/fix/pkg/cli/complete"
	"sylr.dev/fix/pkg/dict"
	"sylr.dev/fix/pkg/errors"
	"sylr.dev/fix/pkg/initiator"
	"sylr.dev/fix/pkg/initiator/application"
	"sylr.dev/fix/pkg/utils"
)

var (
	optionType string
)

var ListSecurityCmd = &cobra.Command{
	Use:               "security",
	Aliases:           []string{"securities"},
	Short:             "List securities",
	Long:              "Send a securitylist FIX Message after initiating a session with a FIX acceptor.",
	Args:              cobra.ExactArgs(0),
	ValidArgsFunction: cobra.NoFileCompletions,
	PersistentPreRunE: utils.MakePersistentPreRunE(Validate),
	RunE:              Execute,
}

func init() {
	ListSecurityCmd.Flags().StringVar(&optionType, "type", "symbol", "Securities type (symbol, product ... etc)")

	ListSecurityCmd.RegisterFlagCompletionFunc("type", complete.SecurityListRequestType)
}

func Validate(cmd *cobra.Command, args []string) error {
	types := utils.PrettyOptionValues(dict.SecurityListRequestTypes)
	search := utils.Search(types, strings.ToLower(optionType))
	if search < 0 {
		return fmt.Errorf("unknown security type")
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

	acceptor, err := context.GetInitiator()
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

	app := application.NewSecurityList()
	app.Logger = logger
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
	if err = init.Start(); err != nil {
		return err
	}

	defer func() {
		app.Stop()
		init.Stop()
	}()

	// Choose right timeout cli option > config > default value (5s)
	var timeout time.Duration
	if options.Timeout != time.Duration(0) {
		timeout = options.Timeout
	} else if acceptor.SocketTimeout != time.Duration(0) {
		timeout = acceptor.SocketTimeout
	} else {
		timeout = 5 * time.Second
	}

	// Wait for session connection
	select {
	case <-time.After(timeout):
		return errors.ConnectionTimeout
	case _, ok := <-app.Connected:
		if !ok {
			return errors.FixLogout
		}
	}

	// Prepare securitylist
	securitylist, err := BuildMessage(*session)
	if err != nil {
		return err
	}

	// Send the order
	err = quickfix.Send(securitylist)
	if err != nil {
		return err
	}

	// Wait for the order response
	var responseMessage *quickfix.Message
	select {
	case <-time.After(timeout):
		return errors.ResponseTimeout
	case responseMessage = <-app.FromAppMessages:
	}

	app.WriteMessageBodyAsTable(os.Stdout, responseMessage)

	return nil
}

func BuildMessage(session config.Session) (quickfix.Messagable, error) {
	etype, err := dict.SecurityListRequestTypeStringToEnum(optionType)
	if err != nil {
		return nil, err
	}

	stype := field.NewSecurityListRequestType(etype)
	reqid := field.NewSecurityReqID(string(enum.SecurityRequestType_SYMBOL))

	// Message
	message := quickfix.NewMessage()
	header := fixt11.NewHeader(&message.Header)

	switch session.BeginString {
	case quickfix.BeginStringFIXT11:
		switch session.DefaultApplVerID {
		case "FIX.5.0SP2":
			header.Set(field.NewMsgType("x"))
			message.Body.Set(reqid)
			message.Body.Set(stype)
		default:
			return nil, errors.FixVersionNotImplemented
		}
	default:
		return nil, errors.FixVersionNotImplemented
	}

	utils.QuickFixMessagePartSetString(&message.Header, session.TargetCompID, field.NewTargetCompID)
	utils.QuickFixMessagePartSetString(&message.Header, session.TargetSubID, field.NewTargetSubID)
	utils.QuickFixMessagePartSetString(&message.Header, session.SenderCompID, field.NewSenderCompID)
	utils.QuickFixMessagePartSetString(&message.Header, session.SenderSubID, field.NewSenderSubID)

	return message, nil
}
