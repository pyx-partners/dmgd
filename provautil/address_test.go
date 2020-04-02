// Copyright (c) 2013, 2014 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package provautil_test

import (
	"fmt"
	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/btcsuite/golangcrypto/ripemd160"
	"testing"
)

func TestAddresses(t *testing.T) {
	tests := []struct {
		addr   string
		keyIDs []btcec.KeyID
		name   string
		net    *chaincfg.Params
		pkHash []byte
		valid  bool
	}{
		{
			addr:   "G9n66A3tweNBdnrWHtPQhojDvgtTHpKp5Ke5EeHqZM1pv",
			keyIDs: []btcec.KeyID{1, 2},
			name:   "mainnet standard address",
			net:    &chaincfg.MainNetParams,
			pkHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			valid:  true,
		},
		{
			addr:   "CBmenNb1jH2fkDXKuEdqUgj3BaLqnGaGAAn6qMQXVv1KzyG8inv6UHMM",
			keyIDs: []btcec.KeyID{1, 2, 3, 4},
			name:   "mainnet 4 of 5 address",
			net:    &chaincfg.MainNetParams,
			pkHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			valid:  true,
		},
		{
			addr:   "T9GooXEi927U4tuUkHsyfxtuDwAGFP2RaDXNGVNchBSz3",
			keyIDs: []btcec.KeyID{1, 2},
			name:   "testnet standard address",
			net:    &chaincfg.TestNetParams,
			pkHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			valid:  true,
		},
		{
			addr:   "9kT6DCNXS7BcCexSzVMXJ8aZXYxjB4ZGhBqGEU4DKm3vdyZDPbFDPV32np61p76X2Hjepz412YLpoNzwyBuWAWTqsq75ApLf6VaM8zApsumNDXnLLBqR9ZoYDYJgqQU4w2kFMP1e1A",
			keyIDs: []btcec.KeyID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19},
			name:   "large number of key ids",
			net:    &chaincfg.MainNetParams,
			pkHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			valid:  true,
		},
		{
			addr:   "4zia3uJBAsmyZhhjNanqsNPThfGJ6R6BwYuJ6fKc",
			keyIDs: []btcec.KeyID{1},
			name:   "only one key id",
			net:    &chaincfg.TestNetParams,
			pkHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			valid:  false,
		},
		{
			addr:   "zFaDkPWYXYgRUvMPFSeE9N7Eu1Hi3WrECgzJLQ1Ye4f1T2dZKgktB3V1U2sqgWYgoMK6Z5RozfmrkwgUxNsnffq21KumgyyJn6rukerMueseUtrgpbU2nKRr3VMHhKRSn5pEACLLKv2eoVz",
			keyIDs: []btcec.KeyID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			name:   "large number of key ids",
			net:    &chaincfg.MainNetParams,
			pkHash: []byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
			valid:  false,
		},
	}

	for _, test := range tests {
		// Encode the address and check for expected errors
		addr, err := provautil.NewAddressProva(test.pkHash,
			test.keyIDs, test.net)
		if (err == nil) != test.valid {
			t.Errorf("%v: encoding test failed: %v", test.name, err)
		}
		// Exit early for expected errors
		if err != nil {
			continue
		}

		// Compare encoded address against expected
		if test.addr != addr.EncodeAddress() {
			t.Errorf("%v: encoding test failed, got addr %v, expected %v", test.name, addr.EncodeAddress(), test.addr)
		}

		// Decode addr and compare error against valid.
		decoded, err := provautil.DecodeAddress(test.addr, test.net)
		if (err == nil) != test.valid {
			t.Errorf("%v: decoding test failed: %v", test.name, err)
			return
		}

		// Exit early for expected errors.
		if err != nil {
			continue
		}

		// Check that the number of keyids is as expected.
		if len(test.keyIDs) != len(decoded.ScriptKeyIDs()) {
			t.Errorf("%v: keyid counts do not match: %d vs %d", test.name, len(test.keyIDs), len(decoded.ScriptKeyIDs()))
			return
		}

		// Check that the keyids are in the equivalent positions.
		for i, keyID := range decoded.ScriptKeyIDs() {
			if test.keyIDs[i] != keyID {
				t.Errorf("%v: keyid does not match: got %d expected %d", test.name, keyID, test.keyIDs[i])
				return
			}
		}

		// Check the script address (public key hash) length.
		if len(decoded.ScriptAddress()) != ripemd160.Size {
			t.Errorf("%v: pkhash is incorrect size: got %d expected %d", test.name, len(decoded.ScriptAddress()), ripemd160.Size)
			return
		}

		// Check that the pkhash bytes are as expected
		for i, pkHashByte := range decoded.ScriptAddress() {
			if test.pkHash[i] != pkHashByte {
				t.Errorf("%v: pkhash does not match: got %v expected %v", test.name, decoded.ScriptAddress(), test.pkHash)
				return
			}
		}

		// Ensure the stringer returns the same address as the
		// original.
		if decodedStringer, ok := decoded.(fmt.Stringer); ok {
			if test.addr != decodedStringer.String() {
				t.Errorf("%v: String on decoded value does not match expected value: %v != %v",
					test.name, test.addr, decodedStringer.String())
				return
			}
		}

		// Encode again and compare against the original.
		encoded := decoded.EncodeAddress()
		if test.addr != encoded {
			t.Errorf("%v: decoding and encoding produced different addressess: %v != %v",
				test.name, test.addr, encoded)
			return
		}

		// Ensure the address is for the expected network.
		if !decoded.IsForNet(test.net) {
			t.Errorf("%v: calculated network does not match expected",
				test.name)
			return
		}
	}
}
