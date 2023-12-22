package dict

import "github.com/quickfixgo/enum"

var Urgencies = map[string]enum.Urgency{
	"NORMAL":     enum.Urgency_NORMAL,
	"FLASH":      enum.Urgency_FLASH,
	"BACKGROUND": enum.Urgency_BACKGROUND,
}
