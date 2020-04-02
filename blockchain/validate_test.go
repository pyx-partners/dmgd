// Copyright (c) 2013-2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package blockchain_test

import (
	"bytes"
	"encoding/hex"
	"github.com/pyx-partners/dmgd/blockchain"
	"github.com/pyx-partners/dmgd/btcec"
	"github.com/pyx-partners/dmgd/chaincfg"
	"github.com/pyx-partners/dmgd/chaincfg/chainhash"
	"github.com/pyx-partners/dmgd/provautil"
	"github.com/pyx-partners/dmgd/txscript"
	"github.com/pyx-partners/dmgd/wire"
	"testing"
	"time"
)

// TestCalcBlockSubsidy tests the block subsidy calculation to ensure it
// returns the expected subsidy value.
func TestCalcBlockSubsidy(t *testing.T) {
	subsidy := blockchain.CalcBlockSubsidy(0, &chaincfg.MainNetParams)

	if subsidy != 0*provautil.AtomsPerGram {
		t.Errorf("TestCalcBlockSubsidy: inconsistent initial block "+
			"subsidy %v", subsidy)
	}
}

// TestSequenceLocksActive tests the SequenceLockActive function to ensure it
// works as expected in all possible combinations/scenarios.
func TestSequenceLocksActive(t *testing.T) {
	seqLock := func(h int32, s int64) *blockchain.SequenceLock {
		return &blockchain.SequenceLock{
			Seconds:     s,
			BlockHeight: h,
		}
	}

	tests := []struct {
		seqLock     *blockchain.SequenceLock
		blockHeight int32
		mtp         time.Time

		want bool
	}{
		// Block based sequence lock with equal block height.
		{seqLock: seqLock(1000, -1), blockHeight: 1001, mtp: time.Unix(9, 0), want: true},

		// Time based sequence lock with mtp past the absolute time.
		{seqLock: seqLock(-1, 30), blockHeight: 2, mtp: time.Unix(31, 0), want: true},

		// Block based sequence lock with current height below seq lock block height.
		{seqLock: seqLock(1000, -1), blockHeight: 90, mtp: time.Unix(9, 0), want: false},

		// Time based sequence lock with current time before lock time.
		{seqLock: seqLock(-1, 30), blockHeight: 2, mtp: time.Unix(29, 0), want: false},

		// Block based sequence lock at the same height, so shouldn't yet be active.
		{seqLock: seqLock(1000, -1), blockHeight: 1000, mtp: time.Unix(9, 0), want: false},

		// Time based sequence lock with current time equal to lock time, so shouldn't yet be active.
		{seqLock: seqLock(-1, 30), blockHeight: 2, mtp: time.Unix(30, 0), want: false},
	}

	t.Logf("Running %d sequence locks tests", len(tests))
	for i, test := range tests {
		got := blockchain.SequenceLockActive(test.seqLock,
			test.blockHeight, test.mtp)
		if got != test.want {
			t.Fatalf("SequenceLockActive #%d got %v want %v", i,
				got, test.want)
		}
	}
}

// TestCheckConnectBlock tests the CheckConnectBlock function to ensure it
// fails.
func TestCheckConnectBlock(t *testing.T) {
	// Create a new database and chain instance to run tests against.
	chain, teardownFunc, err := chainSetup("checkconnectblock",
		&chaincfg.MainNetParams)
	if err != nil {
		t.Errorf("Failed to setup chain instance: %v", err)
		return
	}
	defer teardownFunc()

	// The genesis block should fail to connect since it's already inserted.
	genesisBlock := chaincfg.MainNetParams.GenesisBlock
	err = chain.CheckConnectBlock(provautil.NewBlock(genesisBlock))
	if err == nil {
		t.Errorf("CheckConnectBlock: Did not received expected error")
	}
}

// TestCheckBlockSanity tests the CheckBlockSanity function to ensure it works
// as expected.
func TestCheckBlockSanity(t *testing.T) {
	powLimit := chaincfg.MainNetParams.PowLimit
	block := provautil.NewBlock(&SomeBlock)
	timeSource := blockchain.NewMedianTime()
	err := blockchain.CheckBlockSanity(block, powLimit, timeSource)
	if err != nil {
		t.Errorf("CheckBlockSanity: %v", err)
	}

	// Ensure a block that has a timestamp with a precision higher than one
	// second fails.
	timestamp := block.MsgBlock().Header.Timestamp
	block.MsgBlock().Header.Timestamp = timestamp.Add(time.Nanosecond)
	err = blockchain.CheckBlockSanity(block, powLimit, timeSource)
	if err == nil {
		t.Errorf("CheckBlockSanity: error is nil when it shouldn't be")
	}
}

// SomeBlock is used to test Block operations.
var SomeBlock = wire.MsgBlock{
	Header: wire.BlockHeader{
		Version:    1,
		PrevBlock:  chainhash.Hash([32]byte{119, 45, 217, 219, 98, 171, 237, 137, 252, 106, 160, 236, 130, 80, 139, 241, 92, 104, 211, 86, 65, 166, 88, 18, 125, 35, 43, 22, 129, 219, 112, 44}),
		MerkleRoot: chainhash.Hash([32]byte{248, 53, 49, 152, 191, 87, 162, 223, 132, 68, 215, 125, 18, 36, 14, 37, 29, 27, 141, 50, 2, 62, 204, 216, 18, 164, 54, 199, 110, 74, 23, 254}),
		Timestamp:  time.Unix(1486467380, 0), //
		Bits:       0x2000000f,               //
		Size:       0x00000132,               // 306
		Nonce:      0x00000019,               // 25
	},
	Transactions: []*wire.MsgTx{
		{
			Version: 1,
			TxIn: []*wire.TxIn{
				{
					PreviousOutPoint: wire.OutPoint{
						Hash:  chainhash.Hash{},
						Index: 0xffffffff,
					},
					SignatureScript: []byte{
						0x06, 0x2f, 0x72, 0x6d, 0x67, 0x64, 0x2f,
					},
					Sequence: 0xffffffff,
				},
			},
			TxOut: []*wire.TxOut{
				{
					Value:    0x1388, // 5000
					PkScript: []byte{82, 20, 53, 219, 191, 4, 188, 160, 97, 228, 157, 172, 224, 143, 133, 141, 135, 117, 192, 165, 124, 142, 3, 0, 0, 1, 81, 83, 186},
				},
			},
			LockTime: 0,
		},
	},
}

// TestCheckTransactionSanity tests the CheckTransactionSanity API.
func TestCheckTransactionSanity(t *testing.T) {
	// Create some dummy, but otherwise standard, data for transactions.
	prevOutHash, err := chainhash.NewHashFromStr("01")
	if err != nil {
		t.Fatalf("NewShaHashFromStr: unexpected error: %v", err)
	}
	dummyPrevOut1 := wire.OutPoint{Hash: *prevOutHash, Index: 1}
	dummyPrevOut2 := wire.OutPoint{Hash: *prevOutHash, Index: 2}
	dummySigScript := bytes.Repeat([]byte{0x00}, 65)
	dummyTxIn := wire.TxIn{
		PreviousOutPoint: dummyPrevOut1,
		SignatureScript:  dummySigScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	dummyTxIn2 := wire.TxIn{
		PreviousOutPoint: dummyPrevOut2,
		SignatureScript:  dummySigScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}

	// Create prova txout
	keyId1 := btcec.KeyID(1)
	keyId2 := btcec.KeyID(2)
	payAddr, _ := provautil.NewAddressProva(make([]byte, 20), []btcec.KeyID{keyId1, keyId2}, &chaincfg.RegressionNetParams)
	provaPkScript, _ := txscript.PayToAddrScript(payAddr)
	provaTxOut := wire.TxOut{
		Value:    300,
		PkScript: provaPkScript,
	}

	// Create some dummy admin op output.
	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0x2b, 0x8c, 0x52, 0xb7, 0x7b, 0x32, 0x7c, 0x75,
		0x5b, 0x9b, 0x37, 0x55, 0x00, 0xd3, 0xf4, 0xb2,
		0xda, 0x9b, 0x0a, 0x1f, 0xf6, 0x5f, 0x68, 0x91,
		0xd3, 0x11, 0xfe, 0x94, 0x29, 0x5b, 0xc2, 0x6a,
	})
	data := make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = txscript.AdminOpProvisionKeyAdd
	copy(data[1:], pubKey.SerializeCompressed())
	adminOpPkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: adminOpPkScript,
	}
	// create root tx out
	rootPkScript, _ := txscript.ProvaThreadScript(provautil.RootThread)
	rootTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: rootPkScript,
	}
	// create provision tx out
	provisionPkScript, _ := txscript.ProvaThreadScript(provautil.ProvisionThread)
	provisionTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: provisionPkScript,
	}
	// create provision tx out
	issuePkScript, _ := txscript.ProvaThreadScript(provautil.IssueThread)
	issueTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: issuePkScript,
	}

	tests := []struct {
		name    string
		tx      wire.MsgTx
		isValid bool
		code    blockchain.ErrorCode
	}{
		{
			name: "Typical admin transaction",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			isValid: true,
		},
		{
			name: "Typical issue transaction",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&issueTxOut, &provaTxOut},
				LockTime: 0,
			},
			isValid: true,
		},
		{
			name: "Issue thread with admin op",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&issueTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Issue thread with 0 prova output",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    0,
					PkScript: provaPkScript,
				}},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Issue thread with invalid output",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    0,
					PkScript: []byte{txscript.OP_TRUE},
				}},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Issue thread burning 0 coins",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    0,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Issue thread trying to issue and destroy.",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn}, // only one input => issuance
				TxOut: []*wire.TxOut{&issueTxOut, { // yet admin op => destruction
					Value:    100,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "admin transaction with thread output at pos 1.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&adminOpTxOut, &rootTxOut},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "admin transaction with non-zero output value.",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{&rootTxOut, {
					Value:    500,
					PkScript: adminOpPkScript,
				}},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "admin transaction with more than 1 input.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn, &dummyTxIn2},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Empty admin transaction",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Admin transaction with operation on wrong thread",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&provisionTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
		{
			name: "Admin transaction with invalid operation",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				TxOut: []*wire.TxOut{&rootTxOut, {
					Value:    0,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminTx,
		},
	}

	for _, test := range tests {
		// Ensure standardness is as expected.
		err := blockchain.CheckTransactionSanity(provautil.NewTx(&test.tx))
		if err == nil && test.isValid {
			// Test passes since function returned standard for a
			// transaction which is intended to be standard.
			continue
		}
		if err == nil && !test.isValid {
			t.Errorf("CheckTransactionSanity (%s): standard when "+
				"it should not be", test.name)
			continue
		}
		if err != nil && test.isValid {
			t.Errorf("CheckTransactionSanity (%s): nonstandard "+
				"when it should not be: %v", test.name, err)
			continue
		}

		// Ensure error type is a TxRuleError inside of a RuleError.
		rerr, ok := err.(blockchain.RuleError)
		if !ok {
			t.Errorf("CheckTransactionSanity (%s): unexpected "+
				"error type - got %T", test.name, err)
			continue
		}

		// Ensure the reject code is the expected one.
		if rerr.ErrorCode != test.code {
			t.Errorf("CheckTransactionSanity (%s): unexpected "+
				"error code - got %v, want %v", test.name,
				rerr.ErrorCode, test.code)
			continue
		}
	}
}

// hexToBytes converts the passed hex string into bytes and will panic if there
// is an error.  This is only provided for the hard-coded constants so errors in
// the source code can be detected. It will only (and must only) be called with
// hard-coded values.
func hexToBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic("invalid hex in source file: " + s)
	}
	return b
}

// TestCheckTransactionOutputs tests the CheckTransactionOutputs API.
func TestCheckTransactionOutputs(t *testing.T) {
	// Create some dummy, but otherwise standard, data for transactions.
	prevOutHash, err := chainhash.NewHashFromStr("01")
	if err != nil {
		t.Fatalf("NewShaHashFromStr: unexpected error: %v", err)
	}
	dummyPrevOut1 := wire.OutPoint{Hash: *prevOutHash, Index: 1}
	dummySigScript := bytes.Repeat([]byte{0x00}, 65)
	dummyTxIn := wire.TxIn{
		PreviousOutPoint: dummyPrevOut1,
		SignatureScript:  dummySigScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	// Create prova txout
	keyId1 := btcec.KeyID(1)
	keyId2 := btcec.KeyID(2)
	payAddr, _ := provautil.NewAddressProva(make([]byte, 20), []btcec.KeyID{keyId1, keyId2}, &chaincfg.RegressionNetParams)
	provaPkScript, _ := txscript.PayToAddrScript(payAddr)
	provaTxOut := wire.TxOut{
		Value:    0, // 0 atoms
		PkScript: provaPkScript,
	}
	// Create null txout
	nullScript, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_RETURN).Script()
	if err != nil {
		t.Fatalf("NewScriptBuilder: unexpected error: %v", err)
	}
	nullTxOut := wire.TxOut{
		Value:    0, // 0 atoms
		PkScript: nullScript,
	}

	// Create admin op to add provision key.
	_, pubKey := btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0x2b, 0x8c, 0x52, 0xb7, 0x7b, 0x32, 0x7c, 0x75,
		0x5b, 0x9b, 0x37, 0x55, 0x00, 0xd3, 0xf4, 0xb2,
		0xda, 0x9b, 0x0a, 0x1f, 0xf6, 0x5f, 0x68, 0x91,
		0xd3, 0x11, 0xfe, 0x94, 0x29, 0x5b, 0xc2, 0x6a,
	})
	data := make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = txscript.AdminOpProvisionKeyAdd
	copy(data[1:], pubKey.SerializeCompressed())
	adminOpPkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: adminOpPkScript,
	}
	// Create admin op to revoke provision key.
	data = make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = txscript.AdminOpProvisionKeyRevoke
	copy(data[1:], pubKey.SerializeCompressed())
	adminOpRevokePkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpRevokeTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: adminOpRevokePkScript,
	}
	// Create admin op to revoke validate key.
	data = make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = txscript.AdminOpValidateKeyRevoke
	copy(data[1:], pubKey.SerializeCompressed())
	adminOpRevProvPkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpRevProvTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: adminOpRevProvPkScript,
	}
	// Create admin op to add keyID.
	keyID := btcec.KeyID(2)
	data = make([]byte, 1+btcec.PubKeyBytesLenCompressed+btcec.KeyIDSize)
	data[0] = txscript.AdminOpASPKeyAdd
	copy(data[1:], pubKey.SerializeCompressed())
	keyID.ToAddressFormat(data[1+btcec.PubKeyBytesLenCompressed:])
	adminOpAspPkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpAspTxOut := wire.TxOut{
		Value:    0,
		PkScript: adminOpAspPkScript,
	}
	// Create admin op to add next key ID
	btcec.KeyID(3).ToAddressFormat(data[1+btcec.PubKeyBytesLenCompressed:])
	adminOpAsp2PkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpAsp2TxOut := wire.TxOut{
		Value:    0,
		PkScript: adminOpAsp2PkScript,
	}
	// dummy asp script
	data = make([]byte, 1+btcec.PubKeyBytesLenCompressed+btcec.KeyIDSize)
	data[0] = txscript.AdminOpASPKeyAdd
	copy(data[1:], bytes.Repeat([]byte{0x00}, btcec.PubKeyBytesLenCompressed))
	keyID.ToAddressFormat(data[1+btcec.PubKeyBytesLenCompressed:])
	dummyOpAspPkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	dummyOpAspTxOut := wire.TxOut{
		Value:    0,
		PkScript: dummyOpAspPkScript,
	}
	// Create admin op to revoke keyID.
	data = make([]byte, 1+btcec.PubKeyBytesLenCompressed+btcec.KeyIDSize)
	data[0] = txscript.AdminOpASPKeyRevoke
	copy(data[1:], pubKey.SerializeCompressed())
	keyID.ToAddressFormat(data[1+btcec.PubKeyBytesLenCompressed:])
	adminOpAspRevPkScript, _ := txscript.NewScriptBuilder().AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	adminOpAspRevTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: adminOpAspRevPkScript,
	}
	// create root tx out
	rootPkScript, _ := txscript.ProvaThreadScript(provautil.RootThread)
	rootTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: rootPkScript,
	}
	// create issue tx out
	issuePkScript, _ := txscript.ProvaThreadScript(provautil.IssueThread)
	issueTxOut := wire.TxOut{
		Value:    0, // 0 DMG
		PkScript: issuePkScript,
	}

	tests := []struct {
		name         string
		tx           wire.MsgTx
		lastKeyID    btcec.KeyID
		adminKeySets map[btcec.KeySetType]btcec.PublicKeySet
		aspKeyIdMap  btcec.KeyIdMap
		isCoinbase   bool
		isValid      bool
		code         blockchain.ErrorCode
	}{
		{
			name: "Spend to regular Prova output.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&provaTxOut},
				LockTime: 0,
			},
			aspKeyIdMap: func() btcec.KeyIdMap {
				keyId1 := btcec.KeyID(1)
				keyId2 := btcec.KeyID(2)
				return map[btcec.KeyID]*btcec.PublicKey{keyId1: pubKey, keyId2: pubKey}
			}(),
			isValid: true,
		},
		{
			name: "Spend to Prova with unknown keyID.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&provaTxOut},
				LockTime: 0,
			},
			aspKeyIdMap: func() btcec.KeyIdMap {
				keyId1 := btcec.KeyID(1)
				return map[btcec.KeyID]*btcec.PublicKey{keyId1: pubKey}
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidTx,
		},
		{
			name: "Add key to empty admin set.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			isValid: true,
		},
		{
			name: "Revoking last key from provision set.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpRevokeTxOut},
				LockTime: 0,
			},
			adminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
				keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)
				keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
					"038ef4a121bcaf1b1f175557a12896f8bc93b095e84817f90e9a901cd2113a8202", // priv 2b8c52b77b327c755b9b375500d3f4b2da9b0a1ff65f6891d311fe94295bc26a
				)
				return keySets
			}(),
			isValid: true,
		},
		{
			name: "Adding existing key to set.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			adminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
				keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)
				keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
					"038ef4a121bcaf1b1f175557a12896f8bc93b095e84817f90e9a901cd2113a8202", // priv 2b8c52b77b327c755b9b375500d3f4b2da9b0a1ff65f6891d311fe94295bc26a
				)
				return keySets
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Adding key to full set.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpTxOut},
				LockTime: 0,
			},
			adminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
				keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)
				// fill up the provision key set
				for i := 0; i < blockchain.MaxAdminKeySetSize; i++ {
					privKey, _ := btcec.NewPrivateKey(btcec.S256())
					pubKey := (*btcec.PublicKey)(&privKey.PublicKey)
					keySets[btcec.ProvisionKeySet] = keySets[btcec.ProvisionKeySet].Add(pubKey)
				}
				return keySets
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Revoking non-existing key from set.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpRevokeTxOut},
				LockTime: 0,
			},
			adminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
				keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)
				keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
					"038364914c537fc6c6a675166aea88abf7a2c83b0955b2e6b0611dacfad6242288",
					"0353cc1a8e6fcb764349bce68a56a285316bcea950a6f667fee4c95d5ad2f72815",
					"0324d2903ef1c4f0df2d47cd39184e667bd32d101a319c47ed47a4941f62a1b886",
				)
				return keySets
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Revoking too many from validate set.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpRevProvTxOut},
				LockTime: 0,
			},
			adminKeySets: func() map[btcec.KeySetType]btcec.PublicKeySet {
				keySets := make(map[btcec.KeySetType]btcec.PublicKeySet)
				keySets[btcec.ProvisionKeySet], _ = btcec.ParsePubKeySet(btcec.S256(),
					"025ceeba2ab4a635df2c0301a3d773da06ac5a18a7c3e0d09a795d7e57d233edf1",
					"0353cc1a8e6fcb764349bce68a56a285316bcea950a6f667fee4c95d5ad2f72815",
					"038ef4a121bcaf1b1f175557a12896f8bc93b095e84817f90e9a901cd2113a8202",
				)
				return keySets
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Adding a new keyID.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspTxOut},
				LockTime: 0,
			},
			lastKeyID: btcec.KeyID(1),
			isValid:   true,
		},
		{
			name: "provision keyID that is too high.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&provaTxOut, &adminOpAspTxOut},
				LockTime: 0,
			},
			lastKeyID: btcec.KeyID(4),
			isValid:   false,
			code:      blockchain.ErrInvalidTx,
		},
		{
			name: "provision keyID 2 times in same tx.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspTxOut, &dummyOpAspTxOut},
				LockTime: 0,
			},
			lastKeyID: btcec.KeyID(1),
			isValid:   false,
			code:      blockchain.ErrInvalidAdminOp,
		},
		{
			name: "provision 2 increcemental keyIDs in same tx.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspTxOut, &adminOpAsp2TxOut},
				LockTime: 0,
			},
			lastKeyID: btcec.KeyID(1),
			isValid:   true,
		},
		{
			name: "Add an existing keyID.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspTxOut},
				LockTime: 0,
			},
			aspKeyIdMap: func() btcec.KeyIdMap {
				return map[btcec.KeyID]*btcec.PublicKey{keyID: pubKey}
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Revoke same keyID multiple times in one tx.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspRevTxOut, &adminOpAspRevTxOut},
				LockTime: 0,
			},
			aspKeyIdMap: func() btcec.KeyIdMap {
				keyId1 := btcec.KeyID(2)
				pubKey1, _ := btcec.ParsePubKey(hexToBytes("025ceeba2ab4a635df2c0301a3d773da06ac5a18a7c3e0d09a795d7e57d233edf1"), btcec.S256())
				return map[btcec.KeyID]*btcec.PublicKey{keyId1: pubKey1}
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Revoke unknown keyID.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspRevTxOut},
				LockTime: 0,
			},
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Revoke a keyID with non-matching pubKey.",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&dummyTxIn},
				// this admin op revokes keyID 65536 ( []byte{0, 0, 1, 0} )
				// this admin op revokes pubKey 038ef4a121bcaf1b1f175557a12896f8bc93b095e84817f90e9a901cd2113a8202
				TxOut:    []*wire.TxOut{&rootTxOut, &adminOpAspRevTxOut},
				LockTime: 0,
			},
			aspKeyIdMap: func() btcec.KeyIdMap {
				// keyID 65536 ( []byte{0, 0, 1, 0} )
				bogusPubKey, _ := btcec.ParsePubKey(hexToBytes("025ceeba2ab4a635df2c0301a3d773da06ac5a18a7c3e0d09a795d7e57d233edf1"), btcec.S256())
				return map[btcec.KeyID]*btcec.PublicKey{keyID: bogusPubKey}
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidAdminOp,
		},
		{
			name: "Issue to prova output with unknown keyID.",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&issueTxOut, &provaTxOut},
				LockTime: 0,
			},
			aspKeyIdMap: func() btcec.KeyIdMap {
				keyId2 := btcec.KeyID(2)
				return map[btcec.KeyID]*btcec.PublicKey{keyId2: pubKey}
			}(),
			isValid: false,
			code:    blockchain.ErrInvalidTx,
		},
		{
			name: "Spend to a single null data output",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&nullTxOut},
				LockTime: 0,
			},
			isCoinbase: true,
			isValid:    true,
		},
		{
			name: "Spend to multiple null data outputs",
			tx: wire.MsgTx{
				Version:  1,
				TxIn:     []*wire.TxIn{&dummyTxIn},
				TxOut:    []*wire.TxOut{&nullTxOut, &nullTxOut},
				LockTime: 0,
			},
			isCoinbase: true,
			isValid:    false,
			code:       blockchain.ErrInvalidTx,
		},
	}

	for _, test := range tests {
		keyView := blockchain.NewKeyViewpoint()
		keyView.SetKeys(test.adminKeySets)
		keyView.SetLastKeyID(test.lastKeyID)
		keyView.SetKeyIDs(test.aspKeyIdMap)
		tx := provautil.NewTx(&test.tx)
		if test.isCoinbase {
			tx.SetIndex(0)
		}
		err := blockchain.CheckTransactionOutputs(tx, keyView, &chaincfg.RegressionNetParams)
		if err == nil && test.isValid {
			// Test passes since function returned valid for a
			// transaction which is intended to be valid.
			continue
		}
		if err == nil && !test.isValid {
			t.Errorf("CheckTransactionOutputs (%s): valid when "+
				"it should not be", test.name)
			continue
		}
		if err != nil && test.isValid {
			t.Errorf("CheckTransactionOutputs (%s): invalid "+
				"when it should not be: %v", test.name, err)
			continue
		}

		rerr, ok := err.(blockchain.RuleError)
		if !ok {
			t.Errorf("CheckTransactionOutputs (%s): unexpected "+
				"error type - got %T", test.name, err)
			continue
		}

		// Ensure the reject code is the expected one.
		if rerr.ErrorCode != test.code {
			t.Errorf("CheckTransactionOutputs (%s): unexpected "+
				"error code - got %v, want %v", test.name,
				rerr.ErrorCode, test.code)
			continue
		}
	}
}

// TestCheckTransactionInputs tests the CheckTransactionInputs API.
func TestCheckTransactionInputs(t *testing.T) {
	// Create some dummy, but otherwise standard, data for transactions.
	prevOut := wire.TxOut{
		Value:    400000000,
		PkScript: make([]byte, 20),
	}
	prevMsgTx := wire.MsgTx{
		Version:  1,
		TxOut:    []*wire.TxOut{&prevOut},
		LockTime: 0,
	}
	prevTx := provautil.NewTx(&prevMsgTx)
	// Create prova px script
	keyId1 := btcec.KeyID(1)
	keyId2 := btcec.KeyID(2)
	payAddr, _ := provautil.NewAddressProva(make([]byte, 20), []btcec.KeyID{keyId1, keyId2}, &chaincfg.RegressionNetParams)
	provaPkScript, _ := txscript.PayToAddrScript(payAddr)
	// spend prevTx
	dummyPrevOut1 := wire.OutPoint{Hash: *prevTx.Hash(), Index: 0}
	dummySigScript := bytes.Repeat([]byte{0x00}, 65)
	dummyTxIn := wire.TxIn{
		PreviousOutPoint: dummyPrevOut1,
		SignatureScript:  dummySigScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}
	// create issue tip tx
	issuePkScript, _ := txscript.ProvaThreadScript(provautil.IssueThread)
	issueTxOut := wire.TxOut{
		Value:    0, // 0 atoms
		PkScript: issuePkScript,
	}
	issueTipTx := provautil.NewTx(&wire.MsgTx{
		Version:  1,
		TxIn:     []*wire.TxIn{&dummyTxIn},
		TxOut:    []*wire.TxOut{&issueTxOut},
		LockTime: 0,
	})
	issuePrevOut := wire.OutPoint{Hash: *issueTipTx.Hash(), Index: 0}
	issueTxIn := wire.TxIn{
		PreviousOutPoint: issuePrevOut,
		SignatureScript:  dummySigScript,
		Sequence:         wire.MaxTxInSequenceNum,
	}

	tests := []struct {
		name    string
		tx      wire.MsgTx
		height  uint32
		isValid bool
		code    blockchain.ErrorCode
	}{
		{
			name: "destroy some coins.",
			tx: wire.MsgTx{
				Version: 1, // in values: 0      and 400000000
				TxIn:    []*wire.TxIn{&issueTxIn, &dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    400000000,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			height:  200,
			isValid: true,
		},
		{
			name: "destroy more than input.",
			tx: wire.MsgTx{
				Version: 1, // in values: 0      and 400000000
				TxIn:    []*wire.TxIn{&issueTxIn, &dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    500000000,
					PkScript: []byte{txscript.OP_RETURN},
				}},
				LockTime: 0,
			},
			height:  200,
			isValid: false,
			code:    blockchain.ErrSpendTooHigh,
		},
		{
			name: "destroy and spend more than input in same tx.",
			tx: wire.MsgTx{
				Version: 1, // in values: 0      and 400000000
				TxIn:    []*wire.TxIn{&issueTxIn, &dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    300000000, // destroy
					PkScript: []byte{txscript.OP_RETURN},
				}, {
					Value:    300000000, // change
					PkScript: provaPkScript,
				}},
				LockTime: 0,
			},
			height:  200,
			isValid: false,
			code:    blockchain.ErrSpendTooHigh,
		},
		{
			name: "tx pays a fee that does not exceed the limit.",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&issueTxIn, &dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    400000000 - 5000000,
					PkScript: []byte{txscript.OP_RETURN},
				}},
			},
			height:  200,
			isValid: true,
		},
		{
			name: "tx pays a fee that exceeds the fee limit.",
			tx: wire.MsgTx{
				Version: 1,
				TxIn:    []*wire.TxIn{&issueTxIn, &dummyTxIn},
				TxOut: []*wire.TxOut{&issueTxOut, {
					Value:    300000000 - 1,
					PkScript: []byte{txscript.OP_RETURN},
				}},
			},
			height:  200,
			isValid: false,
			code:    blockchain.ErrFeeTooHigh,
		},
	}

	for _, test := range tests {
		utxoView := blockchain.NewUtxoViewpoint()
		utxoView.AddTxOuts(prevTx, 100)
		utxoView.AddTxOuts(issueTipTx, 100)
		_, err := blockchain.CheckTransactionInputs(provautil.NewTx(&test.tx),
			test.height, utxoView, &chaincfg.MainNetParams)
		if err == nil && test.isValid {
			// Test passes since function returned valid for a
			// transaction which is intended to be valid.
			continue
		}
		if err == nil && !test.isValid {
			t.Errorf("CheckTransactionInputs (%s): valid when "+
				"it should not be", test.name)
			continue
		}
		if err != nil && test.isValid {
			t.Errorf("CheckTransactionInputs (%s): invalid "+
				"when it should not be: %v", test.name, err)
			continue
		}

		rerr, ok := err.(blockchain.RuleError)
		if !ok {
			t.Errorf("CheckTransactionInputs (%s): unexpected "+
				"error type - got %T", test.name, err)
			continue
		}

		// Ensure the reject code is the expected one.
		if rerr.ErrorCode != test.code {
			t.Errorf("CheckTransactionInputs (%s): unexpected "+
				"error code - got %v, want %v", test.name,
				rerr.ErrorCode, test.code)
			continue
		}
	}
}
