package application

import (
	"github.com/rs/zerolog"

	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/tag"

	"sylr.dev/fix/pkg/utils"
)

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

	bridge.router.AddRoute(quickfix.BeginStringFIX42, string(enum.MsgType_BUSINESS_MESSAGE_REJECT), bridge.onBusinessMessageReject)
	bridge.router.AddRoute(quickfix.BeginStringFIX44, string(enum.MsgType_BUSINESS_MESSAGE_REJECT), bridge.onBusinessMessageReject)
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

func (app *Bridge) onNewOrderSingleClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardClientMessageToExchange(msg, sessionID)
}

func (app *Bridge) onOrderCancelRequestClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardClientMessageToExchange(msg, sessionID)
}

func (app *Bridge) onOrderCancelReplaceRequestClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardClientMessageToExchange(msg, sessionID)
}

func (app *Bridge) onMassCancelRequestClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardClientMessageToExchange(msg, sessionID)
}

func (app *Bridge) onQuoteClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardClientMessageToExchange(msg, sessionID)
}

func (app *Bridge) onQuoteCancelClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardClientMessageToExchange(msg, sessionID)
}

func (app *Bridge) forwardClientMessageToExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if len(app.connectedExchanges) == 0 {
		return quickfix.NewMessageRejectError("No connected exchanges", int(tag.BusinessRejectReason), nil)
	}
	target := app.connectedExchanges[0]

	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	app.orderMapping[clOrdId] = sessionID

	if err := quickfix.SendToTarget(msg, target); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

/////////////// Exchange messages

func (app *Bridge) onExecutionReportExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardExchangeMessageToClient(msg, sessionID)
}

func (app *Bridge) onOrderCancelRejectExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardExchangeMessageToClient(msg, sessionID)
}

func (app *Bridge) onMassCancelReportExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.forwardExchangeMessageToClient(msg, sessionID)
}

func (app *Bridge) onQuoteStatusReportExchange(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	return app.unhandledMessage(msg, sessionID)
}

func (app *Bridge) forwardExchangeMessageToClient(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	clOrdId, err := msg.Body.GetString(tag.ClOrdID)
	if err != nil {
		return quickfix.NewMessageRejectError("Missing ClOrdID", int(tag.BusinessRejectReason), nil)
	}

	clientSessionID, found := app.orderMapping[clOrdId]
	if !found {
		app.Logger.Warn().Str("clOrdId", clOrdId).Str("session", sessionID.String()).Msg("No client session found for ClOrdID")
		return nil
	}

	if err := quickfix.SendToTarget(msg, clientSessionID); err != nil {
		return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
	}
	return nil
}

/////////////// BusinessMessageReject messages

func (app *Bridge) onBusinessMessageReject(msg *quickfix.Message, sessionID quickfix.SessionID) quickfix.MessageRejectError {
	if sessionID.IsFIXT() {
		if len(app.connectedExchanges) == 0 {
			return quickfix.NewMessageRejectError("No connected exchanges", int(tag.BusinessRejectReason), nil)
		}
		if err := quickfix.SendToTarget(msg, app.connectedExchanges[0]); err != nil {
			return quickfix.NewMessageRejectError(err.Error(), int(tag.BusinessRejectReason), nil)
		}
	}
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
