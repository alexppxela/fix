//go:build acceptor
// +build acceptor

package cmd

import (
	"sylr.dev/fix/cmd/acceptor"
	"sylr.dev/fix/cmd/acceptor/bridge"
)

func init() {
	FixCmd.AddCommand(acceptor.AcceptorCmd)
	FixCmd.AddCommand(bridge.BridgeCmd)
}
