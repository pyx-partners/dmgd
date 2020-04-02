// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec_test

import (
	"testing"

	"github.com/pyx-partners/dmgd/btcec"
)

func TestPublicKeySetIsEqual(t *testing.T) {
	pubKey1, err := btcec.ParsePubKey(
		[]byte{0x03, 0x26, 0x89, 0xc7, 0xc2, 0xda, 0xb1, 0x33,
			0x09, 0xfb, 0x14, 0x3e, 0x0e, 0x8f, 0xe3, 0x96, 0x34,
			0x25, 0x21, 0x88, 0x7e, 0x97, 0x66, 0x90, 0xb6, 0xb4,
			0x7f, 0x5b, 0x2a, 0x4b, 0x7d, 0x44, 0x8e,
		},
		btcec.S256(),
	)
	if err != nil {
		t.Fatalf("failed to parse raw bytes for pubKey1: %v", err)
	}

	pubKey2, err := btcec.ParsePubKey(
		[]byte{0x02, 0xce, 0x0b, 0x14, 0xfb, 0x84, 0x2b, 0x1b,
			0xa5, 0x49, 0xfd, 0xd6, 0x75, 0xc9, 0x80, 0x75, 0xf1,
			0x2e, 0x9c, 0x51, 0x0f, 0x8e, 0xf5, 0x2b, 0xd0, 0x21,
			0xa9, 0xa1, 0xf4, 0x80, 0x9d, 0x3b, 0x4d,
		},
		btcec.S256(),
	)
	if err != nil {
		t.Fatalf("failed to parse raw bytes for pubKey2: %v", err)
	}

	keySet1 := btcec.PublicKeySet{*pubKey1, *pubKey2}
	keySet2 := btcec.PublicKeySet{*pubKey1}
	keySet3 := btcec.PublicKeySet{*pubKey1, *pubKey2}
	keySet4 := btcec.PublicKeySet{}

	// Check that equivalent keysets are equivalent.
	if !keySet1.Equal(keySet3) {
		t.Fatalf("value of IsEqual is incorrect, %v is "+
			"equal to %v", pubKey1, pubKey1)
	}

	// Check that substractions from a keyset stop equivalence.
	if keySet1.Equal(keySet2) {
		t.Fatalf("value of IsEqual is incorrect, %v is not "+
			"equal to %v", pubKey1, pubKey2)
	}

	// Check that populated and empty keys are not equivalent.
	if keySet1.Equal(keySet4) {
		t.Fatalf("value of IsEqual is incorrect, %v is not "+
			"equal to %v", pubKey1, pubKey2)
	}
}
