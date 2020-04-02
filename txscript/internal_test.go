// Copyright (c) 2013-2015 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
This test file is part of the txscript package rather than than the
txscript_test package so it can bridge access to the internals to properly test
cases which are either not possible or can't reliably be tested via the public
interface.  The functions are only exported while the tests are being run.
*/

package txscript

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/wire"
)

// TstHasCanoncialPushes makes the internal isCanonicalPush function available
// to the test package.
var TstHasCanonicalPushes = canonicalPush

// TstParseScript makes the internal parseScript function available to the
// test package.
var TstParseScript = ParseScript

// TstCalcSignatureHash makes the internal calcSignatureHash function available
// to the test package.
var TstCalcSignatureHash = calcSignatureHash

// TstConcatRawScript makes the ability to add the pass script directly to
// an existing script to the test package.  This differs from AddData since it
// doesn't add a push data opcode.
func (b *ScriptBuilder) TstConcatRawScript(data []byte) *ScriptBuilder {
	if b.err != nil {
		return b
	}

	b.script = append(b.script, data...)
	return b
}

// TstCheckPubKeyEncoding makes the internal checkPubKeyEncoding function
// available to the test package.  Since it only really needs from the engine
// for the flags, just accept the flags and create a new engine skeleton.
func TstCheckPubKeyEncoding(pubKey []byte, flags ScriptFlags) error {
	vm := Engine{flags: flags}
	return vm.checkPubKeyEncoding(pubKey)
}

// TstCheckSignatureEncoding makes the internal checkSignatureEncoding function
// available to the test package.  Since it only really needs from the engine
// for the flags, just accept the flags and create a new engine skeleton with
// them.
func TstCheckSignatureEncoding(sig []byte, flags ScriptFlags) error {
	vm := Engine{flags: flags}
	return vm.checkSignatureEncoding(sig)
}

// TstRemoveOpcode makes the internal removeOpcode function available to the
// test package.
func TstRemoveOpcode(pkscript []byte, opcode byte) ([]byte, error) {
	pops, err := ParseScript(pkscript)
	if err != nil {
		return nil, err
	}
	pops = removeOpcode(pops, opcode)
	return UnparseScript(pops)
}

// TstRemoveOpcodeByData makes the internal removeOpcodeByData function
// available to the test package.
func TstRemoveOpcodeByData(pkscript []byte, data []byte) ([]byte, error) {
	pops, err := ParseScript(pkscript)
	if err != nil {
		return nil, err
	}
	pops = removeOpcodeByData(pops, data)
	return UnparseScript(pops)
}

// TestSetPC allows the test modules to set the program counter to whatever they
// want.
func (vm *Engine) TstSetPC(script, off int) {
	vm.scriptIdx = script
	vm.scriptOff = off
}

// TestSigHashNew tests that calcWitnessSignatureHash according to the digest scheme defined for Prova.
func TestSigHashNew(t *testing.T) {
	// Decode the serialized, unsigned transaction used within the BIP as an example:
	//
	// nVersion:  01000000
	// txin:      02 fff7f7881a8099afa6940d42d1e7f6362bec38171ea3edf433541db4e4ad969f 00000000 00 eeffffff
	//               ef51e1b804cc89d182d279655c3aa89e815b1b309fe287d9b2b55d57b90ec68a 01000000 00 ffffffff
	// txout:     02 202cb20600000000 1976a9148280b37df378db99f66f85c95a783a76ac7a6d5988ac
	//               9093510d00000000 1976a9143bde42dbee7e4dbe6a21b2d50ce2f0167faa815988ac
	// nLockTime: 11000000

	bip143TxEncodedUnsigned := "0100000002fff7f7881a8099afa6940d42d1e7f6362bec38171ea3edf433541db4e4ad969f0000000000eeffffffef51e1b804cc89d182d279655c3aa89e815b1b309fe287d9b2b55d57b90ec68a0100000000ffffffff02202cb206000000001976a9148280b37df378db99f66f85c95a783a76ac7a6d5988ac9093510d000000001976a9143bde42dbee7e4dbe6a21b2d50ce2f0167faa815988ac11000000"
	txRaw, err := hex.DecodeString(bip143TxEncodedUnsigned)
	if err != nil {
		t.Fatalf("unable to decode tx: %v", err)
	}
	r := bytes.NewReader(txRaw)

	tx := wire.NewMsgTx(1)
	if err := tx.Deserialize(r); err != nil {
		t.Fatalf("unable to decode: %v", err)
	}

	// Create a new HashCache adding the intermediate sigHashes of this
	// tx to it.
	hashCache := NewHashCache(90)
	hashCache.AddSigHashes(tx)
	hash := tx.TxHash()
	txSigHashes, found := hashCache.GetSigHashes(&hash)
	if !found {
		t.Fatalf("unable to find sighashes")
	}

	// We'll be generating the sighash for the second input, using sighash
	// all, the proper input amount, and with the corresponding pkScript.
	idx := 1
	shType := SigHashAll
	amt := provautil.Amount(6e8)
	pkScriptEncoded := "00141d0f172a0ecb48aee1be1f2687d2963ae33f71a1"
	decodedScript, err := hex.DecodeString(pkScriptEncoded)
	if err != nil {
		t.Fatalf("unable to decode script")
	}
	opCodes, err := ParseScript(decodedScript)
	if err != nil {
		t.Fatalf("unable to decode script: %v", err)
	}

	// Finally, calculate the sigHash by digest scheme defined for Prova.
	// nVersion:     01000000
	// hashPrevouts: 96b827c8483d4e9b96712b6713a7b68d6e8003a781feba36c31143470b4efd37
	// hashSequence: 52b0a642eea2fb7ae638c36f6252b6750293dbe574a806984b8e4d8548339a3b
	// outpoint:     ef51e1b804cc89d182d279655c3aa89e815b1b309fe287d9b2b55d57b90ec68a01000000
	// scriptCode:   !!we don't encode the script code!!
	// amount:       0046c32300000000
	// nSequence:    ffffffff
	// hashOutputs:  863ef3e1a92afbfdb97f31ad0fc7683ee943e9abcf2501590ff8f6551f47e5e5
	// nLockTime:    11000000
	// nHashType:    01000000
	//
	// because we leave out the scriptCode, the preimage is now different than in the BIP 143 example:
	// the new preimage: 0100000096b827c8483d4e9b96712b6713a7b68d6e8003a781feba36c31143470b4efd3752b0a642eea2fb7ae638c36f6252b6750293dbe574a806984b8e4d8548339a3bef51e1b804cc89d182d279655c3aa89e815b1b309fe287d9b2b55d57b90ec68a010000000046c32300000000ffffffff863ef3e1a92afbfdb97f31ad0fc7683ee943e9abcf2501590ff8f6551f47e5e51100000001000000
	// which should hash256(hash256(preimage)) to expectedHash below.
	sigHash := calcSignatureHashNew(opCodes, txSigHashes, shType, tx, idx, int64(amt))
	expectedHash := "f235bc64db1070171c021a6b8e4b557fffebad26ffe728a6815e512154ea8556"
	if hex.EncodeToString(sigHash) != expectedHash {
		t.Fatalf("sig hashes don't match, expected %v, got %v",
			expectedHash, hex.EncodeToString(sigHash))
	}
}
