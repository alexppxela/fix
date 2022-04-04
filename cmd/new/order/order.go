package neworder

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/fixt11"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"

	"sylr.dev/fix/config"
	"sylr.dev/fix/pkg/cli/complete"
	"sylr.dev/fix/pkg/dict"
	"sylr.dev/fix/pkg/errors"
	"sylr.dev/fix/pkg/initiator"
	"sylr.dev/fix/pkg/initiator/application"
	"sylr.dev/fix/pkg/utils"
)

var (
	optionOrderSide, optionOrderType string
	optionOrderSymbol, optionOrderID string
	optionOrderExpiry                string
	optionOrderQuantity              int64
	optionOrderPrice                 float64
	optionOrderOrigination           string
	optionPartyIDs                   []string
	optionPartyIDSources             []string
	optionPartyRoles                 []string
	optionPartyRoleQualifiers        []int
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
	NewOrderCmd.Flags().StringVar(&optionOrderSide, "side", "", "Order side (buy, sell ... etc)")
	NewOrderCmd.Flags().StringVar(&optionOrderType, "type", "", "Order type (market, limit, stop ... etc)")
	NewOrderCmd.Flags().StringVar(&optionOrderSymbol, "symbol", "", "Order symbol")
	NewOrderCmd.Flags().Int64Var(&optionOrderQuantity, "quantity", 1, "Order quantity")
	NewOrderCmd.Flags().StringVar(&optionOrderID, "id", "", "Order id (uuid autogenerated if not given)")
	NewOrderCmd.Flags().StringVar(&optionOrderExpiry, "expiry", "day", "Order expiry (day, good_till_cancel ... etc)")
	NewOrderCmd.Flags().Float64Var(&optionOrderPrice, "price", 0.0, "Order price")
	NewOrderCmd.Flags().StringVar(&optionOrderOrigination, "origination", "", "Order origination")

	NewOrderCmd.Flags().StringSliceVar(&optionPartyIDs, "party-id", []string{}, "Order party ids")
	NewOrderCmd.Flags().StringSliceVar(&optionPartyIDSources, "party-id-source", []string{}, "Order party id sources")
	NewOrderCmd.Flags().StringSliceVar(&optionPartyRoles, "party-role", []string{}, "Order party roles")
	NewOrderCmd.Flags().IntSliceVar(&optionPartyRoleQualifiers, "party-role-qualifier", []int{}, "Order party role quelifiers")

	NewOrderCmd.MarkFlagRequired("side")
	NewOrderCmd.MarkFlagRequired("type")
	NewOrderCmd.MarkFlagRequired("symbol")
	NewOrderCmd.MarkFlagRequired("quantity")

	NewOrderCmd.RegisterFlagCompletionFunc("side", complete.OrderSide)
	NewOrderCmd.RegisterFlagCompletionFunc("type", complete.OrderType)
	NewOrderCmd.RegisterFlagCompletionFunc("expiry", complete.OrderTimeInForce)
	NewOrderCmd.RegisterFlagCompletionFunc("symbol", cobra.NoFileCompletions)
	NewOrderCmd.RegisterFlagCompletionFunc("origination", complete.OrderOriginationRole)
	NewOrderCmd.RegisterFlagCompletionFunc("party-id", cobra.NoFileCompletions)
	NewOrderCmd.RegisterFlagCompletionFunc("party-id-source", complete.OrderPartyIDSource)
	NewOrderCmd.RegisterFlagCompletionFunc("party-role", complete.OrderPartyIDRole)
	NewOrderCmd.RegisterFlagCompletionFunc("party-role-qualifier", cobra.NoFileCompletions)
}

func Validate(cmd *cobra.Command, args []string) error {
	sides := utils.PrettyOptionValues(dict.OrderSidesReversed)
	search := utils.Search(sides, strings.ToLower(optionOrderSide))
	if search < 0 {
		return errors.FixOrderSideUnknown
	}

	types := utils.PrettyOptionValues(dict.OrderTypesReversed)
	search = utils.Search(types, strings.ToLower(optionOrderType))
	if search < 0 {
		return errors.FixOrderTypeUnknown
	}

	if len(optionOrderOrigination) > 0 {
		originations := utils.PrettyOptionValues(dict.OrderOriginationsReversed)
		search = utils.Search(originations, strings.ToLower(optionOrderOrigination))
		if search < 0 {
			return errors.FixOrderOriginationUnknown
		}
	}

	if len(optionOrderID) == 0 {
		uid := uuid.New()
		optionOrderID = uid.String()
	}

	if strings.ToLower(optionOrderType) == "market" && optionOrderPrice > 0 {
		return errors.OptionsInvalidMarketPrice
	} else if strings.ToLower(optionOrderType) != "market" && optionOrderPrice == 0 {
		return errors.OptionsNoPriceGiven
	}

	if len(optionPartyIDs) != len(optionPartyIDSources) ||
		len(optionPartyIDs) != len(optionPartyRoles) ||
		len(optionPartyIDSources) != len(optionPartyRoles) {
		return fmt.Errorf("%v: you must provide the same number of --party-id, --party-id-source and --party-role", errors.OptionsInconsistentValues)
	}

	if len(optionPartyRoleQualifiers) > 0 &&
		len(optionPartyRoleQualifiers) != len(optionPartyIDs) {
		return fmt.Errorf("%v: you must provide the same number of --party-id, --party-id-source and --party-role", errors.OptionsInconsistentValues)
	}

	for _, t := range optionPartyIDSources {
		if _, ok := dict.PartyIDSourcesReversed[strings.ToUpper(t)]; !ok {
			return fmt.Errorf("%w: unkonwn party ID source `%s`", errors.Options, t)
		}
	}

	for _, t := range optionPartyRoles {
		if _, ok := dict.PartyRolesReversed[strings.ToUpper(t)]; !ok {
			return fmt.Errorf("%w: unkonwn party role `%s`", errors.Options, t)
		}
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
	order, err := buildMessage(*session)
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
	case responseMessage, ok = <-app.FromAdminChan:
		if !ok {
			return errors.FixLogout
		}
	case responseMessage, ok = <-app.FromAppChan:
		if !ok {
			return errors.FixLogout
		}
	}

	// Extract fields from the response
	msgType := field.MsgTypeField{}
	ordStatus := field.OrdStatusField{}
	text := field.TextField{}
	responseMessage.Header.GetField(tag.MsgType, &msgType)
	responseMessage.Body.GetField(tag.OrdStatus, &ordStatus)
	responseMessage.Body.GetField(tag.Text, &text)

	if msgType.Value() == enum.MsgType_REJECT {
		return fmt.Errorf("%w: %s", errors.Fix, text.String())
	}

	switch ordStatus.Value() {
	case enum.OrdStatus_NEW:
		fmt.Printf("Order accepted\n")
		app.WriteMessageBodyAsTable(os.Stdout, responseMessage)
	case enum.OrdStatus_REJECTED:
		if len(text.String()) > 0 {
			err = fmt.Errorf("%w: %s", errors.FixOrderRejected, text.String())
		}
	default:
		if len(text.String()) > 0 {
			err = fmt.Errorf("%w: %s", errors.FixOrderStatusUnknown, text.String())
		}
	}

	return err
}

func buildMessage(session config.Session) (quickfix.Messagable, error) {
	eside, err := dict.OrderSideStringToEnum(optionOrderSide)
	if err != nil {
		return nil, err
	}

	etype, err := dict.OrderTypeStringToEnum(optionOrderType)
	if err != nil {
		return nil, err
	}

	eExpiry, err := dict.OrderTimeInForceStringToEnum(optionOrderExpiry)
	if err != nil {
		return nil, err
	}

	// Prepare order
	clordid := field.NewClOrdID(optionOrderID)
	ordtype := field.NewOrdType(etype)
	transactime := field.NewTransactTime(time.Now())
	ordside := field.NewSide(eside)

	// Message
	message := quickfix.NewMessage()
	header := fixt11.NewHeader(&message.Header)

	switch session.BeginString {
	case quickfix.BeginStringFIXT11:
		switch session.DefaultApplVerID {
		case "FIX.5.0SP2":
			header.Set(field.NewMsgType(enum.MsgType_ORDER_SINGLE))
			message.Body.Set(clordid)
			message.Body.Set(ordside)
			message.Body.Set(transactime)
			message.Body.Set(ordtype)

			parties := quickfix.NewRepeatingGroup(
				tag.NoPartyIDs,
				quickfix.GroupTemplate{
					quickfix.GroupElement(tag.PartyID),
					quickfix.GroupElement(tag.PartyIDSource),
					quickfix.GroupElement(tag.PartyRole),
				},
			)

			for i := range optionPartyIDs {
				party := parties.Add()
				party.Set(field.NewPartyID(optionPartyIDs[i]))
				party.Set(field.NewPartyIDSource(enum.PartyIDSource(dict.PartyIDSourcesReversed[strings.ToUpper(optionPartyIDSources[i])])))
				party.Set(field.NewPartyRole(enum.PartyRole(dict.PartyRolesReversed[strings.ToUpper(optionPartyRoles[i])])))

				// PartyRoleQualifier
				if len(optionPartyRoleQualifiers) > 0 && optionPartyRoleQualifiers[i] != 0 {
					party.Set(field.NewPartyRoleQualifier(optionPartyRoleQualifiers[i]))
				}
			}
			message.Body.SetGroup(parties)

		default:
			return nil, errors.FixVersionNotImplemented
		}
	default:
		return nil, errors.FixVersionNotImplemented
	}

	utils.QuickFixMessagePartSet(&message.Header, session.TargetCompID, field.NewTargetCompID)
	utils.QuickFixMessagePartSet(&message.Header, session.TargetSubID, field.NewTargetSubID)
	utils.QuickFixMessagePartSet(&message.Header, session.SenderCompID, field.NewSenderCompID)
	utils.QuickFixMessagePartSet(&message.Header, session.SenderSubID, field.NewSenderSubID)

	message.Body.Set(field.NewHandlInst(enum.HandlInst_AUTOMATED_EXECUTION_ORDER_PRIVATE_NO_BROKER_INTERVENTION))
	message.Body.Set(field.NewSymbol(optionOrderSymbol))
	if etype != enum.OrdType_MARKET {
		message.Body.Set(field.NewPrice(decimal.NewFromFloat(optionOrderPrice), 2))
	}
	message.Body.Set(field.NewOrderQty(decimal.NewFromInt(optionOrderQuantity), 2))
	message.Body.Set(field.NewTimeInForce(eExpiry))

	if len(optionOrderOrigination) > 0 {
		message.Body.Set(field.NewOrderOrigination(enum.OrderOrigination(dict.OrderOriginationsReversed[strings.ToUpper(optionOrderOrigination)])))
	}

	return message, nil
}
