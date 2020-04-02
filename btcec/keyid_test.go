// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec_test

import (
	"bytes"
	"testing"

	"github.com/pyx-partners/dmgd/btcec"
)

// TestKeyIdToAddressFormat tests converting key ids to address bytes.
func TestKeyIdToAddressFormat(t *testing.T) {
	keyId1 := btcec.KeyID(1)
	keyId1Bytes := []byte{0x01, 0x00, 0x00, 0x00}
	keyId1AddressFormatBytes := make([]byte, 4)
	keyId1.ToAddressFormat(keyId1AddressFormatBytes[:4])

	// Check that the key id is represented as the correct bytes.
	if !bytes.Equal(keyId1AddressFormatBytes, keyId1Bytes) {
		t.Fatalf("value of keyId1 is incorrect, %v is not "+
			"equal to %v", keyId1Bytes, keyId1AddressFormatBytes)
	}

	keyId2 := btcec.KeyID(65536)
	keyId2Bytes := []byte{0x00, 0x00, 0x01, 0x00}
	keyId2AddressFormatBytes := make([]byte, 4)
	keyId2.ToAddressFormat(keyId2AddressFormatBytes[:4])

	// Check that the key id is represented as the correct bytes.
	if !bytes.Equal(keyId2AddressFormatBytes, keyId2Bytes) {
		t.Fatalf("value of keyId2 is incorrect, %v is not "+
			"equal to %v", keyId2Bytes, keyId2AddressFormatBytes)
	}
}

// TestKeyIDFromAddressBuffer tests converting address bytes to key ids.
func TestKeyIDFromAddressBuffer(t *testing.T) {
	keyId1 := btcec.KeyID(1)
	keyId1FromBytes := btcec.KeyIDFromAddressBuffer([]byte{1, 0, 0, 0})

	// Check that the key id is represented as the correct bytes.
	if keyId1 != keyId1FromBytes {
		t.Fatalf("value of keyId1 is incorrect, %v is not "+
			"equal to %v", keyId1, keyId1FromBytes)
	}

	keyId2 := btcec.KeyID(65536)
	keyId2FromBytes := btcec.KeyIDFromAddressBuffer([]byte{0, 0, 1, 0})

	// Check that the key id is represented as the correct bytes.
	if keyId2 != keyId2FromBytes {
		t.Fatalf("value of keyId2 is incorrect, %v is not "+
			"equal to %v", keyId2, keyId2FromBytes)
	}
}
