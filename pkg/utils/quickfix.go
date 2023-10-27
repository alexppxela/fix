package utils

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/go-set"
	"github.com/olekukonko/tablewriter"
	"github.com/quickfixgo/enum"
	"github.com/quickfixgo/tag"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"sylr.dev/fix/pkg/dict"

	"github.com/quickfixgo/quickfix"
	"github.com/quickfixgo/quickfix/datadictionary"
)

var (
	tagFilter = set.From[int]([]int{1724, 453, 447, 448, 452, 2376})
)

type QuickFixMessagePartSetter interface {
	Set(field quickfix.FieldWriter) *quickfix.FieldMap
}

func QuickFixMessagePartSetString[T quickfix.FieldWriter, T2 ~string](setter QuickFixMessagePartSetter, value T2, f func(T2) T) {
	if len(value) > 0 {
		setter.Set(f(value))
	}
}

func QuickFixMessagePartSetDecimal[T quickfix.FieldWriter, T2 ~string](setter QuickFixMessagePartSetter, value T2, f func(decimal.Decimal, int32) T, scale int32) {
	decimal.NewFromString(string(value))
	if len(value) > 0 {
		setter.Set(f(MustNot(decimal.NewFromString(string(value))), scale))
	}
}

type FieldDescription struct {
	Field    quickfix.FieldValue
	Tag      quickfix.Tag
	Required bool
}

func QuickFixMessageCopyFields(outBody *quickfix.Body, inBody quickfix.Body, fieldDescriptions []FieldDescription) quickfix.MessageRejectError {
	for _, f := range fieldDescriptions {
		if err := inBody.GetField(f.Tag, f.Field); err != nil {
			if f.Required {
				return err
			}
		}
		outBody.SetField(f.Tag, f.Field)
	}
	return nil
}

func QuickFixMessageCopyPartyIds(outBody *quickfix.Body, inBody quickfix.Body) quickfix.MessageRejectError {
	inParties := quickfix.NewRepeatingGroup(
		tag.NoPartyIDs,
		quickfix.GroupTemplate{
			quickfix.GroupElement(tag.PartyID),
			quickfix.GroupElement(tag.PartyIDSource),
			quickfix.GroupElement(tag.PartyRole),
			quickfix.GroupElement(tag.PartyRoleQualifier),
		},
	)
	if err := inBody.GetGroup(inParties); err != nil {
		return err
	}
	if inParties.Len() == 0 {
		return nil
	}

	outParties := quickfix.NewRepeatingGroup(
		tag.NoPartyIDs,
		quickfix.GroupTemplate{
			quickfix.GroupElement(tag.PartyID),
			quickfix.GroupElement(tag.PartyIDSource),
			quickfix.GroupElement(tag.PartyRole),
			quickfix.GroupElement(tag.PartyRoleQualifier),
		},
	)

	for i := 0; i < inParties.Len(); i++ {
		group := inParties.Get(i)
		party := outParties.Add()

		var fixString quickfix.FIXString
		if err := group.GetField(tag.PartyID, &fixString); err != nil {
			return err
		}
		party.SetField(tag.PartyID, fixString)
		if err := group.GetField(tag.PartyIDSource, &fixString); err == nil {
			party.SetField(tag.PartyIDSource, fixString)
		}
		var fixInt quickfix.FIXInt
		if err := group.GetField(tag.PartyRole, &fixInt); err != nil {
			return err
		}
		party.SetField(tag.PartyRole, fixInt)
		if err := group.GetField(tag.PartyRoleQualifier, &fixInt); err == nil {
			party.SetField(tag.PartyRoleQualifier, fixInt)
		}
	}
	outBody.SetGroup(outParties)
	return nil
}

type QuickFixAppMessageLogger struct {
	Logger                  *zerolog.Logger
	TransportDataDictionary *datadictionary.DataDictionary
	AppDataDictionary       *datadictionary.DataDictionary
}

func (app *QuickFixAppMessageLogger) LogMessageType(message *quickfix.Message, sessionID quickfix.SessionID, log string) {
	msgType, err := message.MsgType()
	if err != nil {
		app.Logger.Debug().CallerSkipFrame(1).Msgf("%s %s", sessionID.String(), log)
	} else {
		desc := MapSearch(dict.MessageTypes, enum.MsgType(msgType))
		if desc != nil {
			app.Logger.Debug().CallerSkipFrame(1).Msgf("%s %s %s(%s)", sessionID.String(), log, msgType, *desc)
		} else {
			app.Logger.Debug().CallerSkipFrame(1).Msgf("%s %s %s", sessionID.String(), log, msgType)
		}
	}
}

func (app *QuickFixAppMessageLogger) LogMessage(level zerolog.Level, message *quickfix.Message, sessionID quickfix.SessionID, sending bool) {
	if app.Logger.GetLevel() > level {
		return
	}

	formatStr := "<- %s %s"
	if sending {
		formatStr = "-> %s %s"
	}
	app.Logger.WithLevel(level).Msgf(formatStr, "Raw", strings.Replace(message.String(), "\001", "|", -1))
}

func (app *QuickFixAppMessageLogger) WriteMessageBodyAsTable(w io.Writer, message *quickfix.Message) {
	table := tablewriter.NewWriter(w)
	table.SetHeader([]string{"TAG", "DESCRIPTION", "VALUES"})
	table.SetBorders(tablewriter.Border{false, false, false, true})
	table.SetColumnSeparator(" ")
	table.SetCenterSeparator("-")
	table.SetColumnAlignment([]int{tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
	table.SetColWidth(42)

	var line []string

	fields := strings.Split(message.String(), "\001")

	for _, field := range fields[:len(fields)-2] {
		eqIdx := strings.Index(field, "=")
		if eqIdx == -1 {
			fmt.Fprintf(os.Stderr, "Misformed field: %s\n", field)
			continue
		}
		fieldTag := field[:eqIdx]
		tag, err := strconv.Atoi(fieldTag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Bad tag value in field: %s\n", field)
			continue
		}
		if tagFilter.Contains(tag) {
			continue
		}

		value := field[eqIdx+1:]

		if app.TransportDataDictionary != nil {
			if _, found := app.TransportDataDictionary.Header.Fields[tag]; found {
				continue
			}
		}

		var tagDescription = "<unknown>"
		if app.AppDataDictionary != nil {
			if _, found := app.AppDataDictionary.Trailer.Fields[tag]; found {
				continue
			}
			tagField, tok := app.AppDataDictionary.FieldTypeByTag[tag]
			if tok {
				tagDescription = tagField.Name()
			}
			if len(tagField.Enums) > 0 {
				if en, ok := tagField.Enums[value]; ok {
					value += fmt.Sprintf(" (%s)", en.Description)
				}
			}
		}

		line = []string{
			fieldTag,
			tagDescription,
			value,
		}

		table.Append(line)
	}

	table.Render()
}

func MapSearch[K comparable, V comparable](m map[K]V, search V) *K {
	for k, v := range m {
		if search == v {
			return &k
		}
	}

	return nil
}
