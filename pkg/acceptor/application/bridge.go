package application

import (
	"github.com/rs/zerolog"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/field"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"

	"sylr.dev/fix/pkg/utils"
)

type FixTypes struct {
	fixString  quickfix.FIXString
	fixDecimal quickfix.FIXDecimal
	fixInt     quickfix.FIXInt
	fixTime    quickfix.FIXUTCTimestamp
}

func NewBridge() *Bridge {
	bridge := Bridge{
		connectedExchanges: []quickfix.SessionID{},
		orderMapping:       make(map[string]quickfix.SessionID),
		router:             quickfix.NewMessageRouter(),
	}

	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_ORDER_SINGLE), bridge.onNewOrderSingleClient)
	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_ORDER_CANCEL_REQUEST), bridge.onOrderCancelRequestClient)
	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_ORDER_CANCEL_REPLACE_REQUEST), bridge.onOrderCancelReplaceRequestClient)
	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_ORDER_MASS_CANCEL_REQUEST), bridge.onMassCancelRequestClient)
	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_QUOTE), bridge.onQuoteClient)
	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_QUOTE_CANCEL), bridge.onQuoteCancelClient)

	bridge.router.AddRoute(quickfix.BeginStringFIX42, string(enum.MsgType_EXECUTION_REPORT), bridge.onExecutionReportExchange)
	bridge.router.AddRoute(quickfix.BeginStringFIX42, string(enum.MsgType_ORDER_CANCEL_REJECT), bridge.onOrderCancelRejectExchange)
	bridge.router.AddRoute(quickfix.BeginStringFIX42, string(enum.MsgType_ORDER_MASS_CANCEL_REPORT), bridge.onMassCancelReportExchange)
	bridge.router.AddRoute(quickfix.BeginStringFIX42, string(enum.MsgType_QUOTE_STATUS_REPORT), bridge.onQuoteStatusReportExchange)

	bridge.router.AddRoute(quickfix.BeginStringFIX44, string(enum.MsgType_EXECUTION_REPORT), bridge.onExecutionReportExchange)
	bridge.router.AddRoute(quickfix.BeginStringFIX44, string(enum.MsgType_ORDER_CANCEL_REJECT), bridge.onOrderCancelRejectExchange)
	bridge.router.AddRoute(quickfix.BeginStringFIX44, string(enum.MsgType_ORDER_MASS_CANCEL_REPORT), bridge.onMassCancelReportExchange)
	bridge.router.AddRoute(quickfix.BeginStringFIX44, string(enum.MsgType_QUOTE_STATUS_REPORT), bridge.onQuoteStatusReportExchange)

	bridge.router.AddRoute(quickfix.ApplVerIDFIX50SP2, string(enum.MsgType_BUSINESS_MESSAGE_REJECT), bridge.onBusinessMessageReject)

	return &bridge
}

type Bridge struct {
	utils.QuickFixAppMessageLogger

	connectedExchanges []quickfix.SessionID
	orderMapping       map[string]quickfix.SessionID

	router   *quickfix.MessageRouter
	Settings *quickfix.Settings
}

func (app *Bridge) Close() {
}

// OnCreate notifies session creation.
func (app *Bridge) OnCreate(sessionID quickfix.SessionID) {
	app.Logger.Debug().Msgf("New session: %s", sessionID)
}

// OnLogon notifies session successfully logging on.
func (app *Bridge) OnLogon(sessionID quickfix.SessionID) {
	app.Logger.Debug().Msgf("Logon: %s", sessionID)
	if !sessionID.IsFIXT() {
		app.connectedExchanges = append(app.connectedExchanges, sessionID)
	}
}

// OnLogout notifies session logging off or disconnecting.
func (app *Bridge) OnLogout(sessionID quickfix.SessionID) {
	app.Logger.Debug().Msgf("Logout: %s", sessionID)
	if !sessionID.IsFIXT() {
		for i, s := range app.connectedExchanges {
			if s == sessionID {
				app.connectedExchanges = append(app.connectedExchanges[:i], app.connectedExchanges[i+1:]...)
				break
			}
		}
		app.connectedExchanges = append(app.connectedExchanges, sessionID)
	}
}

// ToAdmin notifies admin message being sent to target.
func (app *Bridge) ToAdmin(message *quickfix.Message, sessionID quickfix.SessionID) {
	app.LogMessage(zerolog.TraceLevel, message, sessionID, true)
}

// FromAdmin notifies admin message being received from target.
func (app *Bridge) FromAdmin(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	app.LogMessage(zerolog.TraceLevel, message, sessionID, false)
	return nil
}

// ToApp notifies app message being sent to target.
func (app *Bridge) ToApp(message *quickfix.Message, sessionID quickfix.SessionID) error {
	app.LogMessage(zerolog.TraceLevel, message, sessionID, true)
	return nil
}

// FromApp notifies app message being received from target.
func (app *Bridge) FromApp(message *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	app.LogMessage(zerolog.TraceLevel, message, sessionID, true)
	return app.router.Route(message, sessionID)
}

/////////////// Client messages

func (app *Bridge) onNewOrderSingleClient(order *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if len(app.connectedExchanges) == 0 {
		return quickfix.NewMessageRejectError("No connected exchanges", int(tag.BusinessRejectReason), nil)
	}
	target := app.connectedExchanges[0]

	clOrdId, err := order.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	app.orderMapping[clOrdId] = sessionID

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(target.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_ORDER_SINGLE))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, order.Body, []utils.FieldDescription{
		{&fixType.fixDecimal, tag.OrderQty, true},
		{&fixType.fixInt, tag.OrdType, true},
		{&fixType.fixDecimal, tag.Price, false},
		{&fixType.fixInt, tag.Side, true},
		{&fixType.fixString, tag.Symbol, true},
		{&fixType.fixInt, tag.TimeInForce, true},
		{&fixType.fixTime, tag.TransactTime, true},
		{&fixType.fixInt, tag.OrderOrigination, false},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, order.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, target); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onOrderCancelRequestClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if len(app.connectedExchanges) == 0 {
		return quickfix.NewMessageRejectError("No connected exchanges", int(tag.BusinessRejectReason), nil)
	}
	target := app.connectedExchanges[0]

	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	app.orderMapping[clOrdId] = sessionID

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(target.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_ORDER_CANCEL_REQUEST))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, msg.Body, []utils.FieldDescription{
		{&fixType.fixString, tag.OrigClOrdID, false},
		{&fixType.fixString, tag.OrderID, false},
		{&fixType.fixInt, tag.Side, true},
		{&fixType.fixString, tag.Symbol, false},
		{&fixType.fixTime, tag.TransactTime, true},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, msg.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, target); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onOrderCancelReplaceRequestClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if len(app.connectedExchanges) == 0 {
		return quickfix.NewMessageRejectError("No connected exchanges", int(tag.BusinessRejectReason), nil)
	}
	target := app.connectedExchanges[0]

	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	app.orderMapping[clOrdId] = sessionID

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(target.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_ORDER_CANCEL_REPLACE_REQUEST))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, msg.Body, []utils.FieldDescription{
		{&fixType.fixString, tag.OrigClOrdID, false},
		{&fixType.fixString, tag.OrderID, false},
		{&fixType.fixDecimal, tag.OrderQty, true},
		{&fixType.fixInt, tag.OrdType, true},
		{&fixType.fixDecimal, tag.Price, false},
		{&fixType.fixInt, tag.Side, true},
		{&fixType.fixString, tag.Symbol, false},
		{&fixType.fixInt, tag.TimeInForce, true},
		{&fixType.fixTime, tag.TransactTime, true},
		{&fixType.fixInt, tag.OrderOrigination, false},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, msg.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, target); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onMassCancelRequestClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if len(app.connectedExchanges) == 0 {
		return quickfix.NewMessageRejectError("No connected exchanges", int(tag.BusinessRejectReason), nil)
	}
	target := app.connectedExchanges[0]

	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	app.orderMapping[clOrdId] = sessionID

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(target.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_ORDER_MASS_CANCEL_REQUEST))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, msg.Body, []utils.FieldDescription{
		{&fixType.fixInt, tag.Side, true},
		{&fixType.fixString, tag.Symbol, true},
		{&fixType.fixTime, tag.TransactTime, true},
		{&fixType.fixInt, tag.MassCancelRequestType, true},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, msg.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, target); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onQuoteClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.unhandledMessage(msg, sessionID)
}

func (app *Bridge) onQuoteCancelClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.unhandledMessage(msg, sessionID)
}

/////////////// Exchange messages

func (app *Bridge) onExecutionReportExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	clientSessionID, found := app.orderMapping[clOrdId]
	if !found {
		app.Logger.Warn().Str("clOrdId", clOrdId).Str("session", sessionID.String()).Msg("No client session found for ClOrdID")
		return nil
	}

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(clientSessionID.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_EXECUTION_REPORT))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, msg.Body, []utils.FieldDescription{
		{&fixType.fixInt, tag.Account, false},
		{&fixType.fixString, tag.OrderID, true},
		{&fixType.fixString, tag.SecondaryOrderID, false},
		{&fixType.fixString, tag.OrigClOrdID, false},
		{&fixType.fixString, tag.ExecID, true},
		{&fixType.fixString, tag.TrdMatchID, false},
		{&fixType.fixInt, tag.ExecType, true},
		{&fixType.fixInt, tag.OrdStatus, true},
		{&fixType.fixString, tag.Symbol, true},
		{&fixType.fixInt, tag.OrdType, true},
		{&fixType.fixInt, tag.TimeInForce, true},
		{&fixType.fixInt, tag.Side, true},
		{&fixType.fixDecimal, tag.Price, false},
		{&fixType.fixDecimal, tag.LastPx, false},
		{&fixType.fixDecimal, tag.CumQty, true},
		{&fixType.fixDecimal, tag.OrderQty, false},
		{&fixType.fixDecimal, tag.LeavesQty, true},
		{&fixType.fixDecimal, tag.LastQty, false},
		{&fixType.fixString, tag.Text, false},
		{&fixType.fixTime, tag.TransactTime, true},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, msg.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, clientSessionID); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onOrderCancelRejectExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	clientSessionID, found := app.orderMapping[clOrdId]
	if !found {
		app.Logger.Warn().Str("clOrdId", clOrdId).Str("session", sessionID.String()).Msg("No client session found for ClOrdID")
		return nil
	}

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(clientSessionID.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_ORDER_CANCEL_REJECT))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, msg.Body, []utils.FieldDescription{
		{&fixType.fixInt, tag.Account, false},
		{&fixType.fixString, tag.OrderID, true},
		{&fixType.fixString, tag.SecondaryOrderID, false},
		{&fixType.fixInt, tag.OrdStatus, true},
		{&fixType.fixString, tag.OrigClOrdID, false},
		{&fixType.fixString, tag.Text, false},
		{&fixType.fixTime, tag.TransactTime, true},
		{&fixType.fixInt, tag.CxlRejReason, false},
		{&fixType.fixString, tag.CxlRejResponseTo, true},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, msg.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, clientSessionID); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onMassCancelReportExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	clientSessionID, found := app.orderMapping[clOrdId]
	if !found {
		app.Logger.Warn().Str("clOrdId", clOrdId).Str("session", sessionID.String()).Msg("No client session found for ClOrdID")
		return nil
	}

	message := quickfix.NewMessage()
	message.Header.Set(field.NewBeginString(clientSessionID.BeginString))

	message.Header.SetField(tag.MsgType, field.NewMsgType(enum.MsgType_ORDER_MASS_CANCEL_REPORT))
	message.Body.SetString(tag.ClOrdID, clOrdId)

	var fixType FixTypes
	if err := utils.QuickFixMessageCopyFields(&message.Body, msg.Body, []utils.FieldDescription{
		{&fixType.fixString, tag.MassActionReportID, false},
		{&fixType.fixInt, tag.MassCancelRequestType, true},
		{&fixType.fixInt, tag.MassCancelResponse, true},
		{&fixType.fixInt, tag.MassCancelRejectReason, true},
		{&fixType.fixString, tag.Text, false},
		{&fixType.fixTime, tag.TransactTime, false},
	}); err != nil {
		return err
	}
	if err := utils.QuickFixMessageCopyPartyIds(&message.Body, msg.Body); err != nil {
		return err
	}

	if err := quickfix.SendToTarget(message, clientSessionID); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

func (app *Bridge) onQuoteStatusReportExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.unhandledMessage(msg, sessionID)
}

/////////////// BusinessMessageReject messages

func (app *Bridge) onBusinessMessageReject(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.unhandledMessage(msg, sessionID)
}

func (app *Bridge) unhandledMessage(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	msgType, err := msg.Header.GetString(tag.MsgType)
	if err != nil {
		return err
	}
	app.Logger.Debug().Msgf("Unhandled message: %s for session [%s]", msgType, sessionID.String())
	return nil
}
