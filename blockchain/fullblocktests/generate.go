// Copyright (c) 2016 The btcsuite developers
// Copyright (c) 2017 BitGo
// Copyright (c) 2019 Tranquility Node Ltd
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// The vast majority of the rules tested in this package were ported from the
// the original Java-based 'official' block acceptance tests at
// https://github.com/TheBlueMatt/test-scripts as well as some additional tests
// available in the Core python port of the same.

package fullblocktests

import (
	"errors"
	"fmt"
	"github.com/opacey/dmgd/blockchain"
	"github.com/opacey/dmgd/btcec"
	"github.com/opacey/dmgd/chaincfg"
	"github.com/opacey/dmgd/chaincfg/chainhash"
	"github.com/opacey/dmgd/provautil"
	"github.com/opacey/dmgd/txscript"
	"github.com/opacey/dmgd/wire"
	"math"
	"math/rand"
	"runtime"
	"time"
)

var (
	// Some keys to make tests easier.
	privKey1, _ = btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0x2b, 0x8c, 0x52, 0xb7, 0x7b, 0x32, 0x7c, 0x75,
		0x5b, 0x9b, 0x37, 0x55, 0x00, 0xd3, 0xf4, 0xb2,
		0xda, 0x9b, 0x0a, 0x1f, 0xf6, 0x5f, 0x68, 0x91,
		0xd3, 0x11, 0xfe, 0x94, 0x29, 0x5b, 0xc2, 0x6a,
	})
	pubKey1     = (*btcec.PublicKey)(&privKey1.PublicKey)
	privKey2, _ = btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0xea, 0xf0, 0x2c, 0xa3, 0x48, 0xc5, 0x24, 0xe6,
		0x39, 0x26, 0x55, 0xba, 0x4d, 0x29, 0x60, 0x3c,
		0xd1, 0xa7, 0x34, 0x7d, 0x9d, 0x65, 0xcf, 0xe9,
		0x3c, 0xe1, 0xeb, 0xff, 0xdc, 0xa2, 0x26, 0x94,
	})
	pubKey2     = (*btcec.PublicKey)(&privKey2.PublicKey)
	privKey3, _ = btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0x64, 0x89, 0xdd, 0x3e, 0x30, 0x88, 0xc2, 0xc4,
		0xd6, 0xbc, 0x44, 0x4e, 0x4c, 0x47, 0xf9, 0x2c,
		0x9b, 0xf2, 0x8d, 0x89, 0x65, 0x1a, 0x9e, 0x22,
		0x0d, 0xbc, 0x2c, 0x0d, 0x11, 0x81, 0xc5, 0xe4,
	})
	pubKey3 = (*btcec.PublicKey)(&privKey3.PublicKey)
	// The validate key must be part of the initial validate key set.
	validatePrivKey, _ = btcec.PrivKeyFromBytes(btcec.S256(), []byte{
		0x40, 0x15, 0x28, 0x9a, 0x22, 0x86, 0x58, 0x04, 0x75, 0x20,
		0xf0, 0xd0, 0xab, 0xe7, 0xad, 0x49, 0xab, 0xc7, 0x7f, 0x6b,
		0xe0, 0xbe, 0x63, 0xb3, 0x6b, 0x94, 0xb8, 0x3c, 0x2d, 0x1f,
		0xd9, 0x77,
	})
	// Some keyIDs to make tests easier
	keyId1 = btcec.KeyID(1)
	keyId2 = btcec.KeyID(2)
	// helper function to sign transactions
	lookupKey = func(a provautil.Address) ([]txscript.PrivateKey, error) {
		return []txscript.PrivateKey{
			txscript.PrivateKey{privKey1, true},
			txscript.PrivateKey{privKey2, true},
		}, nil
	}
)

// TestInstance is an interface that describes a specific test instance returned
// by the tests generated in this package.  It should be type asserted to one
// of the concrete test instance types in order to test accordingly.
type TestInstance interface {
	FullBlockTestInstance()
}

// AcceptedBlock defines a test instance that expects a block to be accepted to
// the blockchain either by extending the main chain, on a side chain, or as an
// orphan.
type AcceptedBlock struct {
	Name         string
	Block        *wire.MsgBlock
	Height       uint32
	IsMainChain  bool
	IsOrphan     bool
	ThreadTips   map[provautil.ThreadID]*wire.OutPoint
	TotalSupply  uint64
	AdminKeySets map[btcec.KeySetType]btcec.PublicKeySet
	ASPKeyIdMap  btcec.KeyIdMap
}

// Ensure AcceptedBlock implements the TestInstance interface.
var _ TestInstance = AcceptedBlock{}

// FullBlockTestInstance only exists to allow AcceptedBlock to be treated as a
// TestInstance.
//
// This implements the TestInstance interface.
func (b AcceptedBlock) FullBlockTestInstance() {}

// RejectedBlock defines a test instance that expects a block to be rejected by
// the blockchain consensus rules.
type RejectedBlock struct {
	Name       string
	Block      *wire.MsgBlock
	Height     uint32
	RejectCode blockchain.ErrorCode
}

// Ensure RejectedBlock implements the TestInstance interface.
var _ TestInstance = RejectedBlock{}

// FullBlockTestInstance only exists to allow RejectedBlock to be treated as a
// TestInstance.
//
// This implements the TestInstance interface.
func (b RejectedBlock) FullBlockTestInstance() {}

// OrphanOrRejectedBlock defines a test instance that expects a block to either
// be accepted as an orphan or rejected.  This is useful since some
// implementations might optimize the immediate rejection of orphan blocks when
// their parent was previously rejected, while others might accept it as an
// orphan that eventually gets flushed (since the parent can never be accepted
// to ultimately link it).
type OrphanOrRejectedBlock struct {
	Name   string
	Block  *wire.MsgBlock
	Height uint32
}

// Ensure ExpectedTip implements the TestInstance interface.
var _ TestInstance = OrphanOrRejectedBlock{}

// FullBlockTestInstance only exists to allow OrphanOrRejectedBlock to be
// treated as a TestInstance.
//
// This implements the TestInstance interface.
func (b OrphanOrRejectedBlock) FullBlockTestInstance() {}

// ExpectedTip defines a test instance that expects a block to be the current
// tip of the main chain.
type ExpectedTip struct {
	Name   string
	Block  *wire.MsgBlock
	Height uint32
}

// Ensure ExpectedTip implements the TestInstance interface.
var _ TestInstance = ExpectedTip{}

// FullBlockTestInstance only exists to allow ExpectedTip to be treated as a
// TestInstance.
//
// This implements the TestInstance interface.
func (b ExpectedTip) FullBlockTestInstance() {}

// RejectedNonCanonicalBlock defines a test instance that expects a serialized
// block that is not canonical and therefore should be rejected.
type RejectedNonCanonicalBlock struct {
	Name     string
	RawBlock []byte
	Height   uint32
}

// FullBlockTestInstance only exists to allow RejectedNonCanonicalBlock to be treated as
// a TestInstance.
//
// This implements the TestInstance interface.
func (b RejectedNonCanonicalBlock) FullBlockTestInstance() {}

// spendableOut represents a transaction output that is spendable along with
// additional metadata such as the block its in and how much it pays.
type spendableOut struct {
	prevOut  wire.OutPoint
	pkScript []byte
	amount   provautil.Amount
}

// makeSpendableOutForTx returns a spendable output for the given transaction
// and transaction output index within the transaction.
func makeSpendableOutForTx(tx *wire.MsgTx, txOutIndex uint32) spendableOut {
	return spendableOut{
		prevOut: wire.OutPoint{
			Hash:  tx.TxHash(),
			Index: txOutIndex,
		},
		pkScript: tx.TxOut[0].PkScript,
		amount:   provautil.Amount(tx.TxOut[txOutIndex].Value),
	}
}

// makeSpendableOut returns a spendable output for the given block, transaction
// index within the block, and transaction output index within the transaction.
func makeSpendableOut(block *wire.MsgBlock, txIndex, txOutIndex uint32) spendableOut {
	return makeSpendableOutForTx(block.Transactions[txIndex], txOutIndex)
}

// testGenerator houses state used to easy the process of generating test blocks
// that build from one another along with housing other useful things such as
// available spendable outputs used throughout the tests.
type testGenerator struct {
	params       *chaincfg.Params
	tip          *wire.MsgBlock
	tipName      string
	tipHeight    uint32
	blocks       map[chainhash.Hash]*wire.MsgBlock
	blocksByName map[string]*wire.MsgBlock
	blockHeights map[string]uint32

	// Used for tracking spendable coinbase outputs.
	spendableOuts     []spendableOut
	prevCollectedHash chainhash.Hash

	// Common key for any tests which require signed transactions.
	privKey *btcec.PrivateKey
}

// makeTestGenerator returns a test generator instance initialized with the
// genesis block as the tip.
func makeTestGenerator(params *chaincfg.Params) (testGenerator, error) {
	genesis := params.GenesisBlock
	genesis.Header.Sign(validatePrivKey)
	genesisHash := genesis.Header.BlockHash()
	return testGenerator{
		params:       params,
		blocks:       map[chainhash.Hash]*wire.MsgBlock{genesisHash: genesis},
		blocksByName: map[string]*wire.MsgBlock{"genesis": genesis},
		blockHeights: map[string]uint32{"genesis": 0},
		tip:          genesis,
		tipName:      "genesis",
		tipHeight:    0,
		privKey:      privKey2,
	}, nil
}

// standardCoinbaseScript returns a standard script suitable for use as the
// signature script of the coinbase transaction of a new block.
func standardCoinbaseScript() ([]byte, error) {
	return txscript.NewScriptBuilder().AddData([]byte("/prova/")).
		Script()
}

// provaThreadScript creates a new script to pay a transaction output to an
// Prova Admin Thread.
func provaThreadScript(threadID provautil.ThreadID) []byte {
	builder := txscript.NewScriptBuilder()
	script, err := builder.
		AddInt64(int64(threadID)).
		AddOp(txscript.OP_CHECKTHREAD).Script()
	if err != nil {
		panic(err)
	}
	return script
}

// provaAdminScript creates a new script that executes an admin op.
func provaAdminScript(opcode byte, pubKey *btcec.PublicKey) []byte {
	// size as: <operation (1 byte)> <compressed public key (33 bytes)>
	data := make([]byte, 1+btcec.PubKeyBytesLenCompressed)
	data[0] = opcode
	copy(data[1:], pubKey.SerializeCompressed())
	builder := txscript.NewScriptBuilder()
	script, err := builder.
		AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	if err != nil {
		panic(err)
	}
	return script
}

// provaAdminASPScript creates a new script that executes an admin op
// to provision or deprovision an ASP key.
func provaAdminASPScript(opcode byte, pubKey *btcec.PublicKey, keyID btcec.KeyID) []byte {
	// size as: <operation (1 byte)> <compressed public key (33 bytes)> <key id : 4 bytes>
	data := make([]byte, 1+btcec.PubKeyBytesLenCompressed+btcec.KeyIDSize)
	data[0] = opcode
	copy(data[1:], pubKey.SerializeCompressed())
	keyID.ToAddressFormat(data[1+btcec.PubKeyBytesLenCompressed:])
	builder := txscript.NewScriptBuilder()
	script, err := builder.
		AddOp(txscript.OP_RETURN).
		AddData(data).Script()
	if err != nil {
		panic(err)
	}
	return script
}

// opReturnScript creates an op_return pkScript.
func opReturnScript() []byte {
	return []byte{txscript.OP_RETURN}
}

// createCoinbaseTx returns a coinbase transaction paying an appropriate
// subsidy based on the passed block height.  The coinbase signature script
// conforms to the requirements of version 2 blocks.
func (g *testGenerator) createCoinbaseTx(blockHeight uint32) *wire.MsgTx {
	coinbaseScript, err := standardCoinbaseScript()
	if err != nil {
		panic(err)
	}

	tx := wire.NewMsgTx(1)
	tx.AddTxIn(&wire.TxIn{
		// Coinbase transactions have no inputs, so previous outpoint is
		// zero hash and max index.
		PreviousOutPoint: *wire.NewOutPoint(&chainhash.Hash{},
			wire.MaxPrevOutIndex),
		Sequence:        wire.MaxTxInSequenceNum,
		SignatureScript: coinbaseScript,
	})

	// Create an Prova address that has:
	//   - a random pkHash address, so transaction hashes don't collide
	//   - has keyId1 and keyId2, so it can be spend by always the same
	//      private keys defined for this test suite
	pkHash := make([]byte, 20)
	rand.Read(pkHash)
	addr, _ := provautil.NewAddressProva(pkHash, []btcec.KeyID{keyId1, keyId2}, &chaincfg.RegressionNetParams)
	scriptPkScript, _ := txscript.PayToAddrScript(addr)

	tx.AddTxOut(&wire.TxOut{
		Value:    blockchain.CalcBlockSubsidy(blockHeight, g.params),
		PkScript: scriptPkScript,
	})
	return tx
}

// calcMerkleRoot creates a merkle tree from the slice of transactions and
// returns the root of the tree.
func calcMerkleRoot(txns []*wire.MsgTx) chainhash.Hash {
	if len(txns) == 0 {
		return chainhash.Hash{}
	}

	utilTxns := make([]*provautil.Tx, 0, len(txns))
	for _, tx := range txns {
		utilTxns = append(utilTxns, provautil.NewTx(tx))
	}
	merkles := blockchain.BuildMerkleTreeStore(utilTxns)
	return *merkles[len(merkles)-1]
}

// solveBlock attempts to find a nonce which makes the passed block header hash
// to a value less than the target difficulty.  When a successful solution is
// found true is returned and the nonce field of the passed header is updated
// with the solution.  False is returned if no solution exists.
//
// NOTE: This function will never solve blocks with a nonce of 0.  This is done
// so the 'nextBlock' function can properly detect when a nonce was modified by
// a munge function.
func solveBlock(header *wire.BlockHeader) bool {
	// sbResult is used by the solver goroutines to send results.
	type sbResult struct {
		found bool
		nonce uint32
	}

	// solver accepts a block header and a nonce range to test. It is
	// intended to be run as a goroutine.
	targetDifficulty := blockchain.CompactToBig(header.Bits)
	quit := make(chan bool)
	results := make(chan sbResult)
	solver := func(hdr wire.BlockHeader, startNonce, stopNonce uint32) {
		// We need to modify the nonce field of the header, so make sure
		// we work with a copy of the original header.
		for i := startNonce; i >= startNonce && i <= stopNonce; i++ {
			select {
			case <-quit:
				return
			default:
				hdr.Nonce = uint64(i)
				hash := hdr.BlockHash()
				if blockchain.HashToBig(&hash).Cmp(targetDifficulty) <= 0 {
					results <- sbResult{true, i}
					return
				}
			}
		}
		results <- sbResult{false, 0}
	}

	startNonce := uint32(1)
	stopNonce := uint32(math.MaxUint32)
	numCores := uint32(runtime.NumCPU())
	noncesPerCore := (stopNonce - startNonce) / numCores
	for i := uint32(0); i < numCores; i++ {
		rangeStart := startNonce + (noncesPerCore * i)
		rangeStop := startNonce + (noncesPerCore * (i + 1)) - 1
		if i == numCores-1 {
			rangeStop = stopNonce
		}
		go solver(*header, rangeStart, rangeStop)
	}
	for i := uint32(0); i < numCores; i++ {
		result := <-results
		if result.found {
			close(quit)
			header.Nonce = uint64(result.nonce)
			return true
		}
	}

	return false
}

// additionalTx returns a function that itself takes a block and modifies it by
// adding the the provided transaction.
func additionalTx(tx *wire.MsgTx) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		b.AddTransaction(tx)
	}
}

// changeCoinbaseValue returns a function that itself takes a block and changes
// it to alter the claim of the coinbase reward.
func changeCoinbaseValue(delta int64) func(*wire.MsgBlock) {
	return func(b *wire.MsgBlock) {
		txns := b.Transactions
		coinbaseTx := txns[0]
		coinbaseTx.TxOut[0].Value += delta
	}
}

func makeAddr(priv *btcec.PrivateKey, kids *[2]uint) provautil.Address {
	// Create an Prova address that has:
	//   - a random pkHash address, so transaction hashes don't collide
	//   - has keyId1 and keyId2, so it can be spend by always the same
	//      private keys defined for this test suite
	pkHash := make([]byte, 20)
	rand.Read(pkHash)
	if priv != nil {
		pub := (*btcec.PublicKey)(&priv.PublicKey)
		pkHash = provautil.Hash160(pub.SerializeCompressed())
	}
	keyIDs := []btcec.KeyID{keyId1, keyId2}
	if kids != nil {
		keyIDs[0] = btcec.KeyID(kids[0])
		keyIDs[1] = btcec.KeyID(kids[1])
	}
	addr, _ := provautil.NewAddressProva(pkHash, keyIDs, &chaincfg.RegressionNetParams)
	return addr
}

type ProvaOut struct {
	amount provautil.Amount
	addr   string
}

func createProvaSpendTx(spend *spendableOut, outs []ProvaOut, priv *btcec.PrivateKey) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)

	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: spend.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})

	for _, out := range outs {
		addr, _ := provautil.DecodeAddress(out.addr, &chaincfg.RegressionNetParams)
		scriptPkScript, _ := txscript.PayToAddrScript(addr)
		spendTx.AddTxOut(wire.NewTxOut(int64(out.amount), scriptPkScript))
	}

	privKeys := []txscript.PrivateKey{
		txscript.PrivateKey{privKey1, true},
		txscript.PrivateKey{privKey2, true},
	}
	if priv != nil {
		privKeys[1] = txscript.PrivateKey{priv, true}
	}
	lookupKeyFunc := func(a provautil.Address) ([]txscript.PrivateKey, error) {
		return privKeys, nil
	}

	// Use Account Service Key and Account Recovery Key to sign tx.
	sigScript, _ := txscript.SignTxOutput(&chaincfg.RegressionNetParams, spendTx,
		0, int64(spend.amount), spend.pkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKeyFunc), nil)

	spendTx.TxIn[0].SignatureScript = sigScript

	return spendTx
}

// createSpendTx creates a transaction that spends from the provided spendable
// output and includes an additional unique OP_RETURN output to ensure the
// transaction ends up with a unique hash.  The script is a simple OP_TRUE
// script which avoids the need to track addresses and signature scripts in the
// tests.
func createSpendTx(spend *spendableOut, fee provautil.Amount) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)

	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: spend.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})

	scriptPkScript, _ := txscript.PayToAddrScript(makeAddr(nil, nil))
	spendTx.AddTxOut(wire.NewTxOut(int64(spend.amount-fee), scriptPkScript))

	// Use Account Service Key and Account Recovery Key to sign tx.
	sigScript, _ := txscript.SignTxOutput(&chaincfg.RegressionNetParams, spendTx,
		0, int64(spend.amount), spend.pkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)

	spendTx.TxIn[0].SignatureScript = sigScript

	return spendTx
}

// createAdminTx creates an admin tx.
func createAdminTx(spend *spendableOut, threadID provautil.ThreadID, op byte, pubKey *btcec.PublicKey) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: spend.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})
	txValue := int64(0) // how much the tx is spending. 0 for admin tx.
	spendTx.AddTxOut(wire.NewTxOut(txValue, provaThreadScript(threadID)))
	spendTx.AddTxOut(wire.NewTxOut(txValue,
		provaAdminScript(op, pubKey)))

	sigScript, _ := txscript.SignTxOutput(&chaincfg.RegressionNetParams, spendTx,
		0, int64(spend.amount), spend.pkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)

	spendTx.TxIn[0].SignatureScript = sigScript

	return spendTx
}

type AspOp struct {
	Op     byte
	PubKey *btcec.PublicKey
	KeyID  btcec.KeyID
}

// createASPAdminTx creates an admin tx that provisions a keyID
func createASPAdminTx(spend *spendableOut, ops []AspOp) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: spend.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})
	txValue := int64(0) // how much the tx is spending. 0 for admin tx.
	spendTx.AddTxOut(wire.NewTxOut(txValue,
		provaThreadScript(provautil.ProvisionThread)))
	for _, op := range ops {
		spendTx.AddTxOut(wire.NewTxOut(txValue,
			provaAdminASPScript(op.Op, op.PubKey, op.KeyID)))
	}

	sigScript, _ := txscript.SignTxOutput(&chaincfg.RegressionNetParams, spendTx,
		0, int64(spend.amount), spend.pkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)

	spendTx.TxIn[0].SignatureScript = sigScript

	return spendTx
}

// createIssueTx creates an issue thread admin tx.
// If a spend output is passed, a revoke transaction is build.
// if spend is nil, new tokens of amount in value are issued.
func createIssueTx(thread *spendableOut, value int64, spend *spendableOut) *wire.MsgTx {
	spendTx := wire.NewMsgTx(1)
	// thread input
	spendTx.AddTxIn(&wire.TxIn{
		PreviousOutPoint: thread.prevOut,
		Sequence:         wire.MaxTxInSequenceNum,
		SignatureScript:  nil,
	})
	// thread output
	spendTx.AddTxOut(wire.NewTxOut(int64(0), provaThreadScript(provautil.IssueThread)))
	if spend == nil {
		scriptPkScript, _ := txscript.PayToAddrScript(makeAddr(nil, nil))
		spendTx.AddTxOut(wire.NewTxOut(value, scriptPkScript))
	} else {
		// destroy some tokens:
		// - spend output of amount x
		// - bind amount x in opReturn output
		spendTx.AddTxIn(&wire.TxIn{
			PreviousOutPoint: spend.prevOut,
			Sequence:         wire.MaxTxInSequenceNum,
			SignatureScript:  nil,
		})
		spendTx.AddTxOut(wire.NewTxOut(
			int64(spend.amount),
			opReturnScript(),
		))
	}
	// sign thread input
	sigScript, _ := txscript.SignTxOutput(&chaincfg.RegressionNetParams, spendTx,
		0, int64(thread.amount), thread.pkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)
	spendTx.TxIn[0].SignatureScript = sigScript
	if spend != nil {
		// sign second input
		sigScript2, _ := txscript.SignTxOutput(&chaincfg.RegressionNetParams, spendTx,
			1, int64(spend.amount), spend.pkScript, txscript.SigHashAll, txscript.KeyClosure(lookupKey), nil)
		spendTx.TxIn[1].SignatureScript = sigScript2
	}
	return spendTx
}

// nextBlock builds a new block that extends the current tip associated with the
// generator and updates the generator's tip to the newly generated block.
//
// The block will include the following:
// - A coinbase that pays the required subsidy to an Prova script
// - When a spendable output is provided:
//   - A transaction that spends from the provided output to a new Prova script
//
// Additionally, if one or more munge functions are specified, they will be
// invoked with the block prior to solving it.  This provides callers with the
// opportunity to modify the block which is especially useful for testing.
//
// In order to simply the logic in the munge functions, the following rules are
// applied after all munge functions have been invoked:
// - The merkle root will be recalculated unless it was manually changed
// - The block will be solved unless the nonce was changed
func (g *testGenerator) nextBlock(blockName string, spend *spendableOut, mungers ...func(*wire.MsgBlock)) *wire.MsgBlock {
	// Create coinbase transaction for the block using any additional
	// subsidy if specified.
	nextHeight := g.tipHeight + 1
	coinbaseTx := g.createCoinbaseTx(nextHeight)
	txns := []*wire.MsgTx{coinbaseTx}
	if spend != nil {
		// Create a transaction that spends from the provided spendable
		// output, then add it to the list of transactions to include in the
		// block.
		txns = append(txns, createSpendTx(spend, 0))
	}

	// Use a timestamp that is two minutes after the previous block unless
	// this is the first block in which case the current time is used.
	var ts time.Time
	if nextHeight == 1 {
		ts = time.Unix(time.Now().Unix(), 0)
	} else {
		ts = g.tip.Header.Timestamp.Add(time.Minute * 2)
	}

	block := wire.MsgBlock{
		Header: wire.BlockHeader{
			Version:    1,
			PrevBlock:  g.tip.BlockHash(),
			MerkleRoot: calcMerkleRoot(txns),
			Bits:       g.params.PowLimitBits,
			Timestamp:  ts,
			Height:     nextHeight,
			Nonce:      0, // To be solved.
		},
		Transactions: txns,
	}

	// Perform any block munging just before solving.  Only recalculate the
	// merkle root if it wasn't manually changed by a munge function.
	curMerkleRoot := block.Header.MerkleRoot
	curNonce := block.Header.Nonce
	for _, f := range mungers {
		f(&block)
	}
	if block.Header.MerkleRoot == curMerkleRoot {
		block.Header.MerkleRoot = calcMerkleRoot(block.Transactions)
	}
	block.Header.Size = uint32(block.SerializeSize())
	block.Header.Sign(validatePrivKey)

	// Only solve the block if the nonce wasn't manually changed by a munge
	// function.
	if block.Header.Nonce == curNonce && !solveBlock(&block.Header) {
		panic(fmt.Sprintf("Unable to solve block at height %d",
			nextHeight))
	}
	// Update generator state and return the block.
	blockHash := block.BlockHash()
	g.blocks[blockHash] = &block
	g.blocksByName[blockName] = &block
	g.blockHeights[blockName] = nextHeight
	g.tip = &block
	g.tipName = blockName
	g.tipHeight = nextHeight
	return &block
}

// setTip changes the tip of the instance to the block with the provided name.
// This is useful since the tip is used for things such as generating subsequent
// blocks.
func (g *testGenerator) setTip(blockName string) {
	g.tip = g.blocksByName[blockName]
	g.tipName = blockName
	g.tipHeight = g.blockHeights[blockName]
}

// oldestCoinbaseOuts removes the oldest coinbase output that was previously
// saved to the generator and returns the set as a slice.
func (g *testGenerator) oldestCoinbaseOut() spendableOut {
	op := g.spendableOuts[0]
	g.spendableOuts = g.spendableOuts[1:]
	return op
}

// saveTipCoinbaseOut adds the coinbase tx output in the current tip block to
// the list of spendable outputs.
func (g *testGenerator) saveTipCoinbaseOut() {
	g.spendableOuts = append(g.spendableOuts, makeSpendableOut(g.tip, 0, 0))
	g.prevCollectedHash = g.tip.BlockHash()
}

// Generate returns a slice of tests that can be used to exercise the consensus
// validation rules.  The tests are intended to be flexible enough to allow both
// unit-style tests directly against the blockchain code as well as integration
// style tests over the peer-to-peer network.  To achieve that goal, each test
// contains additional information about the expected result, however that
// information can be ignored when doing comparison tests between two
// independent versions over the peer-to-peer network.
func Generate(includeLargeReorg bool) (tests [][]TestInstance, err error) {
	// In order to simplify the generation code which really should never
	// fail unless the test code itself is broken, panics are used
	// internally.  This deferred func ensures any panics don't escape the
	// generator by replacing the named error return with the underlying
	// panic error.
	defer func() {
		if r := recover(); r != nil {
			tests = nil

			switch rt := r.(type) {
			case string:
				err = errors.New(rt)
			case error:
				err = rt
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()

	// Create a test generator instance initialized with the genesis block
	// as the tip.
	g, err := makeTestGenerator(&chaincfg.RegressionNetParams)
	if err != nil {
		return nil, err
	}

	// Define some convenience helper functions to return an individual test
	// instance that has the described characteristics.
	//
	// acceptBlock creates a test instance that expects the provided block
	// to be accepted by the consensus rules.
	//
	// rejectBlock creates a test instance that expects the provided block
	// to be rejected by the consensus rules.
	//
	// expectTipBlock creates a test instance that expects the provided
	// block to be the current tip of the block chain.
	lastAdminKeySets := chaincfg.RegressionNetParams.AdminKeySets
	lastASPKeys := chaincfg.RegressionNetParams.ASPKeyIdMap
	lastTotalSupply := uint64(0)
	lastThreadTips := make(map[provautil.ThreadID]*wire.OutPoint)
	rootOut := makeSpendableOut(g.tip, 0, 0)
	lastThreadTips[provautil.RootThread] = &rootOut.prevOut
	provisionOut := makeSpendableOut(g.tip, 0, 1)
	lastThreadTips[provautil.ProvisionThread] = &provisionOut.prevOut
	issueOut := makeSpendableOut(g.tip, 0, 2)
	lastThreadTips[provautil.IssueThread] = &issueOut.prevOut

	acceptBlock := func(blockName string, block *wire.MsgBlock, isMainChain, isOrphan bool) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return AcceptedBlock{blockName, block, blockHeight, isMainChain, isOrphan, lastThreadTips, lastTotalSupply, lastAdminKeySets, lastASPKeys}
	}
	rejectBlock := func(blockName string, block *wire.MsgBlock, code blockchain.ErrorCode) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return RejectedBlock{blockName, block, blockHeight, code}
	}
	expectTipBlock := func(blockName string, block *wire.MsgBlock) TestInstance {
		blockHeight := g.blockHeights[blockName]
		return ExpectedTip{blockName, block, blockHeight}
	}

	// Define some convenience helper functions to populate the tests slice
	// with test instances that have the described characteristics.
	//
	// accepted creates and appends a single acceptBlock test instance for
	// the current tip which expects the block to be accepted to the main
	// chain.
	//
	// acceptedToSideChainWithExpectedTip creates an appends a two-instance
	// test.  The first instance is an acceptBlock test instance for the
	// current tip which expects the block to be accepted to a side chain.
	// The second instance is an expectBlockTip test instance for provided
	// values.
	//
	// rejected creates and appends a single rejectBlock test instance for
	// the current tip.
	accepted := func() {
		tests = append(tests, []TestInstance{
			acceptBlock(g.tipName, g.tip, true, false),
		})
	}
	assertThreadTip := func(threadId provautil.ThreadID, out spendableOut) {
		threadTips := provautil.CopyThreadTips(lastThreadTips)
		threadTips[threadId] = &out.prevOut
		lastThreadTips = threadTips
	}
	assertTotalSupply := func(totalSupply uint64) {
		lastTotalSupply = totalSupply
	}
	assertAdminKeys := func(keySetType btcec.KeySetType, adminKeys []btcec.PublicKey) {
		adminKeySets := btcec.DeepCopy(lastAdminKeySets)
		if adminKeys != nil {
			adminKeySets[keySetType] = adminKeys
		}
		lastAdminKeySets = adminKeySets
	}
	assertASPKey := func(adminKey *btcec.PublicKey, keyID btcec.KeyID) {
		aspKeys := lastASPKeys.DeepCopy()
		if aspKeys != nil {
			aspKeys[keyID] = adminKey
		}
		lastASPKeys = aspKeys
	}
	assertNotASPKey := func(adminKey *btcec.PublicKey, keyID btcec.KeyID) {
		aspKeys := lastASPKeys.DeepCopy()
		if aspKeys != nil {
			delete(aspKeys, keyID)
		}
		lastASPKeys = aspKeys
	}
	acceptedToSideChainWithExpectedTip := func(tipName string) {
		tests = append(tests, []TestInstance{
			acceptBlock(g.tipName, g.tip, false, false),
			expectTipBlock(tipName, g.blocksByName[tipName]),
		})
	}
	rejected := func(code blockchain.ErrorCode) {
		tests = append(tests, []TestInstance{
			rejectBlock(g.tipName, g.tip, code),
		})
	}

	// Get the thread tips from genesis
	var outs []*spendableOut
	// start of ROOT THREAD
	outs = append(outs, &rootOut)
	// start of PROVISION THREAD
	outs = append(outs, &provisionOut)
	// start of ISSUE THREAD
	outs = append(outs, &issueOut)

	// ---------------------------------------------------------------------
	// Generate enough blocks to have mature coinbase outputs to work with.
	//
	//   genesis -> bm0 -> bm1 -> ... -> bm99
	// ---------------------------------------------------------------------

	coinbaseMaturity := g.params.CoinbaseMaturity
	var testInstances []TestInstance

	for i := uint16(0); i < coinbaseMaturity; i++ {
		blockName := fmt.Sprintf("bm%d", i)

		g.nextBlock(blockName, nil)
		g.saveTipCoinbaseOut()
		testInstances = append(testInstances, acceptBlock(g.tipName,
			g.tip, true, false))
	}
	tests = append(tests, testInstances)

	// Collect spendable outputs.  This simplifies the code below.
	for i := uint16(0); i < coinbaseMaturity; i++ {
		op := g.oldestCoinbaseOut()
		outs = append(outs, &op)
	}

	// Initial admin set keys
	initialIssuePubKey1 := lastAdminKeySets[btcec.IssueKeySet][0]
	initialIssuePubKey2 := lastAdminKeySets[btcec.IssueKeySet][1]
	initialProvisionPubKey1 := lastAdminKeySets[btcec.ProvisionKeySet][0]
	initialProvisionPubKey2 := lastAdminKeySets[btcec.ProvisionKeySet][1]

	// ---------------------------------------------------------------------
	// The comments below identify the structure of the chain being built.
	//
	// The values in parenthesis repesent which outputs are being spent.
	//
	// For example, b1(0) indicates the first collected spendable output
	// which, due to the code above to create the correct number of blocks,
	// is the first output that can be spent at the current block height due
	// to the coinbase maturity requirement.
	// ---------------------------------------------------------------------
	// Start by building a couple of blocks at current tip.
	//
	//    ... -> b1(3) -> b3() -> b4() -> b5() -> b6() -> b7() -> b8(7)
	//
	// ---------------------------------------------------------------------

	g.nextBlock("b1", outs[3])
	assertTotalSupply(0)
	accepted()

	// Try to spend provision thread with root thread
	issueKeyAddTx := createAdminTx(outs[1], 0, txscript.AdminOpIssueKeyAdd, pubKey1)
	g.nextBlock("b2", nil, additionalTx(issueKeyAddTx))
	rejected(blockchain.ErrInvalidAdminTx)

	// Provision an ISSUE key in b3 and check its there.
	g.setTip("b1")
	issueKeyAddTx = createAdminTx(outs[0], 0, txscript.AdminOpIssueKeyAdd, pubKey1)
	rootThreadOut := makeSpendableOutForTx(issueKeyAddTx, 0)
	g.nextBlock("b3", nil, additionalTx(issueKeyAddTx))
	assertThreadTip(provautil.RootThread, rootThreadOut)
	assertAdminKeys(btcec.IssueKeySet, []btcec.PublicKey{*pubKey1, initialIssuePubKey1, initialIssuePubKey2})
	accepted()

	// Provision another two ISSUE keys and check three are there.
	issueKeyAddTx2 := createAdminTx(&rootThreadOut, 0, txscript.AdminOpIssueKeyAdd, pubKey2)
	rootThreadOut = makeSpendableOutForTx(issueKeyAddTx2, 0)
	issueKeyAddTx3 := createAdminTx(&rootThreadOut, 0, txscript.AdminOpIssueKeyAdd, pubKey3)
	rootThreadOut = makeSpendableOutForTx(issueKeyAddTx3, 0)
	g.nextBlock("b4", nil, additionalTx(issueKeyAddTx2), additionalTx(issueKeyAddTx3))
	assertThreadTip(provautil.RootThread, rootThreadOut)
	assertAdminKeys(btcec.IssueKeySet, []btcec.PublicKey{*pubKey1, *pubKey2, *pubKey3, initialIssuePubKey1, initialIssuePubKey2})
	accepted()

	// Issue some tokens
	issueTx := createIssueTx(outs[2], int64(8000000000), nil)
	g.nextBlock("b5", nil, additionalTx(issueTx))
	assertTotalSupply(8000000000)
	accepted()

	// ---------------------------------------------------------------------
	// Reorg tests for reverting issuance and destruction
	// ---------------------------------------------------------------------
	//
	//
	//   ... -> b5(+8) -> b7() -> b8() -> b9()
	//                \-> b6(-8 +4)

	// Destroy all issued tokens, create some others
	issueThreadOut := makeSpendableOutForTx(issueTx, 0)
	coinsToRevoke := makeSpendableOutForTx(issueTx, 1)
	revokeTx := createIssueTx(&issueThreadOut, 0, &coinsToRevoke)
	issueThreadOut = makeSpendableOutForTx(revokeTx, 0)
	issueTxOnFork := createIssueTx(&issueThreadOut, int64(4000000000), nil)
	g.nextBlock("b6", nil, additionalTx(revokeTx), additionalTx(issueTxOnFork))
	assertTotalSupply(4000000000)
	accepted()

	g.setTip("b5")

	g.nextBlock("b7", nil)
	acceptedToSideChainWithExpectedTip("b6")

	// reorg, things are back to normal
	g.nextBlock("b8", nil)
	assertTotalSupply(8000000000)
	accepted()

	// TODO(prova): revoke with change
	g.nextBlock("b9", nil)
	accepted()

	// Revoke one ISSUE key again
	issueKeyRevokeTx1 := createAdminTx(&rootThreadOut, 0, txscript.AdminOpIssueKeyRevoke, pubKey1)
	rootThreadOut = makeSpendableOutForTx(issueKeyRevokeTx1, 0)
	g.nextBlock("b10", nil, additionalTx(issueKeyRevokeTx1))
	assertThreadTip(provautil.RootThread, rootThreadOut)
	assertAdminKeys(btcec.IssueKeySet, []btcec.PublicKey{*pubKey3, *pubKey2, initialIssuePubKey1, initialIssuePubKey2})
	accepted()

	// add provision keys
	provisionKeyAddTx1 := createAdminTx(&rootThreadOut, 0, txscript.AdminOpProvisionKeyAdd, pubKey1)
	rootThreadOut = makeSpendableOutForTx(provisionKeyAddTx1, 0)
	provisionKeyAddTx2 := createAdminTx(&rootThreadOut, 0, txscript.AdminOpProvisionKeyAdd, pubKey2)
	rootThreadOut = makeSpendableOutForTx(provisionKeyAddTx2, 0)
	g.nextBlock("b11", nil, additionalTx(provisionKeyAddTx1), additionalTx(provisionKeyAddTx2))
	assertThreadTip(provautil.RootThread, rootThreadOut)
	assertAdminKeys(btcec.ProvisionKeySet, []btcec.PublicKey{*pubKey1, *pubKey2, initialProvisionPubKey1, initialProvisionPubKey2})
	accepted()

	// provision a keyID and check
	aspKeyIdTx := createASPAdminTx(outs[1], []AspOp{{txscript.AdminOpASPKeyAdd, pubKey1, btcec.KeyID(3)}})
	g.nextBlock("b12", nil, additionalTx(aspKeyIdTx))
	assertASPKey(pubKey1, btcec.KeyID(3))
	accepted()

	// ---------------------------------------------------------------------
	// Reorg tests for keyID provisioning
	// ---------------------------------------------------------------------
	//
	//   ... -> b12(+3) -> b14() -> b15() -> b16(+4)
	//                 \-> b13(-3 +4)

	provThreadOut := makeSpendableOutForTx(aspKeyIdTx, 0)
	aspKeyIdTx1 := createASPAdminTx(&provThreadOut, []AspOp{{txscript.AdminOpASPKeyRevoke, pubKey1, btcec.KeyID(3)}})
	provThreadOut = makeSpendableOutForTx(aspKeyIdTx1, 0)
	aspKeyIdTx2 := createASPAdminTx(&provThreadOut, []AspOp{{txscript.AdminOpASPKeyAdd, pubKey2, btcec.KeyID(4)}})
	g.nextBlock("b13", nil, additionalTx(aspKeyIdTx1), additionalTx(aspKeyIdTx2))
	assertNotASPKey(pubKey1, btcec.KeyID(3))
	assertASPKey(pubKey2, btcec.KeyID(4))
	accepted()

	g.setTip("b12")
	g.nextBlock("b14", nil)
	acceptedToSideChainWithExpectedTip("b13")

	// reorg, things are back to normal
	g.nextBlock("b15", nil)
	assertASPKey(pubKey1, btcec.KeyID(3))
	assertNotASPKey(pubKey2, btcec.KeyID(4))
	accepted()

	// provision keyID 4, make sure it is not market as used
	provThreadOut = makeSpendableOutForTx(aspKeyIdTx, 0)
	aspKeyIdTx = createASPAdminTx(&provThreadOut, []AspOp{
		{txscript.AdminOpASPKeyAdd, pubKey2, btcec.KeyID(4)},
	})
	g.nextBlock("b16", nil, additionalTx(aspKeyIdTx))
	assertASPKey(pubKey2, btcec.KeyID(4))
	accepted()

	// try to provision same keyID 2 times in one tx
	provThreadOut = makeSpendableOutForTx(aspKeyIdTx, 0)
	invalidAspKeyIdTx := createASPAdminTx(&provThreadOut, []AspOp{
		{txscript.AdminOpASPKeyAdd, pubKey2, btcec.KeyID(5)},
		{txscript.AdminOpASPKeyAdd, pubKey1, btcec.KeyID(5)},
	})
	g.nextBlock("b17", nil, additionalTx(invalidAspKeyIdTx))
	rejected(blockchain.ErrInvalidAdminOp)

	// provision 2 consecutive keyIDs in one tx
	g.setTip("b16")
	provThreadOut = makeSpendableOutForTx(aspKeyIdTx, 0)
	aspKeyIdTx2 = createASPAdminTx(&provThreadOut, []AspOp{
		{txscript.AdminOpASPKeyAdd, pubKey2, btcec.KeyID(5)},
		{txscript.AdminOpASPKeyAdd, pubKey1, btcec.KeyID(6)},
	})
	g.nextBlock("b18", nil, additionalTx(aspKeyIdTx2))
	assertASPKey(pubKey2, btcec.KeyID(5))
	assertASPKey(pubKey1, btcec.KeyID(6))
	accepted()

	// spend some coins to keyID 4
	coinsToSpend := makeSpendableOutForTx(issueTx, 1)
	spendTx := createProvaSpendTx(&coinsToSpend, []ProvaOut{
		{4000000000, makeAddr(privKey3, &[2]uint{2, 4}).String()},
		{4000000000, makeAddr(nil, nil).String()},
	}, nil)
	g.nextBlock("b19", nil, additionalTx(spendTx))
	accepted()

	// revoke keyID 4
	provThreadOut = makeSpendableOutForTx(aspKeyIdTx2, 0)
	aspKeyIdTx = createASPAdminTx(&provThreadOut, []AspOp{{txscript.AdminOpASPKeyRevoke, pubKey2, btcec.KeyID(4)}})
	g.nextBlock("b20", nil, additionalTx(aspKeyIdTx))
	assertNotASPKey(pubKey2, btcec.KeyID(4))
	accepted()

	// send coins with recovery key
	coinsToSpend = makeSpendableOutForTx(spendTx, 0)
	reSpendTx := createProvaSpendTx(&coinsToSpend, []ProvaOut{
		{4000000000, makeAddr(nil, nil).String()},
	}, privKey3)
	g.nextBlock("b21", nil, additionalTx(reSpendTx))
	accepted()

	// ---------------------------------------------------------------------
	// Basic forking and reorg tests.
	// ---------------------------------------------------------------------
	//
	//   ... -> b22(8) -> b23()
	//
	// A new key will be provisioned in b23, then the operation will be
	// reorged away.

	g.nextBlock("b22", outs[8])
	accepted()

	adminKeyAddTx := createAdminTx(&rootThreadOut, 0, txscript.AdminOpIssueKeyAdd, pubKey1)
	rootThreadOutFork := makeSpendableOutForTx(adminKeyAddTx, 0)
	g.nextBlock("b23", nil, additionalTx(adminKeyAddTx))
	assertThreadTip(provautil.RootThread, rootThreadOutFork)
	assertAdminKeys(btcec.IssueKeySet, []btcec.PublicKey{*pubKey3, *pubKey2, *pubKey1, initialIssuePubKey1, initialIssuePubKey2})
	accepted()

	// Create a fork from b20.  There should not be a reorg since b10 was seen
	// first.
	//
	//   ... -> b22(8) -> b23(9)
	//                \-> b24(9)
	g.setTip("b22")
	g.nextBlock("b24", outs[9])
	// blocks on sidechains are not validated for utxos or keysets yet
	acceptedToSideChainWithExpectedTip("b23")

	// Extend b24 fork to make the alternative chain longer and force reorg.
	//
	//   ... -> b22(8) -> b23(9)
	//                \-> b24(9) -> b25(10)
	//
	// The reorg should revent the provisioning of an ISSUE key in b23.
	g.nextBlock("b25", outs[10])
	assertThreadTip(provautil.RootThread, rootThreadOut)
	assertAdminKeys(btcec.IssueKeySet, []btcec.PublicKey{*pubKey3, *pubKey2, initialIssuePubKey1, initialIssuePubKey2}) // The genesis admin state is valid.
	accepted()

	// Extend b23 fork twice to make first chain longer and force reorg.
	//
	//   ... -> b22(8) -> b23(9) -> b26(10) -> b27(11)
	//                \-> b24(9) -> b25(10)
	//
	// key provisioned in b10 will be back in admin set.
	//
	g.setTip("b23")
	g.nextBlock("b26", outs[10])
	// blocks for sidechains don't validate utxos or keysets yet
	acceptedToSideChainWithExpectedTip("b25")

	// key is active again.
	g.nextBlock("b27", outs[11])
	assertThreadTip(provautil.RootThread, rootThreadOutFork)
	assertAdminKeys(btcec.IssueKeySet, []btcec.PublicKey{*pubKey3, *pubKey2, *pubKey1, initialIssuePubKey1, initialIssuePubKey2})
	accepted()

	// ---------------------------------------------------------------------
	// Double spend tests.
	// ---------------------------------------------------------------------

	// Create a fork that double spends.
	//
	//   ... -> b22(8) -> b23(9) -> b26(10) -> b27(11)
	//                                    \-> b28(10) -> b29(12)
	//                \-> b24(9) -> b25(10)
	//
	g.setTip("b26")
	g.nextBlock("b28", outs[10])
	// blocks on sidechains are not validated for utxos or keysets yet
	acceptedToSideChainWithExpectedTip("b27")

	g.nextBlock("b29", outs[12])
	rejected(blockchain.ErrMissingTx) // now doublespend recognized.

	// ---------------------------------------------------------------------
	// Coinbase reward tests.
	// ---------------------------------------------------------------------

	// Attempt to progress the chain past b27 with bad coinbase fee blocks.
	g.setTip("b27")
	issuedCoinsSpend := makeSpendableOutForTx(spendTx, 1)
	createSpendTx := createSpendTx(&issuedCoinsSpend, 1) // Fee: 1
	g.nextBlock("b30", outs[12], additionalTx(createSpendTx), changeCoinbaseValue(0))
	rejected(blockchain.ErrBadCoinbaseValue)

	g.setTip("b27")
	g.nextBlock("b31", outs[12], changeCoinbaseValue(1))
	rejected(blockchain.ErrBadCoinbaseValue)

	return tests, nil
}
