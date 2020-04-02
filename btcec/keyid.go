// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd Inc.
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec

import (
	"encoding/binary"
)

const KeyIDSize = 4

// KeyID is an identifying number of ASP keys in Prova. These are used to
// shorten the byte space requirements of scriptPubKeys and addresses.
type KeyID uint32

// ToAddressFormat encodes a key id in a format suitable for a Prova address.
func (id KeyID) ToAddressFormat(buf []byte) {
	binary.LittleEndian.PutUint32(buf, uint32(id))
}

// KeyIDFromAddressBuffer returns a key id for a address formatted bytes.
func KeyIDFromAddressBuffer(buf []byte) KeyID {
	id := binary.LittleEndian.Uint32(buf)
	return KeyID(id)
}
