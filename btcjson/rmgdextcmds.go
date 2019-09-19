// Copyright (c) 2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// NOTE: This file is intended to house the RPC commands that are supported by
// a prova chain server.

package btcjson

// SetValidateKeysCmd defines the setvalidatekeys JSON-RPC command.
// This command is not a standard command, it is an extension for operating
// prova.
type SetValidateKeysCmd struct {
	PrivKeys []string
}

// NewSetValidateKeysCmd returns a new SetValidateKeysCmd which can
// be used to issue a setvalidatekeys JSON-RPC command.  This command is
// not a standard command. It is an extension for prova.
func NewSetValidateKeysCmd(privKeys []string) *SetValidateKeysCmd {
	return &SetValidateKeysCmd{
		PrivKeys: privKeys,
	}
}

func init() {
	// No special flags for commands in this file.
	flags := UsageFlag(0)

	MustRegisterCmd("setvalidatekeys", (*SetValidateKeysCmd)(nil), flags)
}
