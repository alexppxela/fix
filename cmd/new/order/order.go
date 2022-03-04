package neworder

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	nos50sp1 "github.com/quickfixgo/fix50sp1/newordersingle"
	nos50sp2 "github.com/quickfixgo/fix50sp2/newordersingle"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"sylr.dev/fix/config"
	"sylr.dev/fix/pkg/cli/complete"
	"sylr.dev/fix/pkg/dict"
	"sylr.dev/fix/pkg/errors"
	"sylr.dev/fix/pkg/initiator"
	"sylr.dev/fix/pkg/initiator/application"
	"sylr.dev/fix/pkg/utils"
)

var (
	optionSide, optionType string
	optionSymbol, optionID string
	optionExpiry           string
	optionQuantity         int64
	optionPrice            float64
)

var NewOrderCmd = &cobra.Command{
	Use:               "order",
	Short:             "New single order",
	Long:              "Send a new single order after initiating a session with a FIX acceptor.",
	Args:              cobra.ExactArgs(0),
	ValidArgsFunction: cobra.NoFileCompletions,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		err := Validate(cmd, args)
		if err != nil {
			return err
		}

		if cmd.HasParent() {
			parent := cmd.Parent()
			if parent.PersistentPreRunE != nil {
				return parent.PersistentPreRunE(cmd, args)
			}
		}
		return nil
	},
	RunE: Execute,
}

func init() {
	NewOrderCmd.Flags().StringVar(&optionSide, "side", "", "Order side (buy, sell ... etc)")
	NewOrderCmd.Flags().StringVar(&optionType, "type", "", "Order type (market, limit, stop ... etc)")
	NewOrderCmd.Flags().StringVar(&optionSymbol, "symbol", "", "Order symbol")
	NewOrderCmd.Flags().Int64Var(&optionQuantity, "quantity", 1, "Order quantity")
	NewOrderCmd.Flags().StringVar(&optionID, "id", "", "Order id (uuid autogenerated if not given)")
	NewOrderCmd.Flags().StringVar(&optionExpiry, "expiry", "day", "Order expiry (day, good_till_cancel ... etc)")
	NewOrderCmd.Flags().Float64Var(&optionPrice, "price", 0.0, "Order price")

	NewOrderCmd.MarkFlagRequired("side")
	NewOrderCmd.MarkFlagRequired("type")
	NewOrderCmd.MarkFlagRequired("symbol")
	NewOrderCmd.MarkFlagRequired("quantity")

	NewOrderCmd.RegisterFlagCompletionFunc("side", complete.OrderSide)
	NewOrderCmd.RegisterFlagCompletionFunc("type", complete.OrderType)
	NewOrderCmd.RegisterFlagCompletionFunc("expiry", complete.OrderTimeInForce)
	NewOrderCmd.RegisterFlagCompletionFunc("symbol", cobra.NoFileCompletions)
}

func Validate(cmd *cobra.Command, args []string) error {
	sides := utils.PrettyOptionValues(dict.OrderSidesReversed)
	search := utils.Search(sides, strings.ToLower(optionSide))
	if search < 0 {
		return errors.FixOrderSideUnknown
	}

	types := utils.PrettyOptionValues(dict.OrderTypesReversed)
	search = utils.Search(types, strings.ToLower(optionType))
	if search < 0 {
		return errors.FixOrderTypeUnknown
	}

	if len(optionID) == 0 {
		uid := uuid.New()
		optionID = uid.String()
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

	//
	session := sessions[0]
	initiatior, err := context.GetInitiator()
	if err != nil {
		return err
	}

	transportDict, appDict, err := session.GetFIXDictionaries()
	if err != nil {
		return err
	}

	settings, err := context.ToQuickFixInitiatorSettings()
	if err != nil {
		return err
	}

	app := application.NewNewOrder()
	app.Logger = logger
	app.Settings = settings
	app.TransportDataDictionary = transportDict
	app.AppDataDictionary = appDict

	init, err := initiator.Initiate(app, settings)
	if err != nil {
		return err
	}

	// Start session
	err = init.Start()
	if err != nil {
		return err
	}

	defer init.Stop()

	// Choose right timeout cli option > config > default value (5s)
	var timeout time.Duration
	if options.Timeout != time.Duration(0) {
		timeout = options.Timeout
	} else if initiatior.SocketTimeout != time.Duration(0) {
		timeout = initiatior.SocketTimeout
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

	// Prepare order
	order, err := new(*session)
	if err != nil {
		return err
	}

	// Send the order
	err = quickfix.Send(order)
	if err != nil {
		return err
	}

	// Wait for the order response
	var ok bool
	var responseMessage *quickfix.Message
	select {
	case <-time.After(timeout):
		return errors.ResponseTimeout
	case responseMessage, ok = <-app.FromAppChan:
		if !ok {
			return errors.FixLogout
		}
	}

	// Extract fields from the response
	ordStatus := field.OrdStatusField{}
	text := field.TextField{}
	responseMessage.Body.GetField(tag.OrdStatus, &ordStatus)
	responseMessage.Body.GetField(tag.Text, &text)

	switch ordStatus.Value() {
	case enum.OrdStatus_NEW:
		fmt.Printf("Order accepted\n")
		app.WriteMessageBodyAsTable(os.Stdout, responseMessage)
	case enum.OrdStatus_REJECTED:
		err = errors.FixOrderRejected
		if len(text.String()) > 0 {
			err = fmt.Errorf("%w: %s", err, text.String())
		}
	default:
		err = errors.FixOrderStatusUnknown
		if len(text.String()) > 0 {
			err = fmt.Errorf("%w: %s", err, text.String())
		}
		return err
	}

	return err
}

func new(session config.Session) (quickfix.Messagable, error) {
	var messagable quickfix.Messagable

	eside, err := dict.OrderSideStringToEnum(optionSide)
	if err != nil {
		return nil, err
	}

	etype, err := dict.OrderTypeStringToEnum(optionType)
	if err != nil {
		return nil, err
	}

	eExpiry, err := dict.OrderTimeInForceStringToEnum(optionExpiry)
	if err != nil {
		return nil, err
	}

	// Prepare order
	clordid := field.NewClOrdID(optionID)
	ordtype := field.NewOrdType(etype)
	transactime := field.NewTransactTime(time.Now())
	ordside := field.NewSide(eside)

	switch session.BeginString {
	case quickfix.BeginStringFIXT11:
		switch session.DefaultApplVerID {
		case "FIX.5.0SP1":
			messagable = nos50sp1.New(clordid, ordside, transactime, ordtype)
		case "FIX.5.0SP2":
			messagable = nos50sp2.New(clordid, ordside, transactime, ordtype)
		default:
			return nil, errors.FixVersionNotImplemented
		}
	default:
		return nil, errors.FixVersionNotImplemented
	}

	message := messagable.ToMessage()
	utils.QuickFixMessagePartSet(&message.Header, session.TargetCompID, field.NewTargetCompID)
	utils.QuickFixMessagePartSet(&message.Header, session.TargetSubID, field.NewTargetSubID)
	utils.QuickFixMessagePartSet(&message.Header, session.SenderCompID, field.NewSenderCompID)
	utils.QuickFixMessagePartSet(&message.Header, session.SenderSubID, field.NewSenderSubID)

	message.Body.Set(field.NewHandlInst(enum.HandlInst_AUTOMATED_EXECUTION_ORDER_PRIVATE_NO_BROKER_INTERVENTION))
	message.Body.Set(field.NewSymbol(optionSymbol))
	message.Body.Set(field.NewPrice(decimal.NewFromFloat(optionPrice), 2))
	message.Body.Set(field.NewOrderQty(decimal.NewFromInt(optionQuantity), 2))
	message.Body.Set(field.NewTimeInForce(eExpiry))

	return message, nil
}
