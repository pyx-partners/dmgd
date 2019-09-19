// Copyright (c) 2013-2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package btcec_test

import (
	"testing"

	"github.com/opacey/dmgd/btcec"
)

// TestKeyIdMapEquality tests checking equality between multiple keyIdMaps.
func TestKeyIdMapEquality(t *testing.T) {
	var keyIdMapA btcec.KeyIdMap
	var keyIdMapB btcec.KeyIdMap
	keyIdMapA = make(map[btcec.KeyID]*btcec.PublicKey)
	keyIdMapB = make(map[btcec.KeyID]*btcec.PublicKey)

	// Test to make sure that empty key ids are equal
	if !keyIdMapA.Equal(keyIdMapB) {
		t.Fatalf("Expected that empty maps are equivalent")
	}

	pubKey1, err := btcec.ParsePubKey(
		[]byte{0x02, 0xce, 0x0b, 0x14, 0xfb, 0x84, 0x2b, 0x1b,
			0xa5, 0x49, 0xfd, 0xd6, 0x75, 0xc9, 0x80, 0x75, 0xf1,
			0x2e, 0x9c, 0x51, 0x0f, 0x8e, 0xf5, 0x2b, 0xd0, 0x21,
			0xa9, 0xa1, 0xf4, 0x80, 0x9d, 0x3b, 0x4d,
		},
		btcec.S256(),
	)
	if err != nil {
		t.Fatalf("failed to parse raw bytes for pubKey1: %v", err)
	}
	pubKey2, err := btcec.ParsePubKey(
		[]byte{0x03, 0x26, 0x89, 0xc7, 0xc2, 0xda, 0xb1, 0x33,
			0x09, 0xfb, 0x14, 0x3e, 0x0e, 0x8f, 0xe3, 0x96, 0x34,
			0x25, 0x21, 0x88, 0x7e, 0x97, 0x66, 0x90, 0xb6, 0xb4,
			0x7f, 0x5b, 0x2a, 0x4b, 0x7d, 0x44, 0x8e,
		},
		btcec.S256(),
	)
	if err != nil {
		t.Fatalf("failed to parse raw bytes for pubKey2: %v", err)
	}
	keyIdMapA[btcec.KeyID(1)] = pubKey1
	keyIdMapB[btcec.KeyID(2)] = pubKey2

	// Test different keys in a set of equal length are not equal
	if keyIdMapA.Equal(keyIdMapB) {
		t.Fatalf("Expected equal length, different keys not equal")
	}
}

// TestKeyIdMapDeepCopy tests deep copying a keyIdMap.
func TestKeyIdMapDeepCopy(t *testing.T) {
	var keyIdMapA btcec.KeyIdMap
	var keyIdMapB btcec.KeyIdMap
	keyIdMapA = make(map[btcec.KeyID]*btcec.PublicKey)
	pubKey1, err := btcec.ParsePubKey(
		[]byte{0x02, 0xce, 0x0b, 0x14, 0xfb, 0x84, 0x2b, 0x1b,
			0xa5, 0x49, 0xfd, 0xd6, 0x75, 0xc9, 0x80, 0x75, 0xf1,
			0x2e, 0x9c, 0x51, 0x0f, 0x8e, 0xf5, 0x2b, 0xd0, 0x21,
			0xa9, 0xa1, 0xf4, 0x80, 0x9d, 0x3b, 0x4d,
		},
		btcec.S256(),
	)
	if err != nil {
		t.Fatalf("failed to parse raw bytes for pubKey1: %v", err)
	}
	pubKey2, err := btcec.ParsePubKey(
		[]byte{0x03, 0x26, 0x89, 0xc7, 0xc2, 0xda, 0xb1, 0x33,
			0x09, 0xfb, 0x14, 0x3e, 0x0e, 0x8f, 0xe3, 0x96, 0x34,
			0x25, 0x21, 0x88, 0x7e, 0x97, 0x66, 0x90, 0xb6, 0xb4,
			0x7f, 0x5b, 0x2a, 0x4b, 0x7d, 0x44, 0x8e,
		},
		btcec.S256(),
	)
	if err != nil {
		t.Fatalf("failed to parse raw bytes for pubKey2: %v", err)
	}
	keyIdMapA[btcec.KeyID(1)] = pubKey1
	keyIdMapA[btcec.KeyID(2)] = pubKey2

	// Test that a copied key id map is equal to the original.
	keyIdMapB = keyIdMapA.DeepCopy()
	if !keyIdMapA.Equal(keyIdMapB) {
		t.Fatalf("Expected DeepCopy keyIdMap equals original KeyIdMap")
	}
}
